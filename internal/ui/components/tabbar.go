package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// TabBar displays horizontal numbered tabs
type TabBar struct {
	width     int
	items     []string
	activeIdx int
}

// NewTabBar creates a new tab bar component
func NewTabBar(items []string) *TabBar {
	return &TabBar{
		width:     80,
		items:     items,
		activeIdx: 0,
	}
}

// SetWidth sets the tab bar width
func (t *TabBar) SetWidth(width int) {
	t.width = width
}

// SetItems sets the tab items
func (t *TabBar) SetItems(items []string) {
	t.items = items
	if t.activeIdx >= len(items) {
		t.activeIdx = 0
	}
}

// SetActive sets the active tab index
func (t *TabBar) SetActive(idx int) {
	if idx >= 0 && idx < len(t.items) {
		t.activeIdx = idx
	}
}

// ActiveIndex returns the active tab index
func (t *TabBar) ActiveIndex() int {
	return t.activeIdx
}

// Next moves to the next tab
func (t *TabBar) Next() {
	if len(t.items) > 0 {
		t.activeIdx = (t.activeIdx + 1) % len(t.items)
	}
}

// Prev moves to the previous tab
func (t *TabBar) Prev() {
	if len(t.items) > 0 {
		t.activeIdx = (t.activeIdx - 1 + len(t.items)) % len(t.items)
	}
}

// View renders the tab bar
// Format: [1] Pods  [2] Deployments  [3] Services
func (t *TabBar) View() string {
	if len(t.items) == 0 {
		return ""
	}

	var parts []string
	for i, item := range t.items {
		number := fmt.Sprintf("[%d]", i+1)
		numberStyle := theme.Styles.TabBarNumber

		var itemStyle lipgloss.Style
		if i == t.activeIdx {
			itemStyle = theme.Styles.TabBarItemActive
		} else {
			itemStyle = theme.Styles.TabBarItem
		}

		tab := numberStyle.Render(number) + " " + itemStyle.Render(item)
		parts = append(parts, tab)
	}

	content := strings.Join(parts, "  ")

	// Pad to full width
	contentWidth := lipgloss.Width(content)
	if contentWidth < t.width {
		content = content + strings.Repeat(" ", t.width-contentWidth)
	}

	return theme.Styles.TabBar.Width(t.width).Render(content)
}
