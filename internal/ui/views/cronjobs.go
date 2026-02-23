package views

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// CronJobsLoadedMsg is sent when cronjobs are loaded
type CronJobsLoadedMsg struct {
	CronJobs []k8s.CronJobInfo
	Err      error
}

// cronJobColumns builds the column list for the cronjobs table.
// When showNS is true, the NAMESPACE column is prepended.
func cronJobColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 35, MinWidth: 20, Flexible: true},
		components.Column{Title: "SCHEDULE", Width: 20},
		components.Column{Title: "SUSPEND", Width: 8, Align: lipgloss.Center},
		components.Column{Title: "ACTIVE", Width: 8, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "LAST SCHEDULE", Width: 15, Align: lipgloss.Right},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// CronJobsView displays a list of cronjobs
type CronJobsView struct {
	BaseView
	table    *components.Table
	filter   *components.SearchInput
	client   k8s.Client
	cronjobs []k8s.CronJobInfo
	showNS   bool
	loading  bool
	err      error
	spinner  *components.Spinner
}

// NewCronJobsView creates a new cronjobs view
func NewCronJobsView(client k8s.Client) *CronJobsView {
	v := &CronJobsView{
		table:   components.NewTable(cronJobColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading cronjobs...")

	// Set contextual empty state
	v.table.SetEmptyState("⏰", "No cronjobs found",
		"No cronjobs exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *CronJobsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *CronJobsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case CronJobsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.cronjobs = msg.CronJobs
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
				for _, cj := range v.cronjobs {
					if cj.UID == row.ID {
						cj := cj
						return v, func() tea.Msg {
							return DrillDownCronJobMsg{
								CronJobName: cj.Name,
								Namespace:   cj.Namespace,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, cj := range v.cronjobs {
					if cj.UID == row.ID {
						cj := cj
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "CronJob", Resource: "cronjobs", Namespace: cj.Namespace,
								Name: cj.Name, UID: cj.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, cj := range v.cronjobs {
					if cj.UID == row.ID {
						cj := cj
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "CronJob", Resource: "cronjobs", Namespace: cj.Namespace,
								Name: cj.Name, UID: cj.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, cj := range v.cronjobs {
					if cj.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete CronJob",
								Message: fmt.Sprintf("Delete cronjob %s/%s?", cj.Namespace, cj.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "cronjobs", cj.Namespace, cj.Name)
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
func (v *CronJobsView) View() string {
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
func (v *CronJobsView) Name() string {
	return "CronJobs"
}

// ShortHelp returns keybindings for help
func (v *CronJobsView) ShortHelp() []key.Binding {
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
func (v *CronJobsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *CronJobsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *CronJobsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *CronJobsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(cronJobColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *CronJobsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the cronjob list
func (v *CronJobsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			cronjobs, err := v.client.ListCronJobs(context.Background(), v.namespace)
			return CronJobsLoadedMsg{CronJobs: cronjobs, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *CronJobsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedCronJob returns the currently selected cronjob
func (v *CronJobsView) SelectedCronJob() *k8s.CronJobInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, cj := range v.cronjobs {
			if cj.UID == row.ID {
				return &cj
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *CronJobsView) RowCount() int {
	return v.table.RowCount()
}

func (v *CronJobsView) updateTable() {
	rows := make([]components.Row, len(v.cronjobs))
	for i, cj := range v.cronjobs {
		// Format suspend
		suspend := "False"
		if cj.Suspend {
			suspend = "True"
		}

		// Format last schedule as relative time
		lastSchedule := "<none>"
		if !cj.LastSchedule.IsZero() {
			lastSchedule = formatAge(time.Since(cj.LastSchedule))
		}

		// Determine status for coloring
		status := "Running"
		if cj.Suspend {
			status = "Pending" // Use pending color for suspended
		}

		values := []string{}
		if v.showNS {
			values = append(values, cj.Namespace)
		}
		values = append(values,
			cj.Name,
			cj.Schedule,
			suspend,
			fmt.Sprintf("%d", cj.Active),
			lastSchedule,
			formatAge(cj.Age),
		)

		rows[i] = components.Row{
			ID:     cj.UID,
			Values: values,
			Status: status,
			Labels: cj.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *CronJobsView) GetTable() *components.Table {
	return v.table
}

func (v *CronJobsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
