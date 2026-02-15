package views

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// LogsLoadedMsg is sent when logs are loaded (for initial load)
type LogsLoadedMsg struct {
	Logs string
	Err  error
}

// LogLineMsg is sent when a new log line is received during streaming
type LogLineMsg struct {
	Line string
	Gen  uint64
}

// LogStreamEndedMsg is sent when the log stream ends
type LogStreamEndedMsg struct {
	Err error
	Gen uint64
}

// LogsSavedMsg is sent when logs are saved to file
type LogsSavedMsg struct {
	Path string
	Err  error
}

// timeRange defines a predefined time range for log filtering
type timeRange struct {
	Label   string
	Seconds int64
}

var defaultTimeRanges = []timeRange{
	{Label: "5m", Seconds: 300},
	{Label: "15m", Seconds: 900},
	{Label: "1h", Seconds: 3600},
	{Label: "6h", Seconds: 21600},
	{Label: "24h", Seconds: 86400},
	{Label: "all", Seconds: 0},
}

// LogsView displays logs for a pod with streaming support
type LogsView struct {
	BaseView
	viewport  viewport.Model
	client    k8s.Client
	pod       string
	namespace string
	container string
	logs      strings.Builder
	logLines  []string
	loading   bool
	err       error
	tailLines int64
	spinner   *components.Spinner

	// Streaming state
	streaming    bool
	streamCancel context.CancelFunc
	logChan      chan string
	streamGen    uint64

	// Search
	searchPattern string
	searchRegex   *regexp.Regexp
	searchMatches []int
	searchCursor  int

	// Timestamps
	showTimestamps bool

	// Previous container logs
	showPrevious bool

	// Time range
	timeRangeIdx int

	// Text wrap
	wrapText bool
}

// NewLogsView creates a new logs view
func NewLogsView(client k8s.Client) *LogsView {
	vp := viewport.New(80, 20)
	vp.Style = theme.Styles.Base

	return &LogsView{
		viewport:     vp,
		client:       client,
		tailLines:    100,
		spinner:      components.NewSpinner(),
		searchCursor: -1,
		timeRangeIdx: 5, // "all" — matches current default behavior
	}
}

// SetPod sets the pod to view logs for
func (v *LogsView) SetPod(namespace, pod, container string) {
	v.namespace = namespace
	v.pod = pod
	v.container = container
}

// SetClient sets a new k8s client
func (v *LogsView) SetClient(client k8s.Client) {
	v.client = client
}

// Init initializes the view
func (v *LogsView) Init() tea.Cmd {
	if v.pod == "" {
		return nil
	}
	return v.Refresh()
}

