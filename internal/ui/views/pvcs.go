package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// PVCsLoadedMsg is sent when PVCs are loaded
type PVCsLoadedMsg struct {
	PVCs []k8s.PVCInfo
	Err  error
}

// pvcColumns builds the column list for the PVCs table.
// When showNS is true, the NAMESPACE column is prepended.
func pvcColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 30, MinWidth: 15, Flexible: true},
		components.Column{Title: "STATUS", Width: 10},
		components.Column{Title: "VOLUME", Width: 35, MinWidth: 15, Flexible: true},
		components.Column{Title: "CAPACITY", Width: 10, Align: lipgloss.Right},
		components.Column{Title: "ACCESS", Width: 10},
		components.Column{Title: "STORAGECLASS", Width: 15},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// PVCsView displays a list of persistent volume claims
type PVCsView struct {
	BaseView
	table   *components.Table
	filter  *components.SearchInput
	client  k8s.Client
	pvcs    []k8s.PVCInfo
	showNS  bool
	loading bool
	err     error
	spinner *components.Spinner
}

// NewPVCsView creates a new PVCs view
func NewPVCsView(client k8s.Client) *PVCsView {
	v := &PVCsView{
		table:   components.NewTable(pvcColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading PVCs...")

	// Set contextual empty state
	v.table.SetEmptyState("💾", "No PVCs found",
		"No persistent volume claims exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *PVCsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *PVCsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case PVCsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.pvcs = msg.PVCs
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
				for _, pvc := range v.pvcs {
					if pvc.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "PersistentVolumeClaim",
								Resource:  "persistentvolumeclaims",
								Namespace: pvc.Namespace,
								Name:      pvc.Name,
								UID:       pvc.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, pvc := range v.pvcs {
					if pvc.UID == row.ID {
						pvc := pvc
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "PersistentVolumeClaim", Resource: "persistentvolumeclaims", Namespace: pvc.Namespace,
								Name: pvc.Name, UID: pvc.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, pvc := range v.pvcs {
					if pvc.UID == row.ID {
						pvc := pvc
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "PersistentVolumeClaim", Resource: "persistentvolumeclaims", Namespace: pvc.Namespace,
								Name: pvc.Name, UID: pvc.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, pvc := range v.pvcs {
					if pvc.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete PVC",
								Message: fmt.Sprintf("Delete PVC %s/%s?", pvc.Namespace, pvc.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "persistentvolumeclaims", pvc.Namespace, pvc.Name)
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
func (v *PVCsView) View() string {
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
func (v *PVCsView) Name() string {
	return "PVCs"
}

// ShortHelp returns keybindings for help
func (v *PVCsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *PVCsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *PVCsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *PVCsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *PVCsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(pvcColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *PVCsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the PVC list
func (v *PVCsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			pvcs, err := v.client.ListPVCs(context.Background(), v.namespace)
			return PVCsLoadedMsg{PVCs: pvcs, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *PVCsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedPVC returns the currently selected PVC
func (v *PVCsView) SelectedPVC() *k8s.PVCInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, pvc := range v.pvcs {
			if pvc.UID == row.ID {
				return &pvc
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *PVCsView) RowCount() int {
	return v.table.RowCount()
}

func (v *PVCsView) updateTable() {
	rows := make([]components.Row, len(v.pvcs))
	for i, pvc := range v.pvcs {
		volume := pvc.Volume
		if volume == "" {
			volume = "<pending>"
		}
		capacity := pvc.Capacity
		if capacity == "" {
			capacity = "-"
		}
		accessModes := strings.Join(pvc.AccessModes, ",")
		if accessModes == "" {
			accessModes = "-"
		}
		storageClass := pvc.StorageClass
		if storageClass == "" {
			storageClass = "<default>"
		}

		status := pvc.Status
		values := []string{}
		if v.showNS {
			values = append(values, pvc.Namespace)
		}
		values = append(values,
			pvc.Name,
			status,
			volume,
			capacity,
			accessModes,
			storageClass,
			formatAge(pvc.Age),
		)
		rows[i] = components.Row{
			ID:     pvc.UID,
			Values: values,
			Status: status,
			Labels: pvc.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *PVCsView) GetTable() *components.Table {
	return v.table
}

func (v *PVCsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
