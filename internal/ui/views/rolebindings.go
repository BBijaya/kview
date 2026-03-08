package views

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// RoleBindingsLoadedMsg is sent when RoleBindings are loaded
type RoleBindingsLoadedMsg struct {
	RoleBindings []k8s.RoleBindingInfo
	Err          error
}

// roleBindingColumns builds the column list for the RoleBindings table.
func roleBindingColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 30, MinWidth: 15, Flexible: true},
		components.Column{Title: "ROLE", Width: 30, MinWidth: 12, Flexible: true},
		components.Column{Title: "SUBJECTS", Width: 40, MinWidth: 15, Flexible: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// RoleBindingsView displays a list of RoleBindings
type RoleBindingsView struct {
	BaseView
	table        *components.Table
	filter       *components.SearchInput
	client       k8s.Client
	roleBindings []k8s.RoleBindingInfo
	showNS       bool
	loading      bool
	err          error
	spinner      *components.Spinner
}

// NewRoleBindingsView creates a new RoleBindings view
func NewRoleBindingsView(client k8s.Client) *RoleBindingsView {
	v := &RoleBindingsView{
		table:   components.NewTable(roleBindingColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading RoleBindings...")
	v.table.SetEmptyState("🔐", "No RoleBindings found",
		"No RoleBindings exist in this namespace", "")
	return v
}

// Init initializes the view
func (v *RoleBindingsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *RoleBindingsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case RoleBindingsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.roleBindings = msg.RoleBindings
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
				for _, rb := range v.roleBindings {
					if rb.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      rb.Kind,
								Resource:  "rolebindings",
								Namespace: rb.Namespace,
								Name:      rb.Name,
								UID:       rb.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, rb := range v.roleBindings {
					if rb.UID == row.ID {
						rb := rb
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "RoleBinding", Resource: "rolebindings", Namespace: rb.Namespace,
								Name: rb.Name, UID: rb.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, rb := range v.roleBindings {
					if rb.UID == row.ID {
						rb := rb
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "RoleBinding", Resource: "rolebindings", Namespace: rb.Namespace,
								Name: rb.Name, UID: rb.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, rb := range v.roleBindings {
					if rb.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete RoleBinding",
								Message: fmt.Sprintf("Delete RoleBinding %s/%s?", rb.Namespace, rb.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "rolebindings", rb.Namespace, rb.Name)
								},
							}
						}
					}
				}
			}
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
func (v *RoleBindingsView) View() string {
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

func (v *RoleBindingsView) Name() string           { return "RoleBindings" }
func (v *RoleBindingsView) IsLoading() bool         { return v.loading }
func (v *RoleBindingsView) RowCount() int           { return v.table.RowCount() }
func (v *RoleBindingsView) IsFilterVisible() bool   { return v.filter.IsVisible() }
func (v *RoleBindingsView) GetTable() *components.Table { return v.table }

func (v *RoleBindingsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up, theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter, theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

func (v *RoleBindingsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

func (v *RoleBindingsView) ResetSelection() { v.table.GotoTop() }

func (v *RoleBindingsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(roleBindingColumns(newShowNS))
	}
}

func (v *RoleBindingsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

func (v *RoleBindingsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			rbs, err := v.client.ListRoleBindings(context.Background(), v.namespace)
			return RoleBindingsLoadedMsg{RoleBindings: rbs, Err: err}
		},
	)
}

func (v *RoleBindingsView) SetClient(client k8s.Client) { v.client = client }

func (v *RoleBindingsView) updateTable() {
	rows := make([]components.Row, len(v.roleBindings))
	for i, rb := range v.roleBindings {
		role := rb.RoleKind + "/" + rb.RoleName
		subjects := rb.Subjects
		if subjects == "" {
			subjects = "-"
		}

		values := []string{}
		if v.showNS {
			values = append(values, rb.Namespace)
		}
		values = append(values,
			rb.Name,
			role,
			subjects,
			formatAge(rb.Age),
		)
		rows[i] = components.Row{
			ID:     rb.UID,
			Values: values,
			Labels: rb.Labels,
		}
	}
	v.table.SetRows(rows)
}
