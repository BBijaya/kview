package views

import (
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// StopPortForwardMsg requests stopping a port forward session by ID
type StopPortForwardMsg struct {
	ID int
}

// pfColumns builds the column list for the port forwards table.
func pfColumns() []components.Column {
	return []components.Column{
		{Title: "ID", Width: 5, Align: lipgloss.Right, IsNumeric: true},
		{Title: "NAMESPACE", Width: 15},
		{Title: "RESOURCE", Width: 30, MinWidth: 15, Flexible: true},
		{Title: "CONTAINER", Width: 20, MinWidth: 10, Flexible: true},
		{Title: "LOCAL PORT", Width: 12, Align: lipgloss.Right, IsNumeric: true},
		{Title: "REMOTE PORT", Width: 13, Align: lipgloss.Right, IsNumeric: true},
		{Title: "ADDRESS", Width: 16},
	}
}

// PortForwardsView displays active port forward sessions
type PortForwardsView struct {
	BaseView
	table     *components.Table
	filter    *components.SearchInput
	pfManager *k8s.PortForwardManager
	sessions  []*k8s.PortForwardSession
}

// NewPortForwardsView creates a new port forwards view
func NewPortForwardsView(pfManager *k8s.PortForwardManager) *PortForwardsView {
	v := &PortForwardsView{
		table:     components.NewTable(pfColumns()),
		filter:    components.NewSearchInput(),
		pfManager: pfManager,
	}
	v.focused = true

	v.table.SetEmptyState("🔌", "No active port forwards",
		"No port forward sessions are currently active",
		"Use Shift+F from pods or services to start one")

	return v
}

// Init initializes the view
func (v *PortForwardsView) Init() tea.Cmd {
	v.Refresh()
	return nil
}

// Refresh refreshes the port forwards list
func (v *PortForwardsView) Refresh() tea.Cmd {
	v.sessions = v.pfManager.ActiveSessions()
	v.updateTable()
	return nil
}

// Update handles messages
func (v *PortForwardsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case components.FilterChangedMsg:
		v.table.SetFilter(msg.Value)

	case components.FilterClosedMsg:
		v.filter.Hide()

	case tea.KeyPressMsg:
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

		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				id, err := strconv.Atoi(row.ID)
				if err != nil {
					return v, nil
				}
				// Find session for display
				var session *k8s.PortForwardSession
				for _, s := range v.sessions {
					if s.ID == id {
						session = s
						break
					}
				}
				if session == nil {
					return v, nil
				}
				s := session
				return v, func() tea.Msg {
					return ConfirmActionMsg{
						Title:   "Stop Port Forward",
						Message: fmt.Sprintf("Stop port forward %s/%s (:%d -> %d)?", s.Namespace, s.ResourceName, s.LocalPort, s.RemotePort),
						Action: func() error {
							return v.pfManager.StopForward(s.ID)
						},
					}
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
func (v *PortForwardsView) View() string {
	content := v.table.View()

	if v.filter.IsVisible() {
		content = v.filter.View() + "\n" + content
	}

	return content
}

// Name returns the view name
func (v *PortForwardsView) Name() string {
	return "Port Forwards"
}

// ShortHelp returns keybindings for help
func (v *PortForwardsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Delete,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *PortForwardsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *PortForwardsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *PortForwardsView) IsLoading() bool {
	return false
}

// SelectedName returns the name of the currently selected resource
func (v *PortForwardsView) SelectedName() string {
	return v.table.SelectedValue(2) // RESOURCE column
}

// RowCount returns the number of visible rows
func (v *PortForwardsView) RowCount() int {
	return v.table.RowCount()
}

// GetTable returns the underlying table component.
func (v *PortForwardsView) GetTable() *components.Table {
	return v.table
}

func (v *PortForwardsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}

func (v *PortForwardsView) updateTable() {
	rows := make([]components.Row, len(v.sessions))
	for i, s := range v.sessions {
		resource := s.ResourceType + "/" + s.ResourceName

		rows[i] = components.Row{
			ID: strconv.Itoa(s.ID),
			Values: []string{
				strconv.Itoa(s.ID),
				s.Namespace,
				resource,
				s.Container,
				strconv.Itoa(s.LocalPort),
				strconv.Itoa(s.RemotePort),
				s.Address,
			},
			Status: "Active",
		}
	}
	v.table.SetRows(rows)
}
