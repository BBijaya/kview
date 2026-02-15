package components

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/ui/theme"
)

// Tab represents a single tab
type Tab struct {
	ID    string
	Title string
	Badge int // Optional badge count (e.g., for notifications)
}

// Tabs is a tab switcher component
type Tabs struct {
	tabs      []Tab
	activeIdx int
	width     int
	focused   bool
}

// NewTabs creates a new tabs component
func NewTabs(tabs []Tab) *Tabs {
	return &Tabs{
		tabs:      tabs,
		activeIdx: 0,
		width:     80,
		focused:   true,
	}
}

// SetTabs sets the available tabs
func (t *Tabs) SetTabs(tabs []Tab) {
	t.tabs = tabs
	if t.activeIdx >= len(t.tabs) {
		t.activeIdx = max(0, len(t.tabs)-1)
	}
}

// SetWidth sets the tabs width
func (t *Tabs) SetWidth(width int) {
	t.width = width
}

// SetActive sets the active tab by index
func (t *Tabs) SetActive(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeIdx = index
	}
}

// SetActiveByID sets the active tab by ID
func (t *Tabs) SetActiveByID(id string) {
	for i, tab := range t.tabs {
		if tab.ID == id {
			t.activeIdx = i
			return
		}
	}
}

// ActiveTab returns the active tab
func (t *Tabs) ActiveTab() *Tab {
	if len(t.tabs) == 0 {
		return nil
	}
	return &t.tabs[t.activeIdx]
}

// ActiveIndex returns the active tab index
func (t *Tabs) ActiveIndex() int {
	return t.activeIdx
}

// Focus focuses the tabs
func (t *Tabs) Focus() {
	t.focused = true
}

// Blur unfocuses the tabs
func (t *Tabs) Blur() {
	t.focused = false
}

// Next moves to the next tab
func (t *Tabs) Next() {
	if len(t.tabs) == 0 {
		return
	}
	t.activeIdx = (t.activeIdx + 1) % len(t.tabs)
}

// Prev moves to the previous tab
func (t *Tabs) Prev() {
	if len(t.tabs) == 0 {
		return
	}
	t.activeIdx--
	if t.activeIdx < 0 {
		t.activeIdx = len(t.tabs) - 1
	}
}

// Update handles key events
func (t *Tabs) Update(msg tea.Msg) (*Tabs, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().NextTab):
			t.Next()
			return t, func() tea.Msg {
				return theme.SwitchViewMsg{View: theme.ViewType(t.activeIdx)}
			}
		case key.Matches(msg, theme.DefaultKeyMap().PrevTab):
			t.Prev()
			return t, func() tea.Msg {
				return theme.SwitchViewMsg{View: theme.ViewType(t.activeIdx)}
			}
		case key.Matches(msg, theme.DefaultKeyMap().Pods):
			if len(t.tabs) > 0 {
				t.activeIdx = 0
				return t, func() tea.Msg {
					return theme.SwitchViewMsg{View: theme.ViewPods}
				}
			}
		case key.Matches(msg, theme.DefaultKeyMap().Deployments):
			if len(t.tabs) > 1 {
				t.activeIdx = 1
				return t, func() tea.Msg {
					return theme.SwitchViewMsg{View: theme.ViewDeployments}
				}
			}
		case key.Matches(msg, theme.DefaultKeyMap().Services):
			if len(t.tabs) > 2 {
				t.activeIdx = 2
				return t, func() tea.Msg {
					return theme.SwitchViewMsg{View: theme.ViewServices}
				}
			}
		}
	}

	return t, nil
}

// View renders the tabs
func (t *Tabs) View() string {
	var result string
	for i, tab := range t.tabs {
		style := theme.Styles.Tab
		if i == t.activeIdx {
			style = theme.Styles.TabActive
		}

		content := tab.Title
		if tab.Badge > 0 {
			content += " (" + intToStr(tab.Badge) + ")"
		}

		result += style.Render(content)
	}
	return result
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte(n%10) + '0'}, result...)
		n /= 10
	}
	return string(result)
}
