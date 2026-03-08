package views

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// DaemonSetsLoadedMsg is sent when daemonsets are loaded
type DaemonSetsLoadedMsg struct {
	DaemonSets []k8s.DaemonSetInfo
	Err        error
}

// daemonSetColumns builds the column list for the daemonsets table.
// When showNS is true, the NAMESPACE column is prepended.
func daemonSetColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		components.Column{Title: "DESIRED", Width: 8, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "CURRENT", Width: 8, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "READY", Width: 8, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AVAILABLE", Width: 10, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// DaemonSetsView displays a list of daemonsets
type DaemonSetsView struct {
	BaseView
	table      *components.Table
	filter     *components.SearchInput
	client     k8s.Client
	daemonsets []k8s.DaemonSetInfo
	showNS     bool
	loading    bool
	err        error
	spinner    *components.Spinner
}

// NewDaemonSetsView creates a new daemonsets view
func NewDaemonSetsView(client k8s.Client) *DaemonSetsView {
	v := &DaemonSetsView{
		table:   components.NewTable(daemonSetColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading daemonsets...")

	// Set contextual empty state
	v.table.SetEmptyState("👹", "No daemonsets found",
		"No daemonsets exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *DaemonSetsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *DaemonSetsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case DaemonSetsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.daemonsets = msg.DaemonSets
			v.updateTable()
		}

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

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if row := v.table.SelectedRow(); row != nil {
				for _, ds := range v.daemonsets {
					if ds.UID == row.ID {
						ds := ds
						return v, func() tea.Msg {
							return DrillDownToPodsMsg{
								OwnerKind: "DaemonSet",
								OwnerName: ds.Name,
								Namespace: ds.Namespace,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, ds := range v.daemonsets {
					if ds.UID == row.ID {
						ds := ds
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "DaemonSet", Resource: "daemonsets", Namespace: ds.Namespace,
								Name: ds.Name, UID: ds.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, ds := range v.daemonsets {
					if ds.UID == row.ID {
						ds := ds
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "DaemonSet", Resource: "daemonsets", Namespace: ds.Namespace,
								Name: ds.Name, UID: ds.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Restart):
			if row := v.table.SelectedRow(); row != nil {
				for _, ds := range v.daemonsets {
					if ds.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Restart DaemonSet",
								Message: fmt.Sprintf("Restart daemonset %s/%s?", ds.Namespace, ds.Name),
								Action: func() error {
									return v.restartDaemonSet(ds.Namespace, ds.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, ds := range v.daemonsets {
					if ds.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete DaemonSet",
								Message: fmt.Sprintf("Delete daemonset %s/%s?", ds.Namespace, ds.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "daemonsets", ds.Namespace, ds.Name)
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
func (v *DaemonSetsView) View() string {
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
func (v *DaemonSetsView) Name() string {
	return "DaemonSets"
}

// ShortHelp returns keybindings for help
func (v *DaemonSetsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
		theme.DefaultKeyMap().Restart,
		theme.DefaultKeyMap().Delete,
	}
}

// SetSize sets the view dimensions
func (v *DaemonSetsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *DaemonSetsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *DaemonSetsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *DaemonSetsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(daemonSetColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *DaemonSetsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the daemonset list
func (v *DaemonSetsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			daemonsets, err := v.client.ListDaemonSets(context.Background(), v.namespace)
			return DaemonSetsLoadedMsg{DaemonSets: daemonsets, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *DaemonSetsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedDaemonSet returns the currently selected daemonset
func (v *DaemonSetsView) SelectedDaemonSet() *k8s.DaemonSetInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, ds := range v.daemonsets {
			if ds.UID == row.ID {
				return &ds
			}
		}
	}
	return nil
}

// restartDaemonSet restarts a daemonset by updating its annotations
func (v *DaemonSetsView) restartDaemonSet(namespace, name string) error {
	// Get the current daemonset, update restart annotation
	// This uses a similar pattern to deployment restart
	resource, err := v.client.Get(context.Background(), "daemonsets", namespace, name)
	if err != nil {
		return err
	}

	// Update the pod template annotations to trigger a rollout
	annotations := resource.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Note: Full restart implementation would require patching the daemonset
	// For now, return a message indicating the action
	return fmt.Errorf("daemonset restart requires kubectl or direct API patch - use 'kubectl rollout restart daemonset/%s -n %s'", name, namespace)
}

// RowCount returns the number of visible rows
func (v *DaemonSetsView) RowCount() int {
	return v.table.RowCount()
}

func (v *DaemonSetsView) updateTable() {
	rows := make([]components.Row, len(v.daemonsets))
	for i, ds := range v.daemonsets {
		// Determine status based on ready vs desired
		status := "Running"
		if ds.ReadyNumber < ds.DesiredNumber {
			status = "Progressing"
		}
		if ds.AvailableNumber == 0 && ds.DesiredNumber > 0 {
			status = "Pending"
		}

		values := []string{}
		if v.showNS {
			values = append(values, ds.Namespace)
		}
		values = append(values,
			ds.Name,
			fmt.Sprintf("%d", ds.DesiredNumber),
			fmt.Sprintf("%d", ds.CurrentNumber),
			fmt.Sprintf("%d", ds.ReadyNumber),
			fmt.Sprintf("%d", ds.AvailableNumber),
			formatAge(ds.Age),
		)

		rows[i] = components.Row{
			ID:     ds.UID,
			Values: values,
			Status: status,
			Labels: ds.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *DaemonSetsView) GetTable() *components.Table {
	return v.table
}

func (v *DaemonSetsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
