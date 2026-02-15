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

// JobsLoadedMsg is sent when jobs are loaded
type JobsLoadedMsg struct {
	Jobs []k8s.JobInfo
	Err  error
}

// jobColumns builds the column list for the jobs table.
// When showNS is true, the NAMESPACE column is prepended.
func jobColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		components.Column{Title: "COMPLETIONS", Width: 12, Align: lipgloss.Center},
		components.Column{Title: "DURATION", Width: 12, Align: lipgloss.Right},
		components.Column{Title: "STATUS", Width: 12},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// JobsView displays a list of jobs
type JobsView struct {
	BaseView
	table   *components.Table
	filter  *components.SearchInput
	client  k8s.Client
	jobs    []k8s.JobInfo
	showNS  bool
	loading bool
	err     error
	spinner *components.Spinner
}

// NewJobsView creates a new jobs view
func NewJobsView(client k8s.Client) *JobsView {
	v := &JobsView{
		table:   components.NewTable(jobColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading jobs...")

	// Set contextual empty state
	v.table.SetEmptyState("⚡", "No jobs found",
		"No jobs exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *JobsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *JobsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case JobsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.jobs = msg.Jobs
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
				for _, job := range v.jobs {
					if job.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "Job",
								Resource:  "jobs",
								Namespace: job.Namespace,
								Name:      job.Name,
								UID:       job.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, job := range v.jobs {
					if job.UID == row.ID {
						job := job
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Job", Resource: "jobs", Namespace: job.Namespace,
								Name: job.Name, UID: job.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, job := range v.jobs {
					if job.UID == row.ID {
						job := job
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Job", Resource: "jobs", Namespace: job.Namespace,
								Name: job.Name, UID: job.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, job := range v.jobs {
					if job.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Job",
								Message: fmt.Sprintf("Delete job %s/%s?", job.Namespace, job.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "jobs", job.Namespace, job.Name)
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
func (v *JobsView) View() string {
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
func (v *JobsView) Name() string {
	return "Jobs"
}

// ShortHelp returns keybindings for help
func (v *JobsView) ShortHelp() []key.Binding {
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
func (v *JobsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *JobsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *JobsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *JobsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(jobColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *JobsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the job list
func (v *JobsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			jobs, err := v.client.ListJobs(context.Background(), v.namespace)
			return JobsLoadedMsg{Jobs: jobs, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *JobsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedJob returns the currently selected job
func (v *JobsView) SelectedJob() *k8s.JobInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, job := range v.jobs {
			if job.UID == row.ID {
				return &job
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *JobsView) RowCount() int {
	return v.table.RowCount()
}

func (v *JobsView) updateTable() {
	rows := make([]components.Row, len(v.jobs))
	for i, job := range v.jobs {
		// Format completions as succeeded/total
		completions := fmt.Sprintf("%d/%d", job.Succeeded, job.Completions)

		// Format duration
		duration := formatDuration(job.Duration)

		// Map status for coloring
		status := job.Status
		switch status {
		case "Complete":
			status = "Succeeded" // Use Succeeded for green color
		case "Running":
			status = "Running"
		case "Failed":
			status = "Failed"
		default:
			status = "Pending"
		}

		values := []string{}
		if v.showNS {
			values = append(values, job.Namespace)
		}
		values = append(values,
			job.Name,
			completions,
			duration,
			job.Status,
			formatAge(job.Age),
		)

		rows[i] = components.Row{
			ID:     job.UID,
			Values: values,
			Status: status,
			Labels: job.Labels,
		}
	}
	v.table.SetRows(rows)
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// GetTable returns the underlying table component.
func (v *JobsView) GetTable() *components.Table {
	return v.table
}

func (v *JobsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
