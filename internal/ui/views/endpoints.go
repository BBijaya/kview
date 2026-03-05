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

// EndpointsLoadedMsg is sent when endpoints are loaded
type EndpointsLoadedMsg struct {
	Endpoints []k8s.EndpointInfo
	Err       error
}

// endpointColumns builds the column list for the endpoints table.
func endpointColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 35},
		components.Column{Title: "ENDPOINTS", Width: 40},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// EndpointsView displays a list of endpoints
type EndpointsView struct {
	BaseView
	table     *components.Table
	filter    *components.SearchInput
	client    k8s.Client
	endpoints []k8s.EndpointInfo
	showNS    bool
	loading   bool
	err       error
	spinner   *components.Spinner
}

// NewEndpointsView creates a new endpoints view
func NewEndpointsView(client k8s.Client) *EndpointsView {
	v := &EndpointsView{
		table:   components.NewTable(endpointColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading endpoints...")

	v.table.SetEmptyState("🌐", "No endpoints found",
		"No endpoints exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *EndpointsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *EndpointsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case EndpointsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.endpoints = msg.Endpoints
			v.updateTable()
		}

	case components.FilterChangedMsg:
		v.table.SetFilter(msg.Value)

	case components.FilterClosedMsg:
		v.filter.Hide()

	case tea.KeyMsg:
		if v.filter.IsVisible() {
			var cmd tea.Cmd
			v.filter, cmd = v.filter.Update(msg)
			return v, cmd
		}

		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Filter):
			v.filter.Show()
			return v, nil

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, ep := range v.endpoints {
					if ep.UID == row.ID {
						ep := ep
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Endpoints",
								Message: fmt.Sprintf("Delete endpoints %s/%s?", ep.Namespace, ep.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "endpoints", ep.Namespace, ep.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, ep := range v.endpoints {
					if ep.UID == row.ID {
						ep := ep
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Endpoints", Resource: "endpoints", Namespace: ep.Namespace,
								Name: ep.Name, UID: ep.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, ep := range v.endpoints {
					if ep.UID == row.ID {
						ep := ep
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Endpoints", Resource: "endpoints", Namespace: ep.Namespace,
								Name: ep.Name, UID: ep.UID,
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
func (v *EndpointsView) View() string {
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

// Name returns the view name
func (v *EndpointsView) Name() string {
	return "Endpoints"
}

// ShortHelp returns keybindings for help
func (v *EndpointsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *EndpointsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *EndpointsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *EndpointsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *EndpointsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(endpointColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *EndpointsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the endpoint list
func (v *EndpointsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			endpoints, err := v.client.ListEndpoints(context.Background(), v.namespace)
			return EndpointsLoadedMsg{Endpoints: endpoints, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *EndpointsView) SetClient(client k8s.Client) {
	v.client = client
}

// RowCount returns the number of visible rows
func (v *EndpointsView) RowCount() int {
	return v.table.RowCount()
}

// GetTable returns the underlying table component.
func (v *EndpointsView) GetTable() *components.Table {
	return v.table
}

func (v *EndpointsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}

func (v *EndpointsView) updateTable() {
	rows := make([]components.Row, len(v.endpoints))
	for i, ep := range v.endpoints {
		values := []string{}
		if v.showNS {
			values = append(values, ep.Namespace)
		}
		values = append(values,
			ep.Name,
			ep.Endpoints,
			formatServiceAge(ep.Age),
		)
		rows[i] = components.Row{
			ID:     ep.UID,
			Values: values,
			Status: "Active",
			Labels: ep.Labels,
		}
	}
	v.table.SetRows(rows)
}
