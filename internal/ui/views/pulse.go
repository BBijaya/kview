package views

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/theme"
)

// PulseView displays a k9s-style pulse dashboard with resource type gauges.
type PulseView struct {
	BaseView
	viewport      viewport.Model
	client        k8s.Client
	gauges        [gaugeCount]GaugeData
	prevGauges    [gaugeCount]GaugeData
	hasPrevGauges bool
	cpuHistory    []float64
	memHistory    []float64
	lastCPUPct    int
	lastMemPct    int
	selectedGauge int // 0..gaugeCount-1
	gridCols      int
	needsRefresh  bool
	loading       bool
	err           error
}

// NewPulseView creates a new pulse dashboard view.
func NewPulseView(client k8s.Client) *PulseView {
	vp := viewport.New(80, 20)
	vp.Style = theme.Styles.Base

	return &PulseView{
		viewport: vp,
		client:   client,
		gridCols: 4,
	}
}

// SetNamespace overrides BaseView to trigger a refresh when namespace changes.
func (v *PulseView) SetNamespace(ns string) {
	if ns != v.namespace {
		v.BaseView.SetNamespace(ns)
		v.needsRefresh = true
	}
}

// Init initializes the view and starts data fetching.
func (v *PulseView) Init() tea.Cmd {
	return v.Refresh()
}

// Refresh fetches all pulse data.
func (v *PulseView) Refresh() tea.Cmd {
	v.loading = true
	return pulseRefresh(v.client, v.namespace)
}

// Update handles messages.
func (v *PulseView) Update(msg tea.Msg) (View, tea.Cmd) {
	if v.needsRefresh && !v.loading {
		v.needsRefresh = false
		return v, v.Refresh()
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case PulseDataMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			// Save previous gauges for delta tracking
			if v.gauges[0].Name != "" {
				v.prevGauges = v.gauges
				v.hasPrevGauges = true
			}
			v.gauges = msg.Gauges
			v.lastCPUPct = msg.CPUPct
			v.lastMemPct = msg.MemPct

			// Append to history
			v.cpuHistory = append(v.cpuHistory, float64(msg.CPUPct))
			if len(v.cpuHistory) > sparklineHistoryMax {
				v.cpuHistory = v.cpuHistory[len(v.cpuHistory)-sparklineHistoryMax:]
			}
			v.memHistory = append(v.memHistory, float64(msg.MemPct))
			if len(v.memHistory) > sparklineHistoryMax {
				v.memHistory = v.memHistory[len(v.memHistory)-sparklineHistoryMax:]
			}

			v.updateContent()
		}

		// Schedule next tick
		cmds = append(cmds, tea.Tick(10*time.Second, func(time.Time) tea.Msg {
			return PulseTickMsg{}
		}))

	case PulseTickMsg:
		if !v.loading {
			cmds = append(cmds, v.Refresh())
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			// Drill down to the selected gauge's resource list view
			if v.selectedGauge >= 0 && v.selectedGauge < gaugeCount {
				vt := v.gauges[v.selectedGauge].ViewType
				return v, func() tea.Msg {
					return DrillDownViewMsg{View: vt}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Up):
			v.moveGauge(0, -1)
			v.updateContent()
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().Down):
			v.moveGauge(0, 1)
			v.updateContent()
			return v, nil

		case msg.String() == "left" || msg.String() == "h":
			v.moveGauge(-1, 0)
			v.updateContent()
			return v, nil

		case msg.String() == "right" || msg.String() == "l":
			v.moveGauge(1, 0)
			v.updateContent()
			return v, nil

		case msg.String() == "g":
			v.viewport.GotoTop()

		case msg.String() == "G":
			v.viewport.GotoBottom()
		}
	}

	// Update viewport for scrolling
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// moveGauge moves the selected gauge by dx/dy in grid coordinates.
func (v *PulseView) moveGauge(dx, dy int) {
	cols := v.gridCols
	if cols < 1 {
		cols = 4
	}

	row := v.selectedGauge / cols
	col := v.selectedGauge % cols

	col += dx
	row += dy

	// Wrap columns
	totalRows := (gaugeCount + cols - 1) / cols
	if col < 0 {
		col = cols - 1
		row--
	}
	if col >= cols {
		col = 0
		row++
	}
	if row < 0 {
		row = totalRows - 1
	}
	if row >= totalRows {
		row = 0
	}

	newIdx := row*cols + col
	if newIdx >= gaugeCount {
		// Clamp to last gauge
		newIdx = gaugeCount - 1
	}
	if newIdx < 0 {
		newIdx = 0
	}
	v.selectedGauge = newIdx
}

// View renders the pulse view.
func (v *PulseView) View() string {
	if v.loading && v.gauges[0].Name == "" {
		return theme.Styles.StatusUnknown.Render("Loading pulse data...")
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	var b strings.Builder
	b.WriteString(v.viewport.View())
	b.WriteString("\n")
	b.WriteString(v.renderHelpLine())
	return b.String()
}

func (v *PulseView) renderHelpLine() string {
	w := v.viewport.Width
	line := theme.Styles.Help.Render("←→↑↓ navigate  Enter select  Ctrl+R refresh  Esc back")
	return theme.PadToWidth(line, w, theme.ColorBackground)
}

// --- View interface ---

func (v *PulseView) Name() string {
	return "Pulse"
}

func (v *PulseView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Refresh,
		theme.DefaultKeyMap().Escape,
	}
}

func (v *PulseView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.Width = width
	v.viewport.Height = height - 1 // reserve 1 line for help footer
	if v.viewport.Height < 1 {
		v.viewport.Height = 1
	}
	if v.gauges[0].Name != "" {
		v.updateContent()
	}
}

func (v *PulseView) SetClient(client k8s.Client) {
	v.client = client
}

func (v *PulseView) IsLoading() bool {
	return v.loading
}
