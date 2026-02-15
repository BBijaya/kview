package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/store"
	"github.com/bijaya/kview/internal/ui/theme"
)

// TimelineLoadedMsg is sent when timeline events are loaded
type TimelineLoadedMsg struct {
	Events  []store.Event
	Changes []store.ChangeRecord
	Err     error
}

// TimelineView displays an event timeline
type TimelineView struct {
	BaseView
	viewport viewport.Model
	store    store.Store
	events   []store.Event
	changes  []store.ChangeRecord
	loading  bool
	err      error
	timeRange time.Duration
}

// NewTimelineView creates a new timeline view
func NewTimelineView(s store.Store) *TimelineView {
	vp := viewport.New(80, 20)
	vp.Style = theme.Styles.Base

	return &TimelineView{
		viewport:  vp,
		store:     s,
		timeRange: 1 * time.Hour, // Default to last hour
	}
}

// Init initializes the view
func (v *TimelineView) Init() tea.Cmd {
	return v.Refresh()
}

// Update handles messages
func (v *TimelineView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TimelineLoadedMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			v.events = msg.Events
			v.changes = msg.Changes
			v.updateContent()
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Escape):
			return v, func() tea.Msg {
				return theme.SwitchViewMsg{View: theme.ViewPods}
			}

		case key.Matches(msg, theme.DefaultKeyMap().Refresh):
			return v, v.Refresh()

		case msg.String() == "1":
			v.timeRange = 1 * time.Hour
			return v, v.Refresh()

		case msg.String() == "6":
			v.timeRange = 6 * time.Hour
			return v, v.Refresh()

		case msg.String() == "d":
			v.timeRange = 24 * time.Hour
			return v, v.Refresh()

		case msg.String() == "w":
			v.timeRange = 7 * 24 * time.Hour
			return v, v.Refresh()

		case msg.String() == "G":
			v.viewport.GotoBottom()

		case msg.String() == "g":
			v.viewport.GotoTop()
		}
	}

	// Update viewport
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *TimelineView) View() string {
	var b strings.Builder

	// Header with time range selector
	b.WriteString(theme.Styles.PanelTitle.Render("Event Timeline"))
	b.WriteString(" │ ")
	b.WriteString(v.renderTimeRangeSelector())
	b.WriteString("\n\n")

	if v.loading {
		b.WriteString("Loading timeline...")
		return b.String()
	}

	if v.err != nil {
		b.WriteString(theme.Styles.StatusError.Render("Error: " + v.err.Error()))
		return b.String()
	}

	if len(v.events) == 0 && len(v.changes) == 0 {
		b.WriteString(theme.Styles.StatusUnknown.Render("No events in the selected time range"))
		return b.String()
	}

	b.WriteString(v.viewport.View())
	b.WriteString("\n")
	b.WriteString(theme.Styles.Help.Render("↑↓/pgup/pgdn scroll • g/G top/bottom • 1/6/d/w time range • esc back"))

	return b.String()
}

func (v *TimelineView) renderTimeRangeSelector() string {
	ranges := []struct {
		key   string
		label string
		dur   time.Duration
	}{
		{"1", "1h", 1 * time.Hour},
		{"6", "6h", 6 * time.Hour},
		{"d", "24h", 24 * time.Hour},
		{"w", "7d", 7 * 24 * time.Hour},
	}

	var parts []string
	for _, r := range ranges {
		style := theme.Styles.Tab
		if r.dur == v.timeRange {
			style = theme.Styles.TabActive
		}
		parts = append(parts, style.Render(fmt.Sprintf("[%s] %s", r.key, r.label)))
	}

	return strings.Join(parts, " ")
}

func (v *TimelineView) updateContent() {
	// Merge events and changes into a unified timeline
	items := v.buildTimelineItems()

	var b strings.Builder
	for _, item := range items {
		b.WriteString(v.renderTimelineItem(item))
		b.WriteString("\n")
	}

	v.viewport.SetContent(b.String())
}

type timelineItem struct {
	Timestamp time.Time
	Type      string // "event", "change"
	Icon      string
	Namespace string
	Resource  string
	Message   string
	Severity  string // "normal", "warning", "created", "updated", "deleted"
}

func (v *TimelineView) buildTimelineItems() []timelineItem {
	var items []timelineItem

	// Add events
	for _, e := range v.events {
		severity := "normal"
		icon := "•"
		if e.Type == "Warning" {
			severity = "warning"
			icon = "⚠"
		}

		items = append(items, timelineItem{
			Timestamp: e.LastSeen,
			Type:      "event",
			Icon:      icon,
			Namespace: e.Namespace,
			Resource:  fmt.Sprintf("%s/%s", e.ResourceKind, e.ResourceName),
			Message:   fmt.Sprintf("[%s] %s", e.Reason, e.Message),
			Severity:  severity,
		})
	}

	// Add changes
	for _, c := range v.changes {
		icon := "→"
		severity := "updated"
		switch c.ChangeType {
		case store.ChangeTypeCreated:
			icon = "+"
			severity = "created"
		case store.ChangeTypeDeleted:
			icon = "-"
			severity = "deleted"
		}

		items = append(items, timelineItem{
			Timestamp: c.Timestamp,
			Type:      "change",
			Icon:      icon,
			Namespace: c.Namespace,
			Resource:  fmt.Sprintf("%s/%s", c.ResourceKind, c.ResourceName),
			Message:   c.ChangeType,
			Severity:  severity,
		})
	}

	// Sort by timestamp (newest first)
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Timestamp.After(items[i].Timestamp) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	return items
}

func (v *TimelineView) renderTimelineItem(item timelineItem) string {
	var style = theme.Styles.Base

	switch item.Severity {
	case "warning":
		style = theme.Styles.StatusWarning
	case "created":
		style = theme.Styles.StatusHealthy
	case "deleted":
		style = theme.Styles.StatusError
	}

	timestamp := item.Timestamp.Format("15:04:05")
	if time.Since(item.Timestamp) > 24*time.Hour {
		timestamp = item.Timestamp.Format("Jan 02 15:04")
	}

	return fmt.Sprintf("%s %s %s %s %s",
		theme.Styles.StatusUnknown.Render(timestamp),
		style.Render(item.Icon),
		theme.Styles.Focused.Render(item.Resource),
		style.Render(item.Message),
		theme.Styles.StatusUnknown.Render("("+item.Namespace+")"),
	)
}

// Name returns the view name
func (v *TimelineView) Name() string {
	return "Timeline"
}

// ShortHelp returns keybindings for help
func (v *TimelineView) ShortHelp() []key.Binding {
	return []key.Binding{
		theme.DefaultKeyMap().Up,
		theme.DefaultKeyMap().Down,
		theme.DefaultKeyMap().Refresh,
		theme.DefaultKeyMap().Escape,
	}
}

// SetSize sets the view dimensions
func (v *TimelineView) SetSize(width, height int) {
	v.BaseView.SetSize(width, height)
	v.viewport.Width = width
	v.viewport.Height = height - 5 // Account for header and footer
}

// IsLoading returns whether the view is currently loading data
func (v *TimelineView) IsLoading() bool {
	return v.loading
}

// Refresh loads timeline data
func (v *TimelineView) Refresh() tea.Cmd {
	if v.store == nil {
		return func() tea.Msg {
			return TimelineLoadedMsg{
				Events:  nil,
				Changes: nil,
				Err:     fmt.Errorf("no store configured"),
			}
		}
	}

	v.loading = true
	return func() tea.Msg {
		// This would be implemented with actual store queries
		// For now, return empty
		return TimelineLoadedMsg{
			Events:  nil,
			Changes: nil,
		}
	}
}
