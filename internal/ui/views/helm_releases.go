package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/components"
	"github.com/bijaya/kview/internal/ui/theme"
)

// HelmReleasesLoadedMsg is sent when Helm releases are loaded
type HelmReleasesLoadedMsg struct {
	Releases []k8s.HelmReleaseInfo
	Err      error
}

// helmReleaseColumns builds the column list for the Helm Releases table.
func helmReleaseColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 25, MinWidth: 12, Flexible: true},
		components.Column{Title: "CHART", Width: 20, MinWidth: 10, Flexible: true},
		components.Column{Title: "APP VERSION", Width: 14, MinWidth: 8},
		components.Column{Title: "STATUS", Width: 14, MinWidth: 8},
		components.Column{Title: "REVISION", Width: 10, Align: lipgloss.Right},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// HelmReleasesView displays a list of Helm releases
type HelmReleasesView struct {
	BaseView
	table    *components.Table
	filter   *components.SearchInput
	client   k8s.Client
	releases []k8s.HelmReleaseInfo
	showNS   bool
	loading  bool
	err      error
	spinner  *components.Spinner
}

// NewHelmReleasesView creates a new Helm Releases view
func NewHelmReleasesView(client k8s.Client) *HelmReleasesView {
	v := &HelmReleasesView{
		table:   components.NewTable(helmReleaseColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading Helm Releases...")
	v.table.SetEmptyState("⎈", "No Helm releases found",
		"No Helm releases exist in this namespace", "")
	return v
}

// Init initializes the view
func (v *HelmReleasesView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *HelmReleasesView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case HelmReleasesLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.releases = msg.Releases
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

		case key.Matches(msg, theme.DefaultKeyMap().Enter):
			if row := v.table.SelectedRow(); row != nil {
				for _, rel := range v.releases {
					if rel.UID == row.ID {
						rel := rel
						return v, func() tea.Msg {
							return DrillDownHelmHistoryMsg{
								ReleaseName: rel.Name,
								Namespace:   rel.Namespace,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, rel := range v.releases {
					if rel.UID == row.ID {
						rel := rel
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Secret", Resource: "secrets", Namespace: rel.Namespace,
								Name: helmSecretName(rel.Name, rel.Revision), UID: rel.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, rel := range v.releases {
					if rel.UID == row.ID {
						rel := rel
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Secret", Resource: "secrets", Namespace: rel.Namespace,
								Name: helmSecretName(rel.Name, rel.Revision), UID: rel.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().HelmValues):
			if row := v.table.SelectedRow(); row != nil {
				for _, rel := range v.releases {
					if rel.UID == row.ID {
						rel := rel
						return v, func() tea.Msg {
							return OpenHelmContentMsg{
								Mode:        HelmContentValues,
								ReleaseName: rel.Name,
								Namespace:   rel.Namespace,
								Revision:    rel.Revision,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().HelmManifest):
			if row := v.table.SelectedRow(); row != nil {
				for _, rel := range v.releases {
					if rel.UID == row.ID {
						rel := rel
						return v, func() tea.Msg {
							return OpenHelmContentMsg{
								Mode:        HelmContentManifest,
								ReleaseName: rel.Name,
								Namespace:   rel.Namespace,
								Revision:    rel.Revision,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, rel := range v.releases {
					if rel.UID == row.ID {
						secretName := helmSecretName(rel.Name, rel.Revision)
						ns := rel.Namespace
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Helm Release",
								Message: fmt.Sprintf("Delete Helm release %s/%s (Secret: %s)?", ns, rel.Name, secretName),
								Action: func() error {
									return v.client.Delete(context.Background(), "secrets", ns, secretName)
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
func (v *HelmReleasesView) View() string {
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

func (v *HelmReleasesView) Name() string               { return "Helm Releases" }
func (v *HelmReleasesView) IsLoading() bool             { return v.loading }
func (v *HelmReleasesView) RowCount() int               { return v.table.RowCount() }
func (v *HelmReleasesView) IsFilterVisible() bool       { return v.filter.IsVisible() }
func (v *HelmReleasesView) GetTable() *components.Table { return v.table }

func (v *HelmReleasesView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up, theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter, theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

func (v *HelmReleasesView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

func (v *HelmReleasesView) ResetSelection() { v.table.GotoTop() }

func (v *HelmReleasesView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(helmReleaseColumns(newShowNS))
	}
}

func (v *HelmReleasesView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

func (v *HelmReleasesView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			releases, err := v.client.ListHelmReleases(context.Background(), v.namespace)
			return HelmReleasesLoadedMsg{Releases: releases, Err: err}
		},
	)
}

func (v *HelmReleasesView) SetClient(client k8s.Client) { v.client = client }

func (v *HelmReleasesView) updateTable() {
	rows := make([]components.Row, len(v.releases))
	for i, rel := range v.releases {
		chart := rel.Chart
		if chart == "" {
			chart = "-"
		}
		if rel.ChartVersion != "" {
			chart = chart + "-" + rel.ChartVersion
		}

		appVersion := rel.AppVersion
		if appVersion == "" {
			appVersion = "-"
		}

		status := rel.Status
		if status == "" {
			status = "-"
		}

		revision := fmt.Sprintf("%d", rel.Revision)

		values := []string{}
		if v.showNS {
			values = append(values, rel.Namespace)
		}
		values = append(values,
			rel.Name,
			chart,
			appVersion,
			status,
			revision,
			formatAge(rel.Age),
		)

		// Style status for row coloring
		rowStatus := helmStatusToRowStatus(rel.Status)

		rows[i] = components.Row{
			ID:     rel.UID,
			Values: values,
			Status: rowStatus,
		}
	}
	v.table.SetRows(rows)
}

// helmStatusToRowStatus maps Helm release status to table row status for styling.
func helmStatusToRowStatus(status string) string {
	switch strings.ToLower(status) {
	case "deployed":
		return "Running" // green
	case "failed":
		return "Failed" // red
	case "superseded":
		return "Completed" // muted
	case "pending-install", "pending-upgrade", "pending-rollback":
		return "Pending" // yellow
	case "uninstalling":
		return "Terminating" // warning
	default:
		return ""
	}
}

// helmSecretName returns the Helm 3 Secret name for a release.
// Format: sh.helm.release.v1.<name>.v<revision>
func helmSecretName(releaseName string, revision int) string {
	return fmt.Sprintf("sh.helm.release.v1.%s.v%d", releaseName, revision)
}
