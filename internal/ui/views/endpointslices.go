package views

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// EndpointSlicesLoadedMsg is sent when endpoint slices are loaded
type EndpointSlicesLoadedMsg struct {
	EndpointSlices []k8s.EndpointSliceInfo
	Err            error
}

// endpointSliceColumns builds the column list for the endpoint slices table.
func endpointSliceColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 30},
		components.Column{Title: "ADDRESSTYPE", Width: 12},
		components.Column{Title: "PORTS", Width: 15},
		components.Column{Title: "ENDPOINTS", Width: 45},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// EndpointSlicesView displays a list of endpoint slices
type EndpointSlicesView struct {
	BaseView
	table          *components.Table
	filter         *components.SearchInput
	client         k8s.Client
	endpointSlices []k8s.EndpointSliceInfo
	showNS         bool
	loading        bool
	err            error
	spinner        *components.Spinner
}

// NewEndpointSlicesView creates a new endpoint slices view
func NewEndpointSlicesView(client k8s.Client) *EndpointSlicesView {
	v := &EndpointSlicesView{
		table:   components.NewTable(endpointSliceColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading endpoint slices...")

	v.table.SetEmptyState("🌐", "No endpoint slices found",
		"No endpoint slices exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *EndpointSlicesView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *EndpointSlicesView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case EndpointSlicesLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.endpointSlices = msg.EndpointSlices
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

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, es := range v.endpointSlices {
					if es.UID == row.ID {
						es := es
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete EndpointSlice",
								Message: fmt.Sprintf("Delete endpoint slice %s/%s?", es.Namespace, es.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "endpointslices", es.Namespace, es.Name)
								},
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, es := range v.endpointSlices {
					if es.UID == row.ID {
						es := es
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "EndpointSlice", Resource: "endpointslices", Namespace: es.Namespace,
								Name: es.Name, UID: es.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, es := range v.endpointSlices {
					if es.UID == row.ID {
						es := es
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "EndpointSlice", Resource: "endpointslices", Namespace: es.Namespace,
								Name: es.Name, UID: es.UID,
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
func (v *EndpointSlicesView) View() string {
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
func (v *EndpointSlicesView) Name() string {
	return "EndpointSlices"
}

// ShortHelp returns keybindings for help
func (v *EndpointSlicesView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *EndpointSlicesView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *EndpointSlicesView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *EndpointSlicesView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *EndpointSlicesView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(endpointSliceColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *EndpointSlicesView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the endpoint slices list
func (v *EndpointSlicesView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			endpointSlices, err := v.client.ListEndpointSlices(context.Background(), v.namespace)
			return EndpointSlicesLoadedMsg{EndpointSlices: endpointSlices, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *EndpointSlicesView) SetClient(client k8s.Client) {
	v.client = client
}

// RowCount returns the number of visible rows
func (v *EndpointSlicesView) RowCount() int {
	return v.table.RowCount()
}

// GetTable returns the underlying table component.
func (v *EndpointSlicesView) GetTable() *components.Table {
	return v.table
}

func (v *EndpointSlicesView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}

func (v *EndpointSlicesView) updateTable() {
	rows := make([]components.Row, len(v.endpointSlices))
	for i, es := range v.endpointSlices {
		values := []string{}
		if v.showNS {
			values = append(values, es.Namespace)
		}
		values = append(values,
			es.Name,
			es.AddressType,
			es.Ports,
			es.Endpoints,
			formatServiceAge(es.Age),
		)
		rows[i] = components.Row{
			ID:     es.UID,
			Values: values,
			Status: "Active",
			Labels: es.Labels,
		}
	}
	v.table.SetRows(rows)
}
