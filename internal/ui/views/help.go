package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// HelpView displays all keyboard shortcuts in a scrollable viewport.
type HelpView struct {
	BaseView
	viewport       viewport.Model
	lastBuiltWidth int
}

// NewHelpView creates a new help view.
func NewHelpView() *HelpView {
	vp := viewport.New(80, 20)
	vp.Style = theme.Styles.Base

	return &HelpView{
		viewport: vp,
	}
}

// Init initializes the view.
func (v *HelpView) Init() tea.Cmd {
	v.viewport.SetContent(v.buildContent())
	v.viewport.GotoTop()
	return nil
}

// Update handles messages.
func (v *HelpView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, theme.DefaultKeyMap().Help):
			return v, func() tea.Msg { return GoBackMsg{} }

		case msg.String() == "G":
			v.viewport.GotoBottom()

		case msg.String() == "g":
			v.viewport.GotoTop()

		default:
			var cmd tea.Cmd
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		}
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the view.
func (v *HelpView) View() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	header := theme.Styles.PanelTitle.Render("Keyboard Shortcuts")
	headerWidth := lipgloss.Width(header)
	if headerWidth < v.width {
		header += bgStyle.Render(strings.Repeat(" ", v.width-headerWidth))
	}

	footer := theme.Styles.Help.Render("  ↑↓/pgup/pgdn scroll  g/G top/bottom  ?/esc back")
	footerWidth := lipgloss.Width(footer)
	if footerWidth < v.width {
		footer += bgStyle.Render(strings.Repeat(" ", v.width-footerWidth))
	}

	return header + "\n" + v.viewport.View() + "\n" + footer
}

func (v *HelpView) Name() string    { return "Help" }
func (v *HelpView) Content() string { return v.viewport.View() }

func (v *HelpView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Escape,
	}
}

func (v *HelpView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.Width = width
	v.viewport.Height = height - 3
	if width != v.lastBuiltWidth {
		v.lastBuiltWidth = width
		v.viewport.SetContent(v.buildContent())
	}
}

func (v *HelpView) Refresh() tea.Cmd {
	v.viewport.SetContent(v.buildContent())
	return nil
}

type helpEntry struct {
	key  string
	desc string
}

type helpGroup struct {
	title   string
	entries []helpEntry
}

func (v *HelpView) helpGroups() []helpGroup {
	return []helpGroup{
		{
			title: "Navigation",
			entries: []helpEntry{
				{"↑/k", "Move up"},
				{"↓/j", "Move down"},
				{"pgup", "Page up"},
				{"pgdn", "Page down"},
				{"home/g", "Go to top"},
				{"end/G", "Go to bottom"},
				{"←/→", "Horizontal scroll (tables)"},
			},
		},
		{
			title: "Views & Tabs",
			entries: []helpEntry{
				{"tab", "Next resource tab"},
				{"shift+tab", "Previous resource tab"},
				{"1-9", "Select resource in category"},
			},
		},
		{
			title: "Resource Actions",
			entries: []helpEntry{
				{"enter", "Select / drill down"},
				{"d", "Describe resource"},
				{"y", "Show YAML"},
				{"e", "Edit resource in $EDITOR"},
				{"c", "Copy name / content"},
				{"ctrl+d", "Delete resource"},
			},
		},
		{
			title: "Pod Actions",
			entries: []helpEntry{
				{"l", "View logs"},
				{"s", "Shell into container"},
				{"F", "Port forward"},
			},
		},
		{
			title: "Deployment Actions",
			entries: []helpEntry{
				{"r", "Restart deployment"},
				{"s", "Scale deployment"},
			},
		},
		{
			title: "Helm",
			entries: []helpEntry{
				{"enter", "View release history"},
				{"v", "View Helm values"},
				{"m", "View Helm manifest"},
			},
		},
		{
			title: "Secrets",
			entries: []helpEntry{
				{"x", "Decode secret data (base64)"},
			},
		},
		{
			title: "Filter & Sort",
			entries: []helpEntry{
				{"/", "Filter resources"},
				{"/!", "Inverse filter (combine: !-f, !-l)"},
				{"/-f", "Fuzzy filter"},
				{"/-l", "Label selector filter"},
				{"S", "Toggle sort direction"},
				{"[", "Previous sort column"},
				{"]", "Next sort column"},
			},
		},
		{
			title: "Log Viewer",
			entries: []helpEntry{
				{"/", "Search in logs"},
				{"n/N", "Next/previous match"},
				{"ctrl+s", "Save logs to file"},
				{"t", "Toggle timestamps"},
				{"p", "Toggle previous container logs"},
				{"ctrl+t", "Cycle time range"},
				{"w", "Toggle text wrap"},
			},
		},
		{
			title: "Xray",
			entries: []helpEntry{
				{"X", "Xray view (resource relationships)"},
				{":xray <kind>", "Xray tree for resource type"},
				{":xray <name>", "Xray relationships for resource"},
				{"enter", "Toggle expand/collapse (in xray)"},
			},
		},
		{
			title: "General",
			entries: []helpEntry{
				{"ctrl+r", "Refresh current view"},
				{"ctrl+p", "Open command palette"},
				{"n", "Switch namespace"},
				{"ctrl+k", "Switch context"},
				{"?", "Toggle this help view"},
				{":", "Enter command mode"},
				{"q", "Quit"},
				{"esc", "Go back / close"},
			},
		},
	}
}

