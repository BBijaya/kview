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

// IngressesLoadedMsg is sent when ingresses are loaded
type IngressesLoadedMsg struct {
	Ingresses []k8s.IngressInfo
	Err       error
}

// ingressColumns builds the column list for the ingresses table.
// When showNS is true, the NAMESPACE column is prepended.
func ingressColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 25, MinWidth: 15, Flexible: true},
		components.Column{Title: "CLASS", Width: 12},
		components.Column{Title: "HOSTS", Width: 30, MinWidth: 15, Flexible: true},
		components.Column{Title: "ADDRESS", Width: 20},
		components.Column{Title: "PORTS", Width: 10},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// IngressesView displays a list of ingresses
type IngressesView struct {
	BaseView
	table     *components.Table
	filter    *components.SearchInput
	client    k8s.Client
	ingresses []k8s.IngressInfo
	showNS    bool
	loading   bool
	err       error
	spinner   *components.Spinner
}

// NewIngressesView creates a new ingresses view
func NewIngressesView(client k8s.Client) *IngressesView {
	v := &IngressesView{
		table:   components.NewTable(ingressColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading ingresses...")

	// Set contextual empty state
	v.table.SetEmptyState("🔀", "No ingresses found",
		"No ingresses exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *IngressesView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *IngressesView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case IngressesLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.ingresses = msg.Ingresses
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
				for _, ing := range v.ingresses {
					if ing.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "Ingress",
								Resource:  "ingresses",
								Namespace: ing.Namespace,
								Name:      ing.Name,
								UID:       ing.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, ing := range v.ingresses {
					if ing.UID == row.ID {
						ing := ing
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Ingress", Resource: "ingresses", Namespace: ing.Namespace,
								Name: ing.Name, UID: ing.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, ing := range v.ingresses {
					if ing.UID == row.ID {
						ing := ing
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Ingress", Resource: "ingresses", Namespace: ing.Namespace,
								Name: ing.Name, UID: ing.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, ing := range v.ingresses {
					if ing.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Ingress",
								Message: fmt.Sprintf("Delete ingress %s/%s?", ing.Namespace, ing.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "ingresses", ing.Namespace, ing.Name)
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
func (v *IngressesView) View() string {
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
func (v *IngressesView) Name() string {
	return "Ingresses"
}

// ShortHelp returns keybindings for help
func (v *IngressesView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *IngressesView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *IngressesView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *IngressesView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *IngressesView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(ingressColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *IngressesView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the ingress list
func (v *IngressesView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			ingresses, err := v.client.ListIngresses(context.Background(), v.namespace)
			return IngressesLoadedMsg{Ingresses: ingresses, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *IngressesView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedIngress returns the currently selected ingress
func (v *IngressesView) SelectedIngress() *k8s.IngressInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, ing := range v.ingresses {
			if ing.UID == row.ID {
				return &ing
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *IngressesView) RowCount() int {
	return v.table.RowCount()
}

func (v *IngressesView) updateTable() {
	rows := make([]components.Row, len(v.ingresses))
	for i, ing := range v.ingresses {
		hosts := strings.Join(ing.Hosts, ",")
		if hosts == "" {
			hosts = "*"
		}
		class := ing.Class
		if class == "" {
			class = "<none>"
		}
		address := ing.Address
		if address == "" {
			address = "<pending>"
		}

		values := []string{}
		if v.showNS {
			values = append(values, ing.Namespace)
		}
		values = append(values,
			ing.Name,
			class,
			hosts,
			address,
			ing.Ports,
			formatAge(ing.Age),
		)
		rows[i] = components.Row{
			ID:     ing.UID,
			Values: values,
			Status: "Active",
			Labels: ing.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *IngressesView) GetTable() *components.Table {
	return v.table
}

func (v *IngressesView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
