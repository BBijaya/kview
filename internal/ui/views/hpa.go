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

// HPAsLoadedMsg is sent when HPAs are loaded
type HPAsLoadedMsg struct {
	HPAs []k8s.HPAInfo
	Err  error
}

// hpaColumns builds the column list for the HPAs table.
func hpaColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 25, MinWidth: 15, Flexible: true},
		components.Column{Title: "REFERENCE", Width: 25, MinWidth: 12, Flexible: true},
		components.Column{Title: "TARGETS", Width: 20, MinWidth: 10},
		components.Column{Title: "MINPODS", Width: 8, Align: lipgloss.Right},
		components.Column{Title: "MAXPODS", Width: 8, Align: lipgloss.Right},
		components.Column{Title: "REPLICAS", Width: 9, Align: lipgloss.Right},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// HPAsView displays a list of HorizontalPodAutoscalers
type HPAsView struct {
	BaseView
	table   *components.Table
	filter  *components.SearchInput
	client  k8s.Client
	hpas    []k8s.HPAInfo
	showNS  bool
	loading bool
	err     error
	spinner *components.Spinner
}

// NewHPAsView creates a new HPAs view
func NewHPAsView(client k8s.Client) *HPAsView {
	v := &HPAsView{
		table:   components.NewTable(hpaColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading HPAs...")
	v.table.SetEmptyState("⚖", "No HPAs found",
		"No HorizontalPodAutoscalers exist in this namespace", "")
	return v
}

// Init initializes the view
func (v *HPAsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *HPAsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case HPAsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.hpas = msg.HPAs
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

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if row := v.table.SelectedRow(); row != nil {
				for _, hpa := range v.hpas {
					if hpa.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "HorizontalPodAutoscaler",
								Resource:  "horizontalpodautoscalers",
								Namespace: hpa.Namespace,
								Name:      hpa.Name,
								UID:       hpa.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, hpa := range v.hpas {
					if hpa.UID == row.ID {
						hpa := hpa
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "HorizontalPodAutoscaler", Resource: "horizontalpodautoscalers", Namespace: hpa.Namespace,
								Name: hpa.Name, UID: hpa.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, hpa := range v.hpas {
					if hpa.UID == row.ID {
						hpa := hpa
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "HorizontalPodAutoscaler", Resource: "horizontalpodautoscalers", Namespace: hpa.Namespace,
								Name: hpa.Name, UID: hpa.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, hpa := range v.hpas {
					if hpa.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete HPA",
								Message: fmt.Sprintf("Delete HPA %s/%s?", hpa.Namespace, hpa.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "horizontalpodautoscalers", hpa.Namespace, hpa.Name)
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
func (v *HPAsView) View() string {
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

func (v *HPAsView) Name() string           { return "HPAs" }
func (v *HPAsView) IsLoading() bool         { return v.loading }
func (v *HPAsView) RowCount() int           { return v.table.RowCount() }
func (v *HPAsView) IsFilterVisible() bool   { return v.filter.IsVisible() }
func (v *HPAsView) GetTable() *components.Table { return v.table }

func (v *HPAsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up, theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter, theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

func (v *HPAsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

func (v *HPAsView) ResetSelection() { v.table.GotoTop() }

func (v *HPAsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(hpaColumns(newShowNS))
	}
}

func (v *HPAsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

func (v *HPAsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			hpas, err := v.client.ListHPAs(context.Background(), v.namespace)
			return HPAsLoadedMsg{HPAs: hpas, Err: err}
		},
	)
}

func (v *HPAsView) SetClient(client k8s.Client) { v.client = client }

func (v *HPAsView) updateTable() {
	rows := make([]components.Row, len(v.hpas))
	for i, hpa := range v.hpas {
		values := []string{}
		if v.showNS {
			values = append(values, hpa.Namespace)
		}
		values = append(values,
			hpa.Name,
			hpa.Reference,
			hpa.Targets,
			fmt.Sprintf("%d", hpa.MinReplicas),
			fmt.Sprintf("%d", hpa.MaxReplicas),
			fmt.Sprintf("%d", hpa.CurrentReplicas),
			formatAge(hpa.Age),
		)
		rows[i] = components.Row{
			ID:     hpa.UID,
			Values: values,
			Labels: hpa.Labels,
		}
	}
	v.table.SetRows(rows)
}
