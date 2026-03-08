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

// EventsLoadedMsg is sent when events are loaded
type EventsLoadedMsg struct {
	Events []k8s.EventInfo
	Err    error
}

// eventColumns builds the column list for the events table.
// When showNS is true, the NAMESPACE column is prepended.
func eventColumns(showNS bool) []components.Column {
	cols := []components.Column{}
	if showNS {
		cols = append(cols, components.Column{Title: "NAMESPACE", Width: 15})
	}
	cols = append(cols,
		components.Column{Title: "TYPE", Width: 8},
		components.Column{Title: "REASON", Width: 20},
		components.Column{Title: "OBJECT", Width: 30, MinWidth: 15, Flexible: true},
		components.Column{Title: "MESSAGE", Width: 50, MinWidth: 20, Flexible: true},
		components.Column{Title: "COUNT", Width: 6, Align: lipgloss.Right, IsNumeric: true},
		components.Column{Title: "AGE", Width: 8, Align: lipgloss.Right},
	)
	return cols
}

// EventsView displays a list of events
type EventsView struct {
	BaseView
	table   *components.Table
	filter  *components.SearchInput
	client  k8s.Client
	events  []k8s.EventInfo
	showNS  bool
	loading bool
	err     error
	spinner *components.Spinner
}

// NewEventsView creates a new events view
func NewEventsView(client k8s.Client) *EventsView {
	v := &EventsView{
		table:   components.NewTable(eventColumns(true)),
		filter:  components.NewSearchInput(),
		client:  client,
		showNS:  true,
		spinner: components.NewSpinner(),
	}
	v.focused = true
	v.spinner.SetMessage("Loading events...")

	// Set contextual empty state
	v.table.SetEmptyState("📋", "No events",
		"No recent events in this namespace", "")

	return v
}

// Init initializes the view
func (v *EventsView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *EventsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case EventsLoadedMsg:
		v.loading = false
		v.spinner.Hide()
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.events = msg.Events
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
				for _, event := range v.events {
					if event.UID == row.ID {
						if event.ObjectKind != "" && event.ObjectName != "" {
							kind := event.ObjectKind
							name := event.ObjectName
							ns := event.Namespace
							return v, func() tea.Msg {
								return NavigateToResourceMsg{
									Kind:      kind,
									Name:      name,
									Namespace: ns,
								}
							}
						}
						break
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case key.Matches(msg, theme.DefaultKeyMap().Describe):
			if row := v.table.SelectedRow(); row != nil {
				for _, event := range v.events {
					if event.UID == row.ID {
						event := event
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewDescribe,
								Kind:       "Event", Resource: "events", Namespace: event.Namespace,
								Name: event.Name, UID: event.UID,
							}
						}
					}
				}
			}

		case key.Matches(msg, theme.DefaultKeyMap().YAML):
			if row := v.table.SelectedRow(); row != nil {
				for _, event := range v.events {
					if event.UID == row.ID {
						event := event
						return v, func() tea.Msg {
							return OpenViewMsg{
								TargetView: theme.ViewYAML,
								Kind:       "Event", Resource: "events", Namespace: event.Namespace,
								Name: event.Name, UID: event.UID,
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
func (v *EventsView) View() string {
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
func (v *EventsView) Name() string {
	return "Events"
}

// ShortHelp returns keybindings for help
func (v *EventsView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Enter,
		theme.DefaultKeyMap().Filter,
		theme.DefaultKeyMap().Describe,
	}
}

// SetSize sets the view dimensions
func (v *EventsView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	tableHeight := height
	if v.filter.IsVisible() {
		tableHeight -= 2
	}
	v.table.SetSize(width, tableHeight)
	v.filter.SetWidth(width)
}

// ResetSelection resets the table cursor to the top
func (v *EventsView) ResetSelection() {
	v.table.GotoTop()
}

// IsLoading returns whether the view is currently loading data
func (v *EventsView) IsLoading() bool {
	return v.loading
}

// SetNamespace overrides BaseView to toggle the NAMESPACE column
func (v *EventsView) SetNamespace(ns string) {
	v.BaseView.SetNamespace(ns)
	newShowNS := (ns == "")
	if newShowNS != v.showNS {
		v.showNS = newShowNS
		v.table.SetColumns(eventColumns(newShowNS))
	}
}

// SelectedName returns the name of the currently selected resource
func (v *EventsView) SelectedName() string {
	if v.showNS {
		return v.table.SelectedValue(3)
	}
	return v.table.SelectedValue(2)
}

// Refresh refreshes the event list
func (v *EventsView) Refresh() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.spinner.Show(),
		func() tea.Msg {
			events, err := v.client.ListEvents(context.Background(), v.namespace)
			return EventsLoadedMsg{Events: events, Err: err}
		},
	)
}

// SetClient sets a new k8s client
func (v *EventsView) SetClient(client k8s.Client) {
	v.client = client
}

// SelectedEvent returns the currently selected event
func (v *EventsView) SelectedEvent() *k8s.EventInfo {
	if row := v.table.SelectedRow(); row != nil {
		for _, event := range v.events {
			if event.UID == row.ID {
				return &event
			}
		}
	}
	return nil
}

// RowCount returns the number of visible rows
func (v *EventsView) RowCount() int {
	return v.table.RowCount()
}

func (v *EventsView) updateTable() {
	rows := make([]components.Row, len(v.events))
	for i, event := range v.events {
		// Format object as Kind/Name
		object := fmt.Sprintf("%s/%s", event.ObjectKind, event.ObjectName)

		// Truncate message if too long
		message := event.Message
		if len(message) > 60 {
			message = message[:57] + "..."
		}

		// Use event type for status coloring (Normal=green, Warning=yellow)
		status := event.Type

		values := []string{}
		if v.showNS {
			values = append(values, event.Namespace)
		}
		values = append(values,
			event.Type,
			event.Reason,
			object,
			message,
			fmt.Sprintf("%d", event.Count),
			formatAge(event.Age),
		)

		rows[i] = components.Row{
			ID:     event.UID,
			Values: values,
			Status: status,
			Labels: event.Labels,
		}
	}
	v.table.SetRows(rows)
}

// GetTable returns the underlying table component.
func (v *EventsView) GetTable() *components.Table {
	return v.table
}

func (v *EventsView) IsFilterVisible() bool {
	return v.filter.IsVisible()
}
