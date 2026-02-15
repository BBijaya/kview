package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// ReplicaSetsLoadedMsg is sent when replicasets are loaded
type ReplicaSetsLoadedMsg struct {
	ReplicaSets []k8s.ReplicaSetInfo
	Err         error
}

// replicaSetColumns builds the column list for the replicasets table.
// When showNS is true, the NAMESPACE column is prepended.
func replicaSetColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		components.Column{Title: "DESIRED", Width: 8, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "READY", Width: 8, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AVAILABLE", Width: 10, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "OWNER", Width: 30, MinWidth: 15, Flexible: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// ReplicaSetsView displays a list of replicasets
type ReplicaSetsView struct {
	BaseView
	table       *components.Table
	filter      *components.SearchInput
	client      k8s.Client
	replicasets []k8s.ReplicaSetInfo
	showNS      bool
	loading     bool
	err         error
	spinner     *components.Spinner
}

// NewReplicaSetsView creates a new replicasets view
func NewReplicaSetsView(client k8s.Client) *ReplicaSetsView {
	v := &ReplicaSetsView{
		table:   components.NewTable(replicaSetColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading replicasets...")

	// Set contextual empty state
	v.table.SetEmptyState("📋", "No replicasets found",
		"No replicasets exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *ReplicaSetsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *ReplicaSetsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ReplicaSetsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.replicasets = msg.ReplicaSets
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
				for _, rs := range v.replicasets {
					if rs.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "ReplicaSet",
								Resource:  "replicasets",
								Namespace: rs.Namespace,
								Name:      rs.Name,
								UID:       rs.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, rs := range v.replicasets {
					if rs.UID == row.ID {
						rs := rs
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "ReplicaSet", Resource: "replicasets", Namespace: rs.Namespace,
								Name: rs.Name, UID: rs.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, rs := range v.replicasets {
					if rs.UID == row.ID {
						rs := rs
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "ReplicaSet", Resource: "replicasets", Namespace: rs.Namespace,
								Name: rs.Name, UID: rs.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, rs := range v.replicasets {
					if rs.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete ReplicaSet",
								Message: fmt.Sprintf("Delete replicaset %s/%s?", rs.Namespace, rs.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "replicasets", rs.Namespace, rs.Name)
								},
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
func (v *ReplicaSetsView) View() string {
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
func (v *ReplicaSetsView) Name() string {
	return "ReplicaSets"
}

// ShortHelp returns keybindings for help
func (v *ReplicaSetsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
		theme.DefaultKeyMap().Delete,
	}
}

// SetSize sets the view dimensions
func (v *ReplicaSetsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *ReplicaSetsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *ReplicaSetsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *ReplicaSetsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(replicaSetColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *ReplicaSetsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the replicaset list
func (v *ReplicaSetsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			replicasets, err := v.client.ListReplicaSets(context.Background(), v.namespace)
			return ReplicaSetsLoadedMsg{ReplicaSets: replicasets, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *ReplicaSetsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedReplicaSet returns the currently selected replicaset
func (v *ReplicaSetsView) SelectedReplicaSet() *k8s.ReplicaSetInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, rs := range v.replicasets {
			if rs.UID == row.ID {
				return &rs
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *ReplicaSetsView) RowCount() int {
	return v.table.RowCount()
}

func (v *ReplicaSetsView) updateTable() {
	rows := make([]components.Row, len(v.replicasets))
	for i, rs := range v.replicasets {
		// Format owner info
		owner := ""
		if rs.OwnerKind != "" && rs.OwnerName != "" {
			owner = fmt.Sprintf("%s/%s", rs.OwnerKind, rs.OwnerName)
		}

		// Determine status based on ready vs desired
		status := "Running"
		if rs.ReadyReplicas < rs.DesiredReplicas {
			status = "Progressing"
		}

		values := []string{}
		if v.showNS {
			values = append(values, rs.Namespace)
		}
		values = append(values,
			rs.Name,
			fmt.Sprintf("%d", rs.DesiredReplicas),
			fmt.Sprintf("%d", rs.ReadyReplicas),
			fmt.Sprintf("%d", rs.AvailableReplicas),
			owner,
			formatAge(rs.Age),
		)

		rows[i] = components.Row{
			ID:     rs.UID,
			Values: values,
			Status: status,
			Labels: rs.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *ReplicaSetsView) GetTable() *components.Table {
	return v.table
}

func (v *ReplicaSetsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
