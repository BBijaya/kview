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

// SecretsLoadedMsg is sent when secrets are loaded
type SecretsLoadedMsg struct {
	Secrets []k8s.SecretInfo
	Err     error
}

// secretColumns builds the column list for the secrets table.
// When showNS is true, the NAMESPACE column is prepended.
func secretColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "NAME", Width: 35, MinWidth: 20, Flexible: true},
		components.Column{Title: "TYPE", Width: 30},
		components.Column{Title: "DATA", Width: 6, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// SecretsView displays a list of secrets
type SecretsView struct {
	BaseView
	table   *components.Table
	filter  *components.SearchInput
	client  k8s.Client
	secrets []k8s.SecretInfo
	showNS  bool
	loading bool
	err     error
	spinner *components.Spinner
}

// NewSecretsView creates a new secrets view
func NewSecretsView(client k8s.Client) *SecretsView {
	v := &SecretsView{
		table:   components.NewTable(secretColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading secrets...")

	// Set contextual empty state
	v.table.SetEmptyState("🔐", "No secrets found",
		"No secrets exist in this namespace", "")

	return v
}

// Init initializes the view
func (v *SecretsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *SecretsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case SecretsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.secrets = msg.Secrets
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
				for _, secret := range v.secrets {
					if secret.UID == row.ID {
						return v, func() tea.Msg {
							return ResourceSelectedMsg{
								Kind:      "Secret",
								Resource:  "secrets",
								Namespace: secret.Namespace,
								Name:      secret.Name,
								UID:       secret.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, secret := range v.secrets {
					if secret.UID == row.ID {
						secret := secret
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Secret", Resource: "secrets", Namespace: secret.Namespace,
								Name: secret.Name, UID: secret.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, secret := range v.secrets {
					if secret.UID == row.ID {
						secret := secret
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Secret", Resource: "secrets", Namespace: secret.Namespace,
								Name: secret.Name, UID: secret.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().DecodeSecret):
			if row := v.table.SelectedRow(); row != nil {
				for _, secret := range v.secrets {
					if secret.UID == row.ID {
						secret := secret
						return v, func() tea.Msg {
							return DecodeSecretMsg{
								Namespace: secret.Namespace,
								Name:      secret.Name,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Delete):
			if row := v.table.SelectedRow(); row != nil {
				for _, secret := range v.secrets {
					if secret.UID == row.ID {
						return v, func() tea.Msg {
							return ConfirmActionMsg{
								Title:   "Delete Secret",
								Message: fmt.Sprintf("Delete secret %s/%s?", secret.Namespace, secret.Name),
								Action: func() error {
									return v.client.Delete(context.Background(), "secrets", secret.Namespace, secret.Name)
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
func (v *SecretsView) View() string {
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
func (v *SecretsView) Name() string {
	return "Secrets"
}

// ShortHelp returns keybindings for help
func (v *SecretsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *SecretsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *SecretsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *SecretsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *SecretsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(secretColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *SecretsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(1)
	}
	return v.table.SelectedValue(0)
}

// Refresh refreshes the secret list
func (v *SecretsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			secrets, err := v.client.ListSecrets(context.Background(), v.namespace)
			return SecretsLoadedMsg{Secrets: secrets, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *SecretsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedSecret returns the currently selected secret
func (v *SecretsView) SelectedSecret() *k8s.SecretInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, secret := range v.secrets {
			if secret.UID == row.ID {
				return &secret
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *SecretsView) RowCount() int {
	return v.table.RowCount()
}

func (v *SecretsView) updateTable() {
	rows := make([]components.Row, len(v.secrets))
	for i, secret := range v.secrets {
		values := []string{}
		if v.showNS {
			values = append(values, secret.Namespace)
		}
		values = append(values,
			secret.Name,
			secret.Type,
			fmt.Sprintf("%d", secret.DataCount),
			formatAge(secret.Age),
		)
		rows[i] = components.Row{
			ID:     secret.UID,
			Values: values,
			Status: "Active",
			Labels: secret.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *SecretsView) GetTable() *components.Table {
	return v.table
}

func (v *SecretsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
