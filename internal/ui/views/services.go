package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// ServicesLoadedMsg is sent when services are loaded
type ServicesLoadedMsg struct {
	Services []k8s.ServiceInfo
	Err      error
}

// serviceColumns builds the column list for the services table.
// When showNS is true, the NAMESPACE column is prepended.
func serviceColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 35},
		components.Column{Title: "TYPE", Width: 12},
		components.Column{Title: "CLUSTER-IP", Width: 16},
		components.Column{Title: "EXTERNAL-IP", Width: 16},
		components.Column{Title: "PORTS", Width: 25},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// ServicesView displays a list of services
type ServicesView struct {
	BaseView
	table    *components.Table
	filter   *components.SearchInput
	client   k8s.Client
	services []k8s.ServiceInfo
	showNS   bool
	loading  bool
	err      error
	spinner  *components.Spinner
}

// NewServicesView creates a new services view
func NewServicesView(client k8s.Client) *ServicesView {
	v := &ServicesView{
		table:   components.NewTable(serviceColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading services...")

	// Set contextual empty state
	v.table.SetEmptyState("🌐", "No services found",
		"No services exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *ServicesView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *ServicesView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ServicesLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.services = msg.Services
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
				for _, svc := range v.services {
					if svc.UID == row.ID {
						if len(svc.Selector) > 0 {
							svc := svc
							return v, func() tea.Msg {
								return DrillDownServiceMsg{
									ServiceName: svc.Name,
									Namespace:   svc.Namespace,
									Selector:    svc.Selector,
								}
							}
						}
						// No selector (e.g. ExternalName) — fall through to ResourceSelectedMsg
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "Service",
								Resource:  "services",
								Namespace: svc.Namespace,
								Name:      svc.Name,
								UID:       svc.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, svc := range v.services {
					if svc.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Service",
								Message: fmt.Sprintf("Delete service %s/%s?", svc.Namespace, svc.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "services", svc.Namespace, svc.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, svc := range v.services {
					if svc.UID == row.ID {
						svc := svc
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Service", Resource: "services", Namespace: svc.Namespace,
								Name: svc.Name, UID: svc.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, svc := range v.services {
					if svc.UID == row.ID {
						svc := svc
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Service", Resource: "services", Namespace: svc.Namespace,
								Name: svc.Name, UID: svc.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().PortForward):
			if svc := v.SelectedService(); svc != nil {
				s := *svc
				return v, func() tea.Msg {
					return PortForwardMsg{
						ResourceType: "services",
						ResourceName: s.Name,
						Namespace:    s.Namespace,
						ServicePorts: s.Ports,
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
func (v *ServicesView) View() string {
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
func (v *ServicesView) Name() string {
	return "Services"
}

// ShortHelp returns keybindings for help
func (v *ServicesView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *ServicesView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *ServicesView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *ServicesView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *ServicesView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(serviceColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *ServicesView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the service list
func (v *ServicesView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			services, err := v.client.ListServices(context.Background(), v.namespace)
			return ServicesLoadedMsg{Services: services, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *ServicesView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedService returns the currently selected service
func (v *ServicesView) SelectedService() *k8s.ServiceInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, svc := range v.services {
			if svc.UID == row.ID {
				return &svc
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *ServicesView) RowCount() int {
	return v.table.RowCount()
}

func (v *ServicesView) updateTable() {
	rows := make([]components.Row, len(v.services))
	for i, svc := range v.services {
		externalIP := svc.ExternalIP
		if externalIP == "" {
			externalIP = "<none>"
		}

		ports := formatPorts(svc.Ports)

		values := []string{}
		if v.showNS {
			values = append(values, svc.Namespace)
		}
		values = append(values,
			svc.Name,
			svc.Type,
			svc.ClusterIP,
			externalIP,
			ports,
			formatServiceAge(svc.Age),
		)
		rows[i] = components.Row{
			ID:     svc.UID,
			Values: values,
			Status: "Active", // Services don't have a status like pods
			Labels: svc.Labels,
		}
	}
	v.table.SetRows(rows)
}

func formatPorts(ports []k8s.ServicePort) string {
	var portStrs []string
	for _, p := range ports {
		portStr := fmt.Sprintf("%d/%s", p.Port, p.Protocol)
		if p.NodePort > 0 {
			portStr = fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol)
		}
		portStrs = append(portStrs, portStr)
	}
	return strings.Join(portStrs, ",")
}

func formatServiceAge(d time.Duration) string {
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
func (v *ServicesView) GetTable() *components.Table {
	return v.table
}

func (v *ServicesView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
