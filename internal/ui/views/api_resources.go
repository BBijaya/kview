package views

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// APIResourcesLoadedMsg is sent when API resources are loaded
type APIResourcesLoadedMsg struct {
	Resources []k8s.APIResourceInfo
	Err       error
}

// APIResourcesView displays all discovered API resource types in a table.
type APIResourcesView struct {
	BaseView
	table     *components.Table
	filter    *components.SearchInput
	client    k8s.Client
	resources []k8s.APIResourceInfo
	loading   bool
	err       error
	spinner   *components.Spinner
}

// NewAPIResourcesView creates a new API resources view
func NewAPIResourcesView(client k8s.Client) *APIResourcesView {
	columns := []components.Column{
		{Title: "NAME", Width: 25, MinWidth: 15, Flexible: true},
		{Title: "SHORTNAMES", Width: 12},
		{Title: "GROUP", Width: 25, MinWidth: 10, Flexible: true},
		{Title: "VERSION", Width: 10},
		{Title: "KIND", Width: 25, MinWidth: 15, Flexible: true},
		{Title: "NAMESPACED", Width: 10},
		{Title: "VERBS", Width: 30, MinWidth: 15, Flexible: true},
	}

	v := &APIResourcesView{
		table:   components.NewTable(columns),
		filter:  components.NewSearchInput(),
		client:  client,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading API resources...")

	v.table.SetEmptyState("📦", "No API resources found",
		"API resource discovery may not have completed yet", "")

	return v
}

// Init initializes the view
func (v *APIResourcesView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *APIResourcesView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case APIResourcesLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.resources = msg.Resources
			v.updateTable()
		}

	case components.FilterChangedMsg:
		v.table.SetFilter(msg.Value)

	case components.FilterClosedMsg:
		v.filter.Hide()

	case tea.KeyPressMsg:
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
				resourceName := row.ID
				return v, func() tea.Msg {
					return SwitchToGenericResourceMsg{Resource: resourceName}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()
		}
	}

	if v.loading {
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *APIResourcesView) View() string {
	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	content := v.table.View()

	if v.filter.IsVisible() {
		content = v.filter.View() + "\n" + content
	}

	return content
}

// Name returns the view name
func (v *APIResourcesView) Name() string {
	return "API Resources"
}

// ShortHelp returns keybindings for help
func (v *APIResourcesView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
	}
}

// SetSize sets the view dimensions
func (v *APIResourcesView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *APIResourcesView) ResetSelection() {
	v.table.GotoTop()
}

// SelectedName returns the name of the currently selected resource
func (v *APIResourcesView) SelectedName() string {
	return v.table.SelectedValue(0)
}

// Refresh loads API resources from the registry
func (v *APIResourcesView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			reg := v.client.APIResources()
			if reg == nil {
				return APIResourcesLoadedMsg{Err: nil}
			}
			return APIResourcesLoadedMsg{Resources: reg.All()}
		},
	)
}

// SetClient sets a new k8s client
func (v *APIResourcesView) SetClient(client k8s.Client) {
	v.client = client
}

// GetTable returns the underlying table component.
func (v *APIResourcesView) GetTable() *components.Table {
	return v.table
}

// IsFilterVisible reports whether the filter input is visible.
func (v *APIResourcesView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}

// RowCount returns the number of visible rows
func (v *APIResourcesView) RowCount() int {
	return v.table.RowCount()
}

func (v *APIResourcesView) updateTable() {
	rows := make([]components.Row, len(v.resources))
	for i, res := range v.resources {
		group := res.Group
		if group == "" {
			group = ""
		}

		namespaced := "true"
		if !res.Namespaced {
			namespaced = "false"
		}

		shortNames := strings.Join(res.ShortNames, ",")
		verbs := strings.Join(res.Verbs, ",")

		rows[i] = components.Row{
			ID: res.Resource,
			Values: []string{
				res.Resource,
				shortNames,
				group,
				res.Version,
				res.Kind,
				namespaced,
				verbs,
			},
			Status: lipgloss.NewStyle().Foreground(theme.ColorText).Render(""),
		}
	}
	v.table.SetRows(rows)
}

// SwitchToGenericResourceMsg requests switching to a generic resource view for a specific resource type.
type SwitchToGenericResourceMsg struct {
	Resource string // plural resource name
}
