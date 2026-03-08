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

// DrillDownHelmHistoryMsg requests drill-down from a release to its revision history.
type DrillDownHelmHistoryMsg struct {
	ReleaseName string
	Namespace   string
}

// HelmHistoryLoadedMsg is sent when Helm release history is loaded
type HelmHistoryLoadedMsg struct {
	Releases []k8s.HelmReleaseInfo
	Err      error
}

// helmHistoryColumns builds the column list for the Helm History table.
func helmHistoryColumns() []components.Column {
	return []components.Column{
		{Title: "REVISION", Width: 10, Align: lipgloss.Right},
		{Title: "CHART", Width: 20, MinWidth: 10, Flexible: true},
		{Title: "APP VERSION", Width: 14, MinWidth: 8},
		{Title: "STATUS", Width: 16, MinWidth: 8},
		{Title: "AGE", Width: 8, Align: lipgloss.Right},
	}
}

// HelmHistoryView displays all revisions for a specific Helm release
type HelmHistoryView struct {
	BaseView
	table       *components.Table
	filter      *components.SearchInput
	client      k8s.Client
	releaseName string
	revisions   []k8s.HelmReleaseInfo
	loading     bool
	err         error
	spinner     *components.Spinner
}

// NewHelmHistoryView creates a new Helm History view
func NewHelmHistoryView(client k8s.Client) *HelmHistoryView {
	v := &HelmHistoryView{
		table:   components.NewTable(helmHistoryColumns()),
		filter:  components.NewSearchInput(),
		client:  client,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading release history...")
	v.table.SetEmptyState("⎈", "No revisions found",
		"No revisions exist for this release", "")
	return v
}

// SetRelease configures the view for a specific release.
func (v *HelmHistoryView) SetRelease(namespace, releaseName string) {
	v.namespace = namespace
	v.releaseName = releaseName
}

// Init initializes the view
func (v *HelmHistoryView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *HelmHistoryView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case HelmHistoryLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.revisions = msg.Releases
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

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if row := v.table.SelectedRow(); row != nil {
				for _, rev := range v.revisions {
					if rev.UID == row.ID {
						rev := rev
						secretName := helmSecretName(rev.Name, rev.Revision)
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Secret", Resource: "secrets", Namespace: rev.Namespace,
								Name: secretName, UID: rev.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, rev := range v.revisions {
					if rev.UID == row.ID {
						rev := rev
						secretName := helmSecretName(rev.Name, rev.Revision)
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Secret", Resource: "secrets", Namespace: rev.Namespace,
								Name: secretName, UID: rev.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, rev := range v.revisions {
					if rev.UID == row.ID {
						rev := rev
						secretName := helmSecretName(rev.Name, rev.Revision)
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Secret", Resource: "secrets", Namespace: rev.Namespace,
								Name: secretName, UID: rev.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().HelmValues):
			if row := v.table.SelectedRow(); row != nil {
				for _, rev := range v.revisions {
					if rev.UID == row.ID {
						rev := rev
						return v, func() tea.Msg {
							return OpenHelmContentMsg{
								Mode:        HelmContentValues,
								ReleaseName: rev.Name,
								Namespace:   rev.Namespace,
								Revision:    rev.Revision,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().HelmManifest):
			if row := v.table.SelectedRow(); row != nil {
				for _, rev := range v.revisions {
					if rev.UID == row.ID {
						rev := rev
						return v, func() tea.Msg {
							return OpenHelmContentMsg{
								Mode:        HelmContentManifest,
								ReleaseName: rev.Name,
								Namespace:   rev.Namespace,
								Revision:    rev.Revision,
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
func (v *HelmHistoryView) View() string {
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

func (v *HelmHistoryView) Name() string               { return "Helm History" }
func (v *HelmHistoryView) IsLoading() bool             { return v.loading }
func (v *HelmHistoryView) RowCount() int               { return v.table.RowCount() }
func (v *HelmHistoryView) IsFilterVisible() bool       { return v.filter.IsVisible() }
func (v *HelmHistoryView) GetTable() *components.Table { return v.table }

func (v *HelmHistoryView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up, theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
		theme.DefaultKeyMap().Escape,
	}
}

func (v *HelmHistoryView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

func (v *HelmHistoryView) ResetSelection() { v.table.GotoTop() }

func (v *HelmHistoryView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
}

func (v *HelmHistoryView) SelectedName() string {
	return v.table.SelectedValue(0)
}

func (v *HelmHistoryView) Refresh() tea.Cmd {
	v.loading = true
	releaseName := v.releaseName
	ns := v.namespace
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			releases, err := v.client.ListHelmReleaseHistory(context.Background(), ns, releaseName)
			return HelmHistoryLoadedMsg{Releases: releases, Err: err}
		},
	)
}

func (v *HelmHistoryView) SetClient(client k8s.Client) { v.client = client }

func (v *HelmHistoryView) updateTable() {
	rows := make([]components.Row, len(v.revisions))
	for i, rev := range v.revisions {
		chart := rev.Chart
		if chart == "" {
			chart = "-"
		}
		if rev.ChartVersion != "" {
			chart = chart + "-" + rev.ChartVersion
		}

		appVersion := rev.AppVersion
		if appVersion == "" {
			appVersion = "-"
		}

		status := rev.Status
		if status == "" {
			status = "-"
		}

		revision := fmt.Sprintf("%d", rev.Revision)

		values := []string{
			revision,
			chart,
			appVersion,
			status,
			formatAge(rev.Age),
		}

		rowStatus := helmStatusToRowStatus(rev.Status)

		rows[i] = components.Row{
			ID:     rev.UID,
			Values: values,
			Status: rowStatus,
		}
	}
	v.table.SetRows(rows)
}
