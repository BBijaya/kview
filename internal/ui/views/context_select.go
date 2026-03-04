package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// ContextSelectedMsg is sent when a context is selected from the list
type ContextSelectedMsg struct {
	Context string
}

// ContextsListLoadedMsg is sent when context list is loaded
type ContextsListLoadedMsg struct {
	Contexts []k8s.ContextInfo
	Err      error
}

// ContextSelectView displays contexts in a table for selection
type ContextSelectView struct {
	BaseView
	table    *components.Table
	contexts []k8s.ContextInfo
	loading  bool
	err      error
	spinner  *components.Spinner
}

// NewContextSelectView creates a new context select view
func NewContextSelectView() *ContextSelectView {
	columns := []components.Column{
		{Title: "CURRENT", Width: 8, Align: lipgloss.Center},
		{Title: "NAME", Width: 30, MinWidth: 15, Flexible: true},
		{Title: "CLUSTER", Width: 25, MinWidth: 15, Flexible: true},
		{Title: "USER", Width: 25, MinWidth: 15, Flexible: true},
		{Title: "NAMESPACE", Width: 15, MinWidth: 10, Flexible: true},
	}

	v := &ContextSelectView{
		table:   components.NewTable(columns),
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading contexts...")

	v.table.SetEmptyState("", "No contexts found",
		"Could not read kubeconfig contexts", "")

	return v
}

// Init initializes the view
func (v *ContextSelectView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *ContextSelectView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ContextsListLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.contexts = msg.Contexts
			v.updateTable()
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if row := v.table.SelectedRow(); row != nil {
				ctx := row.ID
				return v, func() tea.Msg {
					return ContextSelectedMsg{Context: ctx}
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
func (v *ContextSelectView) View() string {
	if v.loading {
		return v.spinner.ViewCentered(v.width, v.height)
	}

	if v.err != nil {
		return theme.Styles.StatusError.Render("Error: " + v.err.Error())
	}

	return v.table.View()
}

// Name returns the view name
func (v *ContextSelectView) Name() string {
	return "Contexts"
}

// ShortHelp returns keybindings for help
func (v *ContextSelectView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *ContextSelectView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.table.SetSize(width, height)
}

// ResetSelection resets the table cursor to the top
func (v *ContextSelectView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *ContextSelectView) IsLoading() bool {
	return v.loading
}

// SelectedName returns the name of the currently selected context
func (v *ContextSelectView) SelectedName() string {
	return v.table.SelectedValue(1) // NAME column
}

// SetClient sets a new k8s client (interface compliance, not used for context listing)
func (v *ContextSelectView) SetClient(client k8s.Client) {
	// Contexts are read from kubeconfig directly, no client needed
}

// Refresh fetches the context list
func (v *ContextSelectView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			contexts, err := k8s.GetContexts()
			return ContextsListLoadedMsg{Contexts: contexts, Err: err}
		},
	)
}

func (v *ContextSelectView) updateTable() {
	var rows []components.Row
	for _, ctx := range v.contexts {
		current := ""
		if ctx.Current {
			current = "*"
		}
		ns := ctx.Namespace
		if ns == "" {
			ns = "default"
		}
		rows = append(rows, components.Row{
			ID:     ctx.Name,
			Values: []string{current, ctx.Name, ctx.Cluster, ctx.User, ns},
			Status: "Active",
		})
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *ContextSelectView) GetTable() *components.Table {
	return v.table
}
