package views

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// NamespaceSelectedMsg is sent when a namespace is selected from the list
type NamespaceSelectedMsg struct {
	Namespace string
}

// NamespacesListLoadedMsg is sent when namespace list is loaded
type NamespacesListLoadedMsg struct {
	Namespaces []k8s.NamespaceInfo
	Err        error
}

// NamespaceSelectView displays namespaces in a table for selection
type NamespaceSelectView struct {
	BaseView
	table      *components.Table
	filter     *components.SearchInput
	client     k8s.Client
	namespaces []k8s.NamespaceInfo
	loading    bool
	err        error
	spinner    *components.Spinner
}

// NewNamespaceSelectView creates a new namespace select view
func NewNamespaceSelectView(client k8s.Client) *NamespaceSelectView {
	columns := []components.Column{
		{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		{Title: "STATUS", Width: 10, Align: lipgloss.Center},
		{Title: "AGE", Width: 8, Align: lipgloss.Right},
	}

	v := &NamespaceSelectView{
		table:   components.NewTable(columns),
		filter:  components.NewSearchInput(),
		client:  client,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading namespaces...")

	v.table.SetEmptyState("", "No namespaces found",
		"Could not retrieve namespaces from the cluster", "")

	return v
}

// Init initializes the view
func (v *NamespaceSelectView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *NamespaceSelectView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case NamespacesListLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.namespaces = msg.Namespaces
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
				ns := row.ID
				return v, func() tea.Msg {
					return NamespaceSelectedMsg{Namespace: ns}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg {
				return GoBackMsg{}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()
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
func (v *NamespaceSelectView) View() string {
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
func (v *NamespaceSelectView) Name() string {
	return "Namespaces"
}

// ShortHelp returns keybindings for help
func (v *NamespaceSelectView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *NamespaceSelectView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *NamespaceSelectView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *NamespaceSelectView) IsLoading() bool {
	return v.loading
}

// SelectedName returns the name of the currently selected resource
func (v *NamespaceSelectView) SelectedName() string {
	return v.table.SelectedValue(0)
}

// SetClient sets a new k8s client
func (v *NamespaceSelectView) SetClient(client k8s.Client) {
	v.client = client
}

// Refresh fetches the namespace list
func (v *NamespaceSelectView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			namespaces, err := v.client.ListNamespaceInfos(context.Background())
			return NamespacesListLoadedMsg{Namespaces: namespaces, Err: err}
		},
	)
}

func (v *NamespaceSelectView) updateTable() {
	// First row: "all" option
	rows := []components.Row{
		{
			ID:     "",
			Values: []string{"all", "Active", ""},
			Status: "Active",
		},
	}
	for _, ns := range v.namespaces {
		rows = append(rows, components.Row{
			ID:     ns.Name,
			Values: []string{ns.Name, ns.Status, formatAge(ns.Age)},
			Status: ns.Status,
		})
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *NamespaceSelectView) GetTable() *components.Table {
	return v.table
}

func (v *NamespaceSelectView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
