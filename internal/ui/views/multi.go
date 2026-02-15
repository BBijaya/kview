package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// MultiClusterLoadedMsg is sent when multi-cluster data is loaded
type MultiClusterLoadedMsg struct {
	Clusters map[string][]k8s.PodInfo
	Err      error
}

// MultiClusterView displays resources from multiple clusters side by side
type MultiClusterView struct {
	BaseView
	manager    *k8s.MultiClusterManager
	table      *components.Table
	clusters   map[string][]k8s.PodInfo
	loading    bool
	err        error
	viewMode   string // "pods", "deployments", "services"
}

// NewMultiClusterView creates a new multi-cluster view
func NewMultiClusterView(manager *k8s.MultiClusterManager) *MultiClusterView {
	columns := []components.Column{
		{Title: "CLUSTER", Width: 15},
		{Title: "NAMESPACE", Width: 15},
		{Title: "NAME", Width: 30, MinWidth: 20, Flexible: true},
		{Title: "STATUS", Width: 18},
		{Title: "READY", Width: 7},
	}

	return &MultiClusterView{
		manager:  manager,
		table:    components.NewTable(columns),
		viewMode: "pods",
	}
}

// Init initializes the view
func (v *MultiClusterView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *MultiClusterView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case MultiClusterLoadedMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.clusters = msg.Clusters
			v.updateTable()
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg {
				return theme.SwitchViewMsg{View: theme.ViewPods}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()
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
func (v *MultiClusterView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(theme.Styles.PanelTitle.Render("Multi-Cluster View"))
	b.WriteString(" │ ")
	b.WriteString(fmt.Sprintf("%d clusters connected", v.manager.ConnectedCount()))
	b.WriteString("\n\n")

	// Cluster info
	for _, info := range v.manager.GetClusterInfo() {
		style := theme.Styles.Tab
		if info.IsActive {
			style = theme.Styles.TabActive
		}
		b.WriteString(style.Render(fmt.Sprintf("[%s] %s", info.ContextName, info.ServerVersion)))
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	if v.loading {
		b.WriteString("Loading clusters...")
		return b.String()
	}

	if v.err != nil {
		b.WriteString(theme.Styles.StatusError.Render("Error: " + v.err.Error()))
		return b.String()
	}

	if len(v.clusters) == 0 {
		b.WriteString(theme.Styles.StatusUnknown.Render("No clusters connected"))
		b.WriteString("\n\n")
		b.WriteString("Connect to clusters using:\n")
		b.WriteString("  kubectl config use-context <context-name>\n")
		return b.String()
	}

	// Table
	b.WriteString(v.table.View())
	b.WriteString("\n")
	b.WriteString(theme.Styles.Help.Render("↑↓ navigate • esc back"))

	return b.String()
}

func (v *MultiClusterView) updateTable() {
	var rows []components.Row

	for clusterName, pods := range v.clusters {
		for _, pod := range pods {
			status := getPodStatus(pod)
			rows = append(rows, components.Row{
				ID: clusterName + "/" + pod.UID,
				Values: []string{
					clusterName,
					pod.Namespace,
					pod.Name,
					status,
					pod.Ready,
				},
				Status: status,
				Labels: pod.Labels,
			})
		}
	}

	v.table.SetRows(rows)
}

// Name returns the view name
func (v *MultiClusterView) Name() string {
	return "Multi-Cluster"
}

// ShortHelp returns keybindings for help
func (v *MultiClusterView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Refresh,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *MultiClusterView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.table.SetSize(width, height-8)
}

// ResetSelection resets the table cursor to the top
func (v *MultiClusterView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *MultiClusterView) IsLoading() bool {
	return v.loading
}

// SelectedName returns the name of the currently selected resource
func (v *MultiClusterView) SelectedName() string {
	return v.table.SelectedValue(1)
}

// Refresh loads data from all clusters
func (v *MultiClusterView) Refresh() tea.Cmd {
	v.loading = true
	return func() tea.Msg {
		clusters, err := v.manager.ComparePodsAcrossClusters(context.Background(), v.namespace)
		return MultiClusterLoadedMsg{
			Clusters: clusters,
			Err:      err,
		}
	}
}

// GetTable returns the underlying table component.
func (v *MultiClusterView) GetTable() *components.Table {
	return v.table
}
