package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// NodesLoadedMsg is sent when nodes are loaded
type NodesLoadedMsg struct {
	Nodes []k8s.NodeInfo
	Err   error
}

// NodeMetricsLoadedMsg is sent when node metrics are loaded
type NodeMetricsLoadedMsg struct {
	Metrics []k8s.NodeMetrics
	Err     error
}

// NodesView displays a list of nodes (cluster-scoped)
type NodesView struct {
	BaseView
	table   *components.Table
	filter  *components.SearchInput
	client  k8s.Client
	nodes   []k8s.NodeInfo
	metrics map[string]*k8s.NodeMetrics
	loading bool
	err     error
	spinner *components.Spinner
}

// NewNodesView creates a new nodes view
func NewNodesView(client k8s.Client) *NodesView {
	columns := []components.Column{
		{Title: "NAME", Width: 30, MinWidth: 20, Flexible: true},
		{Title: "STATUS", Width: 10},
		{Title: "ROLE", Width: 15},
		{Title: "TAINTS", Width: 20, MinWidth: 10, Flexible: true},
		{Title: "VERSION", Width: 12},
		{Title: "AGE", Width: 8, Align: lipgloss.Right},
		{Title: "PODS", Width: 6, Align: lipgloss.Right},
		{Title: "CPU", Width: 7, Align: lipgloss.Right},
		{Title: "CPU/A", Width: 7, Align: lipgloss.Right},
		{Title: "%CPU", Width: 6, Align: lipgloss.Right},
		{Title: "MEM", Width: 7, Align: lipgloss.Right},
		{Title: "MEM/A", Width: 7, Align: lipgloss.Right},
		{Title: "%MEM", Width: 6, Align: lipgloss.Right},
	}

	v := &NodesView{
		table:   components.NewTable(columns),
		filter:  components.NewSearchInput(),
		client:  client,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading nodes...")

	// Set contextual empty state
	v.table.SetEmptyState("🖥️", "No nodes found",
		"Unable to retrieve cluster nodes", "")

	return v
}

// Init initializes the view
func (v *NodesView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *NodesView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case NodesLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.nodes = msg.Nodes
			v.updateTable()
		}

	case NodeMetricsLoadedMsg:
		if msg.Err == nil {
			v.metrics = make(map[string]*k8s.NodeMetrics)
			for i := range msg.Metrics {
				m := &msg.Metrics[i]
				v.metrics[m.Name] = m
			}
			v.updateTable()
		}

	case components.FilterChangedMsg:
		v.table.SetFilter(msg.Value)

	case components.FilterClosedMsg:
		v.filter.Hide()

	case tea.KeyMsg:
		// Handle filter input first if visible
		if v.filter.IsVisible() {
			var cmd tea.Cmd
			v.filter, cmd = v.filter.Update(msg)
			return v, cmd
		}

		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Filter):
			v.filter.Show()
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if row := v.table.SelectedRow(); row != nil {
				for _, node := range v.nodes {
					if node.UID == row.ID {
						name := node.Name
						return v, func() tea.Msg {
							return DrillDownNodeMsg{NodeName: name}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, node := range v.nodes {
					if node.UID == row.ID {
						node := node
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Node", Resource: "nodes", Namespace: "",
								Name: node.Name, UID: node.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, node := range v.nodes {
					if node.UID == row.ID {
						node := node
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Node", Resource: "nodes", Namespace: "",
								Name: node.Name, UID: node.UID,
							}
						}
					}
				}
			}
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

	// Update table
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *NodesView) View() string {
	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	content := v.table.View()

	// Add filter input if visible
	if v.filter.IsVisible() {
		content = v.filter.View() + "\n" + content
	}

	return content
}

// Name returns the view name
func (v *NodesView) Name() string {
	return "Nodes"
}

// ShortHelp returns keybindings for help
func (v *NodesView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *NodesView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *NodesView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *NodesView) IsLoading() bool {
	return v.loading
}

// SelectedName returns the name of the currently selected resource
func (v *NodesView) SelectedName() string {
	return v.table.SelectedValue(0)
}

// Refresh refreshes the node list
func (v *NodesView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			nodes, err := v.client.ListNodes(context.Background())
			return NodesLoadedMsg{Nodes: nodes, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *NodesView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedNode returns the currently selected node
func (v *NodesView) SelectedNode() *k8s.NodeInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, node := range v.nodes {
			if node.UID == row.ID {
				return &node
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *NodesView) RowCount() int {
	return v.table.RowCount()
}

func (v *NodesView) updateTable() {
	rows := make([]components.Row, len(v.nodes))
	for i, node := range v.nodes {
		roles := strings.Join(node.Roles, ",")
		taints := strings.Join(node.Taints, ",")

		cpuUsage := "n/a"
		cpuAlloc := "n/a"
		cpuPct := "n/a"
		memUsage := "n/a"
		memAlloc := "n/a"
		memPct := "n/a"

		if node.CPUAllocatable > 0 {
			cpuAlloc = k8s.FormatCPU(node.CPUAllocatable)
		}
		if node.MemAllocatable > 0 {
			memAlloc = k8s.FormatMemory(node.MemAllocatable)
		}

		if m, ok := v.metrics[node.Name]; ok {
			cpuUsage = k8s.FormatCPU(m.CPUUsage)
			memUsage = k8s.FormatMemory(m.MemUsage)
			if node.CPUAllocatable > 0 {
				cpuPct = fmt.Sprintf("%d%%", m.CPUUsage*100/node.CPUAllocatable)
			}
			if node.MemAllocatable > 0 {
				memPct = fmt.Sprintf("%d%%", m.MemUsage*100/node.MemAllocatable)
			}
		}

		rows[i] = components.Row{
			ID: node.UID,
			Values: []string{
				node.Name,
				node.Status,
				roles,
				taints,
				node.Version,
				formatAge(node.Age),
				fmt.Sprintf("%d", node.PodCount),
				cpuUsage,
				cpuAlloc,
				cpuPct,
				memUsage,
				memAlloc,
				memPct,
			},
			Status: node.Status,
			Labels: node.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *NodesView) GetTable() *components.Table {
	return v.table
}

func (v *NodesView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