// renderGroup renders a single help group as a slice of lines, each padded to colWidth.
func (v *HelpView) renderGroup(g helpGroup, colWidth int) []string {
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Background(theme.ColorBackground)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	lines := make([]string, 0, 1+len(g.entries)+1)

	// Title line
	title := headerStyle.Render(fmt.Sprintf("  %s", g.title))
	titleWidth := lipgloss.Width(title)
	if titleWidth < colWidth {
		title += bgStyle.Render(strings.Repeat(" ", colWidth-titleWidth))
	}
	lines = append(lines, title)

	// Entry lines
	for _, e := range g.entries {
		keyText := fmt.Sprintf("    %-12s", e.key)
		line := keyStyle.Render(keyText) + descStyle.Render(e.desc)
		lineWidth := lipgloss.Width(line)
		if lineWidth < colWidth {
			line += bgStyle.Render(strings.Repeat(" ", colWidth-lineWidth))
		}
		lines = append(lines, line)
	}

	// Separator (blank bg line)
	lines = append(lines, bgStyle.Render(strings.Repeat(" ", colWidth)))

	return lines
}

func (v *HelpView) buildContent() string {
	groups := v.helpGroups()

	const minColWidth = 38
	const colGap = 3

	width := v.width
	if width < 1 {
		width = 80
	}

	// Calculate number of columns based on available width
	numCols := max(1, (width+colGap)/(minColWidth+colGap))
	if numCols > 3 {
		numCols = 3
	}
	colWidth := (width - colGap*(numCols-1)) / numCols

	// Compute group heights: title + entries + separator
	groupHeights := make([]int, len(groups))
	totalLines := 0
	for i, g := range groups {
		groupHeights[i] = 1 + len(g.entries) + 1
		totalLines += groupHeights[i]
	}

	// Distribute groups into columns (greedy, balanced height)
	columns := make([][]int, numCols) // columns[col] = list of group indices
	for i := range columns {
		columns[i] = []int{}
	}
	targetPerCol := (totalLines + numCols - 1) / numCols // ceiling division
	col := 0
	colLines := 0
	for i, h := range groupHeights {
		// If adding this group would exceed target and we're not on the last column,
		// and we already have at least one group in this column, move to next column
		if colLines > 0 && colLines+h > targetPerCol && col < numCols-1 {
			col++
			colLines = 0
		}
		columns[col] = append(columns[col], i)
		colLines += h
	}

	// Render each column's groups into lines
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	colRendered := make([][]string, numCols)
	maxHeight := 0

	for c := 0; c < numCols; c++ {
		var lines []string
		for _, gi := range columns[c] {
			lines = append(lines, v.renderGroup(groups[gi], colWidth)...)
		}
		// Remove trailing separator from last group in column
		if len(lines) > 0 {
			lines = lines[:len(lines)-1]
		}
		colRendered[c] = lines
		if len(lines) > maxHeight {
			maxHeight = len(lines)
		}
	}

	// Pad shorter columns to maxHeight with bg-only lines
	emptyLines := make([]string, numCols)
	for c := 0; c < numCols; c++ {
		emptyLines[c] = bgStyle.Render(strings.Repeat(" ", colWidth))
		for len(colRendered[c]) < maxHeight {
			colRendered[c] = append(colRendered[c], emptyLines[c])
		}
	}

	// Join columns line by line
	gapStr := bgStyle.Render(strings.Repeat(" ", colGap))
	var b strings.Builder

	for lineIdx := 0; lineIdx < maxHeight; lineIdx++ {
		var line string
		for c := 0; c < numCols; c++ {
			if c > 0 {
				line += gapStr
			}
			line += colRendered[c][lineIdx]
		}
		// Pad final composed line to full width
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			line += bgStyle.Render(strings.Repeat(" ", width-lineWidth))
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}
