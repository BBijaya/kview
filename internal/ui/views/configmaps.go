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

// ConfigMapsLoadedMsg is sent when configmaps are loaded
type ConfigMapsLoadedMsg struct {
	ConfigMaps []k8s.ConfigMapInfo
	Err        error
}

// configMapColumns builds the column list for the configmaps table.
// When showNS is true, the NAMESPACE column is prepended.
func configMapColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 40, MinWidth: 20, Flexible: true},
		components.Column{Title: "DATA", Width: 6, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// ConfigMapsView displays a list of configmaps
type ConfigMapsView struct {
	BaseView
	table      *components.Table
	filter     *components.SearchInput
	client     k8s.Client
	configmaps []k8s.ConfigMapInfo
	showNS     bool
	loading    bool
	err        error
	spinner    *components.Spinner
}

// NewConfigMapsView creates a new configmaps view
func NewConfigMapsView(client k8s.Client) *ConfigMapsView {
	v := &ConfigMapsView{
		table:   components.NewTable(configMapColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading configmaps...")

	// Set contextual empty state
	v.table.SetEmptyState("⚙️", "No configmaps found",
		"No configmaps exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *ConfigMapsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *ConfigMapsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ConfigMapsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.configmaps = msg.ConfigMaps
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
				for _, cm := range v.configmaps {
					if cm.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "ConfigMap",
								Resource:  "configmaps",
								Namespace: cm.Namespace,
								Name:      cm.Name,
								UID:       cm.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, cm := range v.configmaps {
					if cm.UID == row.ID {
						cm := cm
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "ConfigMap", Resource: "configmaps", Namespace: cm.Namespace,
								Name: cm.Name, UID: cm.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, cm := range v.configmaps {
					if cm.UID == row.ID {
						cm := cm
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "ConfigMap", Resource: "configmaps", Namespace: cm.Namespace,
								Name: cm.Name, UID: cm.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, cm := range v.configmaps {
					if cm.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete ConfigMap",
								Message: fmt.Sprintf("Delete configmap %s/%s?", cm.Namespace, cm.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "configmaps", cm.Namespace, cm.Name)
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
func (v *ConfigMapsView) View() string {
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
func (v *ConfigMapsView) Name() string {
	return "ConfigMaps"
}

// ShortHelp returns keybindings for help
func (v *ConfigMapsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *ConfigMapsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *ConfigMapsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *ConfigMapsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *ConfigMapsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(configMapColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *ConfigMapsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the configmap list
func (v *ConfigMapsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			configmaps, err := v.client.ListConfigMaps(context.Background(), v.namespace)
			return ConfigMapsLoadedMsg{ConfigMaps: configmaps, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *ConfigMapsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedConfigMap returns the currently selected configmap
func (v *ConfigMapsView) SelectedConfigMap() *k8s.ConfigMapInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, cm := range v.configmaps {
			if cm.UID == row.ID {
				return &cm
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *ConfigMapsView) RowCount() int {
	return v.table.RowCount()
}

func (v *ConfigMapsView) updateTable() {
	rows := make([]components.Row, len(v.configmaps))
	for i, cm := range v.configmaps {
		values := []string{}
		if v.showNS {
			values = append(values, cm.Namespace)
		}
		values = append(values,
			cm.Name,
			fmt.Sprintf("%d", cm.DataCount),
			formatAge(cm.Age),
		)
		rows[i] = components.Row{
			ID:     cm.UID,
			Values: values,
			Status: "Active",
			Labels: cm.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *ConfigMapsView) GetTable() *components.Table {
	return v.table
}

func (v *ConfigMapsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
