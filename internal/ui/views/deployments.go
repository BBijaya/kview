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

// DeploymentsLoadedMsg is sent when deployments are loaded
type DeploymentsLoadedMsg struct {
	Deployments []k8s.DeploymentInfo
	Err         error
}

// StatusMsg represents a status message to display
type StatusMsg struct {
	Message string
	IsError bool
}

// ScalePickerMsg requests the app to show the scale picker overlay
type ScalePickerMsg struct {
	Namespace       string
	Name            string
	Kind            string // "deployments" or "statefulsets"
	CurrentReplicas int32
}

// deploymentColumns builds the column list for the deployments table.
// When showNS is true, the NAMESPACE column is prepended.
func deploymentColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 35, MinWidth: 20, Flexible: true},
		components.Column{Title: "READY", Width: 10, Align: lipgloss.Center},
		components.Column{Title: "UP-TO-DATE", Width: 12, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AVAILABLE", Width: 11, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// DeploymentsView displays a list of deployments
type DeploymentsView struct {
	BaseView
	table       *components.Table
	filter      *components.SearchInput
	client      k8s.Client
	deployments []k8s.DeploymentInfo
	showNS      bool
	loading     bool
	err         error
	spinner     *components.Spinner
}

// NewDeploymentsView creates a new deployments view
func NewDeploymentsView(client k8s.Client) *DeploymentsView {
	v := &DeploymentsView{
		table:   components.NewTable(deploymentColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading deployments...")

	// Set contextual empty state
	v.table.SetEmptyState("🚀", "No deployments found",
		"No deployments exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *DeploymentsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *DeploymentsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case DeploymentsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.deployments = msg.Deployments
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
				for _, dep := range v.deployments {
					if dep.UID == row.ID {
						dep := dep
						return v, func() tea.Msg {
							return DrillDownDeploymentMsg{
								DeploymentName: dep.Name,
								Namespace:      dep.Namespace,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Restart):
			if row := v.table.SelectedRow(); row != nil {
				for _, dep := range v.deployments {
					if dep.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Restart Deployment",
								Message: fmt.Sprintf("Restart deployment %s/%s?", dep.Namespace, dep.Name),
								Action: func() error {
									return v.client.Restart(context.Background(), "deployments", dep.Namespace, dep.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Scale):
			if row := v.table.SelectedRow(); row != nil {
				for _, dep := range v.deployments {
					if dep.UID == row.ID {
						dep := dep
						return v, func() tea.Msg {
							return ScalePickerMsg{
								Namespace:       dep.Namespace,
								Name:            dep.Name,
								Kind:            "deployments",
								CurrentReplicas: dep.Replicas,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, dep := range v.deployments {
					if dep.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Deployment",
								Message: fmt.Sprintf("Delete deployment %s/%s?", dep.Namespace, dep.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "deployments", dep.Namespace, dep.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, dep := range v.deployments {
					if dep.UID == row.ID {
						dep := dep
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Deployment", Resource: "deployments", Namespace: dep.Namespace,
								Name: dep.Name, UID: dep.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, dep := range v.deployments {
					if dep.UID == row.ID {
						dep := dep
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Deployment", Resource: "deployments", Namespace: dep.Namespace,
								Name: dep.Name, UID: dep.UID,
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
func (v *DeploymentsView) View() string {
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
func (v *DeploymentsView) Name() string {
	return "Deployments"
}

// ShortHelp returns keybindings for help
func (v *DeploymentsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Restart,
		theme.DefaultKeyMap().Scale,
	}
}

// SetSize sets the view dimensions
func (v *DeploymentsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *DeploymentsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *DeploymentsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *DeploymentsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(deploymentColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *DeploymentsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the deployment list
func (v *DeploymentsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			deployments, err := v.client.ListDeployments(context.Background(), v.namespace)
			return DeploymentsLoadedMsg{Deployments: deployments, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *DeploymentsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedDeployment returns the currently selected deployment
func (v *DeploymentsView) SelectedDeployment() *k8s.DeploymentInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, dep := range v.deployments {
			if dep.UID == row.ID {
				return &dep
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *DeploymentsView) RowCount() int {
	return v.table.RowCount()
}

func (v *DeploymentsView) updateTable() {
	rows := make([]components.Row, len(v.deployments))
	for i, dep := range v.deployments {
		status := getDeploymentStatus(dep)
		values := []string{}
		if v.showNS {
			values = append(values, dep.Namespace)
		}
		values = append(values,
			dep.Name,
			fmt.Sprintf("%d/%d", dep.ReadyReplicas, dep.Replicas),
			fmt.Sprintf("%d", dep.UpdatedReplicas),
			fmt.Sprintf("%d", dep.AvailableReplicas),
			formatDeploymentAge(dep.Age),
		)
		rows[i] = components.Row{
			ID:     dep.UID,
			Values: values,
			Status: status,
			Labels: dep.Labels,
		}
	}
	v.table.SetRows(rows)
}

func getDeploymentStatus(dep k8s.DeploymentInfo) string {
	if dep.ReadyReplicas == dep.Replicas && dep.Replicas > 0 {
		return "Available"
	}
	if dep.ReadyReplicas < dep.Replicas {
		if dep.UpdatedReplicas < dep.Replicas {
			return "Progressing"
		}
		return "Degraded"
	}
	if dep.Replicas == 0 {
		return "Scaled to 0"
	}
	return "Unknown"
}

func formatDeploymentAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// GetTable returns the underlying table component.
func (v *DeploymentsView) GetTable() *components.Table {
	return v.table
}

func (v *DeploymentsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