// Update handles messages
func (v *LogsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case LogsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.logs.Reset()
			v.logs.WriteString(msg.Logs)
			v.logLines = strings.Split(msg.Logs, "\n")
			v.updateViewportContent()

			// Auto-start streaming unless showing previous logs
			if !v.showPrevious && v.pod != "" {
				cmds = append(cmds, v.startStreaming())
			}
		}

	case LogLineMsg:
		if msg.Gen != v.streamGen {
			break // stale message from old stream
		}
		// Append new log line
		if v.logs.Len() > 0 {
			v.logs.WriteString("\n")
		}
		v.logs.WriteString(msg.Line)
		v.logLines = append(v.logLines, msg.Line)

		// Limit buffer size (keep last 10000 lines)
		if len(v.logLines) > 10000 {
			v.logLines = v.logLines[len(v.logLines)-10000:]
			v.logs.Reset()
			v.logs.WriteString(strings.Join(v.logLines, "\n"))
		}

		// If search is active, check new line for matches
		if v.searchRegex != nil {
			lineIdx := len(v.logLines) - 1
			if v.searchRegex.MatchString(v.logLines[lineIdx]) {
				v.searchMatches = append(v.searchMatches, lineIdx)
			}
		}

		v.updateViewportContent()
		cmds = append(cmds, v.waitForLogLine(v.logChan, v.streamGen))

	case LogStreamEndedMsg:
		if msg.Gen != v.streamGen {
			break // stale message from old stream
		}
		v.streaming = false
		v.spinner.Hide()
		if msg.Err != nil && msg.Err != io.EOF && msg.Err != context.Canceled {
			v.err = msg.Err
		}

	case tea.KeyMsg:
		// Normal mode key handling
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			// Stop streaming before switching views
			v.stopStreaming()
			v.clearSearch()
			return v, func() tea.Msg {
				return GoBackMsg{}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().LogSearchNext):
			if v.searchRegex != nil && len(v.searchMatches) > 0 {
				v.searchCursor = (v.searchCursor + 1) % len(v.searchMatches)
				v.jumpToMatch(v.searchCursor)
			}
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().LogSearchPrev):
			if v.searchRegex != nil && len(v.searchMatches) > 0 {
				v.searchCursor--
				if v.searchCursor < 0 {
					v.searchCursor = len(v.searchMatches) - 1
				}
				v.jumpToMatch(v.searchCursor)
			}
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().LogSave):
			return v, v.saveLogsCmd()

		case key.Matches(msg, theme.DefaultKeyMap().LogTimestamp):
			v.showTimestamps = !v.showTimestamps
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().LogPrevious):
			v.showPrevious = !v.showPrevious
			v.stopStreaming()
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().LogTimeRange):
			v.timeRangeIdx = (v.timeRangeIdx + 1) % len(defaultTimeRanges)
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().LogWrap):
			v.wrapText = !v.wrapText
			v.updateViewportContent()
			return v, nil

		case msg.String() == "f":
			v.viewport.GotoBottom()
			return v, nil

		case msg.String() == "G":
			v.viewport.GotoBottom()

		case msg.String() == "g":
			v.viewport.GotoTop()

		case msg.String() == "x":
			// Clear logs
			v.logs.Reset()
			v.logLines = nil
			v.clearSearch()
			v.viewport.SetContent("")

		default:
			// Let viewport handle scrolling keys (up/down/pgup/pgdn)
			var cmd tea.Cmd
			v.viewport, cmd = v.viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return v, tea.Batch(cmds...)
		}
	}

	// Update spinner
	if v.loading {
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update viewport
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *LogsView) View() string {
	if v.pod == "" {
		return theme.Styles.StatusUnknown.Render("No pod selected. Press Escape to go back.")
	}

	if v.loading && v.logs.Len() == 0 {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil && v.logs.Len() == 0 {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Header
	header := theme.Styles.PanelTitle.Render("Logs: " + v.namespace + "/" + v.pod)
	if v.container != "" {
		header += theme.Styles.PanelTitle.Render(" (" + v.container + ")")
	}

	// Line count
	header += bgStyle.Render(" ") + theme.Styles.InfoValueMuted.Render(fmt.Sprintf("[%d lines]", len(v.logLines)))

	// Status indicators
	var status []string
	if v.viewport.AtBottom() {
		status = append(status, theme.Styles.StatusHealthy.Render("[tail]"))
	}
	if v.streaming {
		status = append(status, theme.Styles.StatusPending.Render("[streaming]"))
	}
	if v.showTimestamps {
		status = append(status, theme.Styles.InfoValueMuted.Render("[timestamps]"))
	}
	if v.showPrevious {
		status = append(status, theme.Styles.StatusWarning.Render("[previous]"))
	}
	if v.wrapText {
		status = append(status, theme.Styles.InfoValueMuted.Render("[wrap]"))
	}
	if v.timeRangeIdx != 5 { // not "all" (default)
		status = append(status, theme.Styles.InfoValueMuted.Render("["+defaultTimeRanges[v.timeRangeIdx].Label+"]"))
	}
	if v.searchPattern != "" {
		matchInfo := fmt.Sprintf("[/%s/ %d matches]", v.searchPattern, len(v.searchMatches))
		status = append(status, theme.Styles.StatusPending.Render(matchInfo))
	}
	if len(status) > 0 {
		header += bgStyle.Render(" ") + strings.Join(status, bgStyle.Render(" "))
	}

	// Pad header to full width
	headerWidth := lipgloss.Width(header)
	if headerWidth < v.width {
		header += bgStyle.Render(strings.Repeat(" ", v.width-headerWidth))
	}

	// Footer
	footer := theme.Styles.Help.Render("↑↓ scroll • g/G top/bottom • f tail • / search • n/N next/prev • t timestamps • p previous • w wrap • ctrl+t range • ctrl+s save • x clear • esc back")

	// Pad footer to full width
	footerWidth := lipgloss.Width(footer)
	if footerWidth < v.width {
		footer += bgStyle.Render(strings.Repeat(" ", v.width-footerWidth))
	}

	return header + "\n" + v.viewport.View() + "\n" + footer
}

// Name returns the view name
func (v *LogsView) Name() string {
	return "Logs"
}

// ShortHelp returns keybindings for help
func (v *LogsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *LogsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.Width = width
	v.viewport.Height = height - 3 // Account for header and footer
	// Re-apply wrapping on resize
	if v.wrapText && v.logs.Len() > 0 {
		v.updateViewportContent()
	}
}

// IsLoading returns whether the view is currently loading data
func (v *LogsView) IsLoading() bool {
	return v.loading
}

// Content returns the current log text
func (v *LogsView) Content() string {
	return v.logs.String()
}

// ApplySearch compiles pattern as case-insensitive regex, finds matches, jumps to first.
// Empty pattern clears the search.
func (v *LogsView) ApplySearch(pattern string) {
	if pattern == "" {
		v.clearSearch()
		v.updateViewportContent()
		return
	}
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		re, _ = regexp.Compile(regexp.QuoteMeta(pattern))
	}
	v.searchPattern = pattern
	v.searchRegex = re
	v.updateSearchMatches()
	if len(v.searchMatches) > 0 {
		v.searchCursor = 0
		v.jumpToMatch(v.searchCursor)
	}
	v.updateViewportContent()
}

// ActiveSearchPattern returns the current search pattern, or "" if none.
func (v *LogsView) ActiveSearchPattern() string {
	return v.searchPattern
}

// Refresh refreshes the logs (non-streaming load)
func (v *LogsView) Refresh() tea.Cmd {
	if v.pod == "" {
		return nil
	}

	// Stop any existing stream
	v.stopStreaming()

	v.loading = true
	v.spinner.SetMessage("Loading logs...")
	cmds := []tea.Cmd{v.spinner.Show()}

	cmds = append(cmds, func() tea.Msg {
		opts := k8s.LogOptions{
			Container:  v.container,
			Follow:     false,
			Previous:   v.showPrevious,
			Timestamps: v.showTimestamps,
		}

		// Time range handling
		tr := defaultTimeRanges[v.timeRangeIdx]
		if tr.Seconds > 0 {
			opts.SinceSeconds = tr.Seconds
		} else {
			// "all" — use TailLines for default behavior
			opts.TailLines = v.tailLines
		}

		reader, err := v.client.Logs(context.Background(), v.namespace, v.pod, v.container, opts)
		if err != nil {
			return LogsLoadedMsg{Err: err}
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			return LogsLoadedMsg{Err: err}
		}

		return LogsLoadedMsg{Logs: string(data)}
	})

	return tea.Batch(cmds...)
}

// startStreaming starts the log streaming goroutine
func (v *LogsView) startStreaming() tea.Cmd {
	if v.pod == "" {
		return nil
	}

	// Stop any existing stream
	if v.streamCancel != nil {
		v.streamCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	v.streamCancel = cancel
	v.streaming = true
	v.streamGen++
	gen := v.streamGen

	ch := make(chan string, 50)
	v.logChan = ch

	// Goroutine: read from K8s log stream, send lines to channel
	go func() {
		defer close(ch)

		opts := k8s.LogOptions{
			Container:    v.container,
			Follow:       true,
			SinceSeconds: 1, // Only new lines (avoid duplicating initial load)
			Timestamps:   v.showTimestamps,
		}

		reader, err := v.client.Logs(ctx, v.namespace, v.pod, v.container, opts)
		if err != nil {
			ch <- "Error starting log stream: " + err.Error()
			return
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case ch <- scanner.Text():
			}
		}
	}()

	return v.waitForLogLine(ch, gen)
}

// waitForLogLine blocks on the channel and returns a LogLineMsg or LogStreamEndedMsg
func (v *LogsView) waitForLogLine(ch <-chan string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return LogStreamEndedMsg{Gen: gen}
		}
		return LogLineMsg{Line: line, Gen: gen}
	}
}

