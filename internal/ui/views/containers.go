package views

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// containerRow holds a container and whether it is an init container
type containerRow struct {
	k8s.ContainerInfo
	IsInit bool
}

// ContainersView displays the containers of a specific pod
type ContainersView struct {
	BaseView
	table      *components.Table
	client     k8s.Client
	pod        k8s.PodInfo
	containers []containerRow
}

// NewContainersView creates a new containers view
func NewContainersView(client k8s.Client) *ContainersView {
	cols := []components.Column{
		{Title: "NAME", Width: 30, MinWidth: 15, Flexible: true},
		{Title: "IMAGE", Width: 40, MinWidth: 20, Flexible: true},
		{Title: "READY", Width: 7, Align: lipgloss.Center},
		{Title: "STATE", Width: 12},
		{Title: "RESTARTS", Width: 10, Align: lipgloss.Right, IsNumeric: true},
	}
	v := &ContainersView{
		table:  components.NewTable(cols),
		client: client,
	}
	v.focused = true
	v.table.SetEmptyState("", "No containers", "This pod has no containers", "")
	return v
}

// SetPod sets the parent pod and builds the containers table
func (v *ContainersView) SetPod(pod k8s.PodInfo) {
	v.pod = pod
	v.containers = nil

	// Init containers first (prefixed with "init:")
	for _, c := range pod.InitContainers {
		v.containers = append(v.containers, containerRow{ContainerInfo: c, IsInit: true})
	}
	// Regular containers
	for _, c := range pod.Containers {
		v.containers = append(v.containers, containerRow{ContainerInfo: c, IsInit: false})
	}

	v.updateTable()
}

func (v *ContainersView) updateTable() {
	rows := make([]components.Row, len(v.containers))
	for i, c := range v.containers {
		name := c.Name
		if c.IsInit {
			name = "init:" + name
		}

		ready := "false"
		if c.Ready {
			ready = "true"
		}

		state := c.State
		if c.StateReason != "" {
			state = c.StateReason
		}

		rows[i] = components.Row{
			ID:     fmt.Sprintf("%s/%s", v.pod.UID, c.Name),
			Values: []string{name, c.Image, ready, state, fmt.Sprintf("%d", c.RestartCount)},
			Status: state,
		}
	}
	v.table.SetRows(rows)
}

// Init is a no-op; data is passed via SetPod
func (v *ContainersView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *ContainersView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Enter),
			key.Matches(msg, theme.DefaultKeyMap().Logs):
			// Open logs for the selected container
			if c := v.selectedContainer(); c != nil {
				container := c.Name
				pod := v.pod
				return v, func() tea.Msg {
					return OpenViewMsg{
						TargetView: theme.ViewLogs,
						Kind:       "Pod",
						Resource:   "pods",
						Namespace:  pod.Namespace,
						Name:       pod.Name,
						UID:        pod.UID,
						Container:  container,
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Shell):
			if c := v.selectedContainer(); c != nil {
				container := c.Name
				pod := v.pod
				return v, func() tea.Msg {
					return ExecShellMsg{
						Namespace: pod.Namespace,
						Pod:       pod.Name,
						Container: container,
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			pod := v.pod
			return v, func() tea.Msg {
				return OpenViewMsg{
					TargetView: theme.ViewDescribe,
					Kind:       "Pod",
					Resource:   "pods",
					Namespace:  pod.Namespace,
					Name:       pod.Name,
					UID:        pod.UID,
				}
			}
		}
	}

	// Update table
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

// View renders the view
func (v *ContainersView) View() string {
	return v.table.View()
}

// Name returns the view name
func (v *ContainersView) Name() string {
	return "Containers"
}

// ShortHelp returns keybindings for help
func (v *ContainersView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Logs,
		theme.DefaultKeyMap().Describe,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *ContainersView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.table.SetSize(width, height)
}

// Refresh is a no-op; container data comes from the parent pod
func (v *ContainersView) Refresh() tea.Cmd {
	return nil
}

// SelectedName returns the name of the currently selected container
func (v *ContainersView) SelectedName() string {
	return v.table.SelectedValue(0)
}

// RowCount returns the number of visible rows
func (v *ContainersView) RowCount() int {
	return v.table.RowCount()
}

// GetTable returns the underlying table component
func (v *ContainersView) GetTable() *components.Table {
	return v.table
}

// Pod returns the current parent pod
func (v *ContainersView) Pod() k8s.PodInfo {
	return v.pod
}

// SetClient sets a new k8s client
func (v *ContainersView) SetClient(client k8s.Client) {
	v.client = client
}

func (v *ContainersView) selectedContainer() *containerRow {
	row := v.table.SelectedRow()
	if row == nil {
		return nil
	}
	for i := range v.containers {
		id := fmt.Sprintf("%s/%s", v.pod.UID, v.containers[i].Name)
		if id == row.ID {
			return &v.containers[i]
		}
	}
	return nil
}
