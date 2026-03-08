package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// PVsLoadedMsg is sent when PVs are loaded
type PVsLoadedMsg struct {
	PVs []k8s.PVInfo
	Err error
}

// pvColumns builds the column list for the PVs table.
func pvColumns() []components.Column {
	return []components.Column{
		{Title: "NAME", Width: 30, MinWidth: 15, Flexible: true},
		{Title: "CAPACITY", Width: 10, Align: lipgloss.Right},
		{Title: "ACCESS", Width: 10},
		{Title: "RECLAIM", Width: 10},
		{Title: "STATUS", Width: 10},
		{Title: "CLAIM", Width: 30, MinWidth: 15, Flexible: true},
		{Title: "STORAGECLASS", Width: 15},
		{Title: "AGE", Width: 8, Align: lipgloss.Right},
	}
}

// PVsView displays a list of PersistentVolumes
type PVsView struct {
	BaseView
	table   *components.Table
	filter  *components.SearchInput
	client  k8s.Client
	pvs     []k8s.PVInfo
	loading bool
	err     error
	spinner *components.Spinner
}

// NewPVsView creates a new PVs view
func NewPVsView(client k8s.Client) *PVsView {
	v := &PVsView{
		table:   components.NewTable(pvColumns()),
		filter:  components.NewSearchInput(),
		client:  client,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading PVs...")
	v.table.SetEmptyState("💾", "No PVs found",
		"No PersistentVolumes exist in the cluster", "")
	return v
}

// Init initializes the view
func (v *PVsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *PVsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case PVsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.pvs = msg.PVs
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
				for _, pv := range v.pvs {
					if pv.UID == row.ID {
						if pv.Claim != "" {
							// Parse "namespace/name" format
							parts := strings.SplitN(pv.Claim, "/", 2)
							if len(parts) == 2 {
								claimNS := parts[0]
								claimName := parts[1]
								return v, func() tea.Msg {
									return NavigateToResourceMsg{
										Kind:      "PersistentVolumeClaim",
										Name:      claimName,
										Namespace: claimNS,
									}
								}
							}
						}
						// Unbound PV — fall through to ResourceSelectedMsg
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:     "PersistentVolume",
								Resource: "persistentvolumes",
								Name:     pv.Name,
								UID:      pv.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, pv := range v.pvs {
					if pv.UID == row.ID {
						pv := pv
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "PersistentVolume", Resource: "persistentvolumes",
								Name:       pv.Name, UID: pv.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, pv := range v.pvs {
					if pv.UID == row.ID {
						pv := pv
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "PersistentVolume", Resource: "persistentvolumes",
								Name:       pv.Name, UID: pv.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, pv := range v.pvs {
					if pv.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete PV",
								Message: fmt.Sprintf("Delete PV %s?", pv.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "persistentvolumes", "", pv.Name)
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
func (v *PVsView) View() string {
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

func (v *PVsView) Name() string           { return "PVs" }
func (v *PVsView) IsLoading() bool         { return v.loading }
func (v *PVsView) RowCount() int           { return v.table.RowCount() }
func (v *PVsView) IsFilterVisible() bool   { return v.filter.IsVisible() }
func (v *PVsView) GetTable() *components.Table { return v.table }

func (v *PVsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up, theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter, theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

func (v *PVsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

func (v *PVsView) ResetSelection() { v.table.GotoTop() }

func (v *PVsView) SelectedName() string {
	return v.table.SelectedValue(0)
}

func (v *PVsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			pvs, err := v.client.ListPVs(context.Background())
			return PVsLoadedMsg{PVs: pvs, Err: err}
		},
	)
}

func (v *PVsView) SetClient(client k8s.Client) { v.client = client }

func (v *PVsView) updateTable() {
	rows := make([]components.Row, len(v.pvs))
	for i, pv := range v.pvs {
		capacity := pv.Capacity
		if capacity == "" {
			capacity = "-"
		}
		accessModes := strings.Join(pv.AccessModes, ",")
		if accessModes == "" {
			accessModes = "-"
		}
		reclaimPolicy := pv.ReclaimPolicy
		if reclaimPolicy == "" {
			reclaimPolicy = "-"
		}
		claim := pv.Claim
		if claim == "" {
			claim = "-"
		}
		storageClass := pv.StorageClass
		if storageClass == "" {
			storageClass = "-"
		}

		rows[i] = components.Row{
			ID: pv.UID,
			Values: []string{
				pv.Name,
				capacity,
				accessModes,
				reclaimPolicy,
				pv.Status,
				claim,
				storageClass,
				formatAge(pv.Age),
			},
			Status: pv.Status,
			Labels: pv.Labels,
		}
	}
	v.table.SetRows(rows)
}