// stopStreaming stops the log streaming
func (v *LogsView) stopStreaming() {
	if v.streamCancel != nil {
		v.streamCancel()
		v.streamCancel = nil
	}
	v.streaming = false
}

// updateViewportContent applies wrapping and search highlighting, then updates the viewport.
// Auto-scrolls to bottom if the viewport was already at the bottom before the update.
func (v *LogsView) updateViewportContent() {
	wasAtBottom := v.viewport.AtBottom()

	content := v.logs.String()

	if v.wrapText && v.viewport.Width > 0 {
		content = wrapLines(content, v.viewport.Width)
	}

	if v.searchRegex != nil {
		content = v.applyHighlighting(content)
	}

	v.viewport.SetContent(content)
	if wasAtBottom {
		v.viewport.GotoBottom()
	}
}

// applyHighlighting wraps regex matches with ANSI highlight styling
func (v *LogsView) applyHighlighting(content string) string {
	if v.searchRegex == nil {
		return content
	}

	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#F59E0B")).
		Foreground(lipgloss.Color("#1B1B3A"))

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		matches := v.searchRegex.FindAllStringIndex(line, -1)
		if len(matches) == 0 {
			continue
		}
		// Build highlighted line from right to left to preserve indices
		for j := len(matches) - 1; j >= 0; j-- {
			m := matches[j]
			matched := line[m[0]:m[1]]
			line = line[:m[0]] + highlightStyle.Render(matched) + line[m[1]:]
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// updateSearchMatches scans logLines for regex matches and populates searchMatches
func (v *LogsView) updateSearchMatches() {
	v.searchMatches = nil
	v.searchCursor = -1
	if v.searchRegex == nil {
		return
	}
	for i, line := range v.logLines {
		if v.searchRegex.MatchString(line) {
			v.searchMatches = append(v.searchMatches, i)
		}
	}
}

// jumpToMatch scrolls the viewport to make the match at the given cursor position visible
func (v *LogsView) jumpToMatch(cursor int) {
	if cursor < 0 || cursor >= len(v.searchMatches) {
		return
	}
	lineIdx := v.searchMatches[cursor]

	// When wrapping is on, line indices in logLines don't map 1:1 to viewport lines.
	// Approximate by counting wrapped lines up to the target.
	if v.wrapText && v.viewport.Width > 0 {
		wrappedLine := 0
		for i := 0; i < lineIdx && i < len(v.logLines); i++ {
			lineLen := len(v.logLines[i])
			if lineLen <= v.viewport.Width || v.viewport.Width <= 0 {
				wrappedLine++
			} else {
				wrappedLine += (lineLen + v.viewport.Width - 1) / v.viewport.Width
			}
		}
		v.viewport.SetYOffset(wrappedLine)
	} else {
		v.viewport.SetYOffset(lineIdx)
	}
}

// clearSearch clears the search state and reverts highlighting
func (v *LogsView) clearSearch() {
	v.searchPattern = ""
	v.searchRegex = nil
	v.searchMatches = nil
	v.searchCursor = -1
}

// saveLogsCmd saves the current logs to a file
func (v *LogsView) saveLogsCmd() tea.Cmd {
	content := v.logs.String()
	ns := v.namespace
	pod := v.pod

	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return LogsSavedMsg{Err: fmt.Errorf("cannot find home dir: %w", err)}
		}

		dir := filepath.Join(home, "kview-logs")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return LogsSavedMsg{Err: fmt.Errorf("cannot create dir: %w", err)}
		}

		ts := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("%s-%s-%s.log", ns, pod, ts)
		path := filepath.Join(dir, filename)

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return LogsSavedMsg{Err: err}
		}

		return LogsSavedMsg{Path: path}
	}
}

// wrapLines wraps long lines to fit the viewport width
func wrapLines(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrapped []string

	for _, line := range lines {
		if len(line) <= width {
			wrapped = append(wrapped, line)
			continue
		}

		// Wrap long lines
		for len(line) > width {
			wrapped = append(wrapped, line[:width])
			line = line[width:]
		}
		if len(line) > 0 {
			wrapped = append(wrapped, line)
		}
	}

	return strings.Join(wrapped, "\n")
}
