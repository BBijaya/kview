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

// StatefulSetsLoadedMsg is sent when statefulsets are loaded
type StatefulSetsLoadedMsg struct {
	StatefulSets []k8s.StatefulSetInfo
	Err          error
}

// statefulSetColumns builds the column list for the statefulsets table.
// When showNS is true, the NAMESPACE column is prepended.
func statefulSetColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		components.Column{Title: "READY", Width: 10, Align: lipgloss.Center},
		components.Column{Title: "SERVICE", Width: 25, MinWidth: 15, Flexible: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// StatefulSetsView displays a list of statefulsets
type StatefulSetsView struct {
	BaseView
	table        *components.Table
	filter       *components.SearchInput
	client       k8s.Client
	statefulsets []k8s.StatefulSetInfo
	showNS       bool
	loading      bool
	err          error
	spinner      *components.Spinner
}

// NewStatefulSetsView creates a new statefulsets view
func NewStatefulSetsView(client k8s.Client) *StatefulSetsView {
	v := &StatefulSetsView{
		table:   components.NewTable(statefulSetColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading statefulsets...")

	// Set contextual empty state
	v.table.SetEmptyState("📊", "No statefulsets found",
		"No statefulsets exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *StatefulSetsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *StatefulSetsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case StatefulSetsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.statefulsets = msg.StatefulSets
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
				for _, sts := range v.statefulsets {
					if sts.UID == row.ID {
						sts := sts
						return v, func() tea.Msg {
							return DrillDownToPodsMsg{
								OwnerKind: "StatefulSet",
								OwnerName: sts.Name,
								Namespace: sts.Namespace,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, sts := range v.statefulsets {
					if sts.UID == row.ID {
						sts := sts
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "StatefulSet", Resource: "statefulsets", Namespace: sts.Namespace,
								Name: sts.Name, UID: sts.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, sts := range v.statefulsets {
					if sts.UID == row.ID {
						sts := sts
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "StatefulSet", Resource: "statefulsets", Namespace: sts.Namespace,
								Name: sts.Name, UID: sts.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, sts := range v.statefulsets {
					if sts.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete StatefulSet",
								Message: fmt.Sprintf("Delete statefulset %s/%s?", sts.Namespace, sts.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "statefulsets", sts.Namespace, sts.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Restart):
			if row := v.table.SelectedRow(); row != nil {
				for _, sts := range v.statefulsets {
					if sts.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Restart StatefulSet",
								Message: fmt.Sprintf("Restart statefulset %s/%s?", sts.Namespace, sts.Name),
								Action: func() error {
									return v.client.Restart(context.Background(), "statefulsets", sts.Namespace, sts.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Scale):
			if row := v.table.SelectedRow(); row != nil {
				for _, sts := range v.statefulsets {
					if sts.UID == row.ID {
						sts := sts
						return v, func() tea.Msg {
							return ScalePickerMsg{
								Namespace:       sts.Namespace,
								Name:            sts.Name,
								Kind:            "statefulsets",
								CurrentReplicas: sts.Replicas,
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
func (v *StatefulSetsView) View() string {
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
func (v *StatefulSetsView) Name() string {
	return "StatefulSets"
}

// ShortHelp returns keybindings for help
func (v *StatefulSetsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
		theme.DefaultKeyMap().Restart,
		theme.DefaultKeyMap().Scale,
	}
}

// SetSize sets the view dimensions
func (v *StatefulSetsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *StatefulSetsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *StatefulSetsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *StatefulSetsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(statefulSetColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *StatefulSetsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the statefulset list
func (v *StatefulSetsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			statefulsets, err := v.client.ListStatefulSets(context.Background(), v.namespace)
			return StatefulSetsLoadedMsg{StatefulSets: statefulsets, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *StatefulSetsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedStatefulSet returns the currently selected statefulset
func (v *StatefulSetsView) SelectedStatefulSet() *k8s.StatefulSetInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, sts := range v.statefulsets {
			if sts.UID == row.ID {
				return &sts
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *StatefulSetsView) RowCount() int {
	return v.table.RowCount()
}

func (v *StatefulSetsView) updateTable() {
	rows := make([]components.Row, len(v.statefulsets))
	for i, sts := range v.statefulsets {
		ready := fmt.Sprintf("%d/%d", sts.ReadyReplicas, sts.Replicas)

		status := "Running"
		if sts.ReadyReplicas < sts.Replicas {
			status = "Progressing"
		}

		values := []string{}
		if v.showNS {
			values = append(values, sts.Namespace)
		}
		values = append(values,
			sts.Name,
			ready,
			sts.ServiceName,
			formatAge(sts.Age),
		)

		rows[i] = components.Row{
			ID:     sts.UID,
			Values: values,
			Status: status,
			Labels: sts.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *StatefulSetsView) GetTable() *components.Table {
	return v.table
}

func (v *StatefulSetsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
