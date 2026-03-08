package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ListItem represents an item in the list
type ListItem struct {
	ID          string
	Title       string
	Description string
	Selected    bool
}

// List is a selectable list component
type List struct {
	items         []ListItem
	filteredItems []ListItem
	cursor        int
	offset        int
	width         int
	height        int
	filter        string
	focused       bool
	multiSelect   bool
	title         string
}

// NewList creates a new list component
func NewList(title string) *List {
	return &List{
		items:         []ListItem{},
		filteredItems: []ListItem{},
		cursor:        0,
		offset:        0,
		width:         40,
		height:        10,
		focused:       true,
		title:         title,
	}
}

// SetSize sets the list dimensions
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// SetItems sets the list items
func (l *List) SetItems(items []ListItem) {
	l.items = items
	l.applyFilter()
	if l.cursor >= len(l.filteredItems) {
		l.cursor = max(0, len(l.filteredItems)-1)
	}
}

// SetFilter sets the filter string
func (l *List) SetFilter(filter string) {
	l.filter = filter
	l.applyFilter()
	l.cursor = 0
	l.offset = 0
}

// SetMultiSelect enables or disables multi-select mode
func (l *List) SetMultiSelect(enabled bool) {
	l.multiSelect = enabled
}

// Focus focuses the list
func (l *List) Focus() {
	l.focused = true
}

// Blur unfocuses the list
func (l *List) Blur() {
	l.focused = false
}

// SelectedItem returns the currently focused item
func (l *List) SelectedItem() *ListItem {
	if len(l.filteredItems) == 0 || l.cursor >= len(l.filteredItems) {
		return nil
	}
	return &l.filteredItems[l.cursor]
}

// SelectedItems returns all selected items (for multi-select)
func (l *List) SelectedItems() []ListItem {
	var selected []ListItem
	for _, item := range l.items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected
}

// Update handles key events
func (l *List) Update(msg tea.Msg) (*List, tea.Cmd) {
	if !l.focused {
		return l, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Up):
			l.moveUp()
		case key.Matches(msg, theme.DefaultKeyMap().Down):
			l.moveDown()
		case key.Matches(msg, theme.DefaultKeyMap().Select):
			if l.multiSelect {
				l.toggleSelect()
			}
		case key.Matches(msg, theme.DefaultKeyMap().Home):
			l.goToTop()
		case key.Matches(msg, theme.DefaultKeyMap().End):
			l.goToBottom()
		}
	}

	return l, nil
}

// View renders the list
func (l *List) View() string {
	var b strings.Builder

	// Title
	if l.title != "" {
		titleStyle := theme.Styles.PanelTitle
		b.WriteString(titleStyle.Render(l.title))
		b.WriteString("\n")
	}

	// Calculate visible items
	titleLines := 0
	if l.title != "" {
		titleLines = 1
	}
	visibleItems := l.height - titleLines
	if visibleItems < 1 {
		visibleItems = 1
	}

	// Adjust offset
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+visibleItems {
		l.offset = l.cursor - visibleItems + 1
	}

	// Render items
	if len(l.filteredItems) == 0 {
		b.WriteString(theme.Styles.StatusUnknown.Render("No items"))
	} else {
		endIdx := min(l.offset+visibleItems, len(l.filteredItems))
		for i := l.offset; i < endIdx; i++ {
			item := l.filteredItems[i]

			style := theme.Styles.PaletteItem
			if i == l.cursor && l.focused {
				style = theme.Styles.PaletteSelected
			}

			// Selection indicator for multi-select
			prefix := "  "
			if l.multiSelect {
				if item.Selected {
					prefix = "[x] "
				} else {
					prefix = "[ ] "
				}
			}

			content := prefix + item.Title
			if item.Description != "" && l.width > 30 {
				desc := theme.Styles.StatusUnknown.Render(" " + item.Description)
				content = prefix + item.Title + desc
			}

			// Truncate to width
			content = theme.TruncateString(content, l.width-2)
			b.WriteString(style.Width(l.width).Render(content))

			if i < endIdx-1 {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func (l *List) moveUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

func (l *List) moveDown() {
	if l.cursor < len(l.filteredItems)-1 {
		l.cursor++
	}
}

func (l *List) goToTop() {
	l.cursor = 0
	l.offset = 0
}

func (l *List) goToBottom() {
	if len(l.filteredItems) > 0 {
		l.cursor = len(l.filteredItems) - 1
	}
}

func (l *List) toggleSelect() {
	if l.cursor < len(l.filteredItems) {
		// Find the original item and toggle
		targetID := l.filteredItems[l.cursor].ID
		for i := range l.items {
			if l.items[i].ID == targetID {
				l.items[i].Selected = !l.items[i].Selected
				break
			}
		}
		// Update filtered items
		l.filteredItems[l.cursor].Selected = !l.filteredItems[l.cursor].Selected
	}
}

func (l *List) applyFilter() {
	if l.filter == "" {
		l.filteredItems = make([]ListItem, len(l.items))
		copy(l.filteredItems, l.items)
		return
	}

	filter := strings.ToLower(l.filter)
	l.filteredItems = make([]ListItem, 0, len(l.items))
	for _, item := range l.items {
		if strings.Contains(strings.ToLower(item.Title), filter) ||
			strings.Contains(strings.ToLower(item.Description), filter) {
			l.filteredItems = append(l.filteredItems, item)
		}
	}
}
