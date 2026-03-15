package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// GenericResourcesLoadedMsg is sent when generic resources are loaded
type GenericResourcesLoadedMsg struct {
	Resources []k8s.Resource
	Err       error
}

// genericColumns builds the column list for the generic resource table.
// When showNS is true, the NAMESPACE column is prepended.
func genericColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		components.Column{Title: "AGE", Width: 10, Align: lipgloss.Right},
	)
	return cols
}

// GenericResourceView displays a list of any discovered resource type.
// It always renders the table (never a spinner) to match the behavior of
// the 14 built-in informer-backed views and avoid background leaks during
// the loading transition.
type GenericResourceView struct {
	BaseView
	table        *components.Table
	filter       *components.SearchInput
	client       k8s.Client
	resources    []k8s.Resource
	showNS        bool
	clusterScoped bool // true for cluster-scoped resources (never show NAMESPACE)
	loading       bool
	err           error
	resourceName  string // plural for API calls, e.g. "certificates"
	displayName   string // for UI, e.g. "Certificates"
	kindName      string // PascalCase kind for describe/delete, e.g. "Certificate"
}

// NewGenericResourceView creates a new generic resource view
func NewGenericResourceView(client k8s.Client) *GenericResourceView {
	v := &GenericResourceView{
		table:  components.NewTable(genericColumns(true)),
		filter: components.NewSearchInput(),
		client: client,
		showNS: true,
	}
	v.focused = true

	v.table.SetEmptyState("", "No resources found",
		"No resources of this type exist in this namespace", "")

	return v
}

// SetResource configures which resource type to display and resets state.
// namespaced indicates whether the resource is namespace-scoped; cluster-scoped
// resources never show the NAMESPACE column.
func (v *GenericResourceView) SetResource(resource, kind, display string, namespaced bool) {
	v.resourceName = resource
	v.kindName = kind
	v.displayName = display
	v.clusterScoped = !namespaced
	v.resources = nil
	v.err = nil
	v.loading = true

	// For cluster-scoped resources, always hide the namespace column
	if v.clusterScoped {
		v.showNS = false
		v.table.SetColumns(genericColumns(false))
	}

	v.table.SetEmptyState("", "No "+display+" found",
		"No "+display+" exist in this namespace", "")
	v.table.SetRows(nil)
	v.table.GotoTop()
}

// ResourceKind returns the plural resource name (for getViewTypeString)
func (v *GenericResourceView) ResourceKind() string {
	return v.resourceName
}

// KindName returns the PascalCase kind name (e.g. "Certificate").
func (v *GenericResourceView) KindName() string {
	return v.kindName
}

// Init initializes the view
func (v *GenericResourceView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *GenericResourceView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case GenericResourcesLoadedMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.resources = msg.Resources
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
				for _, r := range v.resources {
					if r.UID == row.ID {
						r := r
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      v.kindName,
								Resource:  v.resourceName,
								Namespace: r.Namespace,
								Name:      r.Name,
								UID:       r.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, r := range v.resources {
					if r.UID == row.ID {
						r := r
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       v.kindName,
								Resource:   v.resourceName,
								Namespace:  r.Namespace,
								Name:       r.Name,
								UID:        r.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, r := range v.resources {
					if r.UID == row.ID {
						r := r
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       v.kindName,
								Resource:   v.resourceName,
								Namespace:  r.Namespace,
								Name:       r.Name,
								UID:        r.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, r := range v.resources {
					if r.UID == row.ID {
						r := r
						resName := v.resourceName
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete " + v.kindName,
								Message: fmt.Sprintf("Delete %s %s/%s?", v.kindName, r.Namespace, r.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), resName, r.Namespace, r.Name)
								},
							}
						}
					}
				}
			}
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

// View renders the view. Always renders the table (with its empty state) to
// maintain a stable line count and full background coverage, matching the
// behavior of the built-in informer-backed views.
func (v *GenericResourceView) View() string {
	if v.err != nil {
		// Render error with full background fill to avoid terminal background leak
		bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
		errLine := theme.Styles.StatusError.Render("Error: " + v.err.Error())
		errWidth := lipgloss.Width(errLine)
		if errWidth < v.width {
			errLine += bgStyle.Render(strings.Repeat(" ", v.width-errWidth))
		}
		var lines []string
		lines = append(lines, errLine)
		emptyLine := bgStyle.Render(strings.Repeat(" ", v.width))
		for i := 1; i < v.height; i++ {
			lines = append(lines, emptyLine)
		}
		return strings.Join(lines, "\n")
	}

	content := v.table.View()

	if v.filter.IsVisible() {
		content = v.filter.View() + "\n" + content
	}

	return content
}

// Name returns the view name
func (v *GenericResourceView) Name() string {
	if v.displayName != "" {
		return v.displayName
	}
	return "Resources"
}

// ShortHelp returns keybindings for help
func (v *GenericResourceView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *GenericResourceView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *GenericResourceView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *GenericResourceView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column.
// Cluster-scoped resources never show the NAMESPACE column.
func (v *GenericResourceView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	if v.clusterScoped {
		if v.showNS {
			v.showNS = false
			v.table.SetColumns(genericColumns(false))
		}
		return
	}
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(genericColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *GenericResourceView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the resource list
func (v *GenericResourceView) Refresh() tea.Cmd {
	v.loading = true
	resName := v.resourceName
	return func() tea.Msg {
		resources, err := v.client.List(context.Background(), resName, v.namespace)
		return GenericResourcesLoadedMsg{Resources: resources, Err: err}
	}
}

// SetClient sets a new k8s client
func (v *GenericResourceView) SetClient(client k8s.Client) {
	v.client = client
}

// RowCount returns the number of visible rows
func (v *GenericResourceView) RowCount() int {
	return v.table.RowCount()
}

// GetTable returns the underlying table component
func (v *GenericResourceView) GetTable() *components.Table {
	return v.table
}

// IsFilterVisible returns whether the filter is active
func (v *GenericResourceView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}

func (v *GenericResourceView) updateTable() {
	rows := make([]components.Row, len(v.resources))
	for i, r := range v.resources {
		values := []string{}
		if v.showNS {
			values = append(values, r.Namespace)
		}

		age := time.Since(r.Raw.GetCreationTimestamp().Time)
		values = append(values,
			r.Name,
			formatAge(age),
		)

		rows[i] = components.Row{
			ID:     r.UID,
			Values: values,
			Status: "Active",
			Labels: r.Labels,
		}
	}
	v.table.SetRows(rows)
}
