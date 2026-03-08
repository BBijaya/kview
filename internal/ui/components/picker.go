package components

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// PickerItem represents an item in the picker
type PickerItem struct {
	ID    string
	Label string
	Desc  string
}

// PickerSelectedMsg is sent when an item is selected
type PickerSelectedMsg struct {
	PickerID string
	Item     PickerItem
}

// PickerCancelledMsg is sent when the picker is cancelled
type PickerCancelledMsg struct {
	PickerID string
}

// Picker is a modal list picker with loading state, filtering, and navigation
type Picker struct {
	id       string
	title    string
	visible  bool
	loading  bool
	items    []PickerItem
	filtered []PickerItem
	selected int
	width    int
	height   int

	// Filter input
	filterInput textinput.Model
	filtering   bool

	// Loading spinner
	spinner spinner.Model

	// Max visible items
	maxVisible int
	scrollTop  int
}

// NewPicker creates a new picker
func NewPicker(id, title string) *Picker {
	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.CharLimit = 50
	ti.SetWidth(40)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	return &Picker{
		id:          id,
		title:       title,
		visible:     false,
		loading:     false,
		items:       []PickerItem{},
		filtered:    []PickerItem{},
		selected:    0,
		filterInput: ti,
		filtering:   false,
		spinner:     s,
		maxVisible:  10,
		scrollTop:   0,
	}
}

// Show makes the picker visible
func (p *Picker) Show() tea.Cmd {
	p.visible = true
	p.selected = 0
	p.scrollTop = 0
	p.filterInput.SetValue("")
	p.filtering = false
	p.updateFiltered()
	return nil
}

// ShowLoading shows the picker in loading state
func (p *Picker) ShowLoading() tea.Cmd {
	p.visible = true
	p.loading = true
	p.selected = 0
	p.scrollTop = 0
	p.filterInput.SetValue("")
	p.filtering = false
	return p.spinner.Tick
}

// Hide hides the picker
func (p *Picker) Hide() {
	p.visible = false
	p.loading = false
	p.filtering = false
}

// IsVisible returns whether the picker is visible
func (p *Picker) IsVisible() bool {
	return p.visible
}

// IsLoading returns whether the picker is in loading state
func (p *Picker) IsLoading() bool {
	return p.loading
}

// SetItems sets the picker items and clears loading state
func (p *Picker) SetItems(items []PickerItem) {
	p.items = items
	p.loading = false
	p.selected = 0
	p.scrollTop = 0
	p.updateFiltered()
}

// SetSize sets the picker size
func (p *Picker) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.maxVisible = height/2 - 4
	if p.maxVisible < 5 {
		p.maxVisible = 5
	}
}

// GetID returns the picker ID
func (p *Picker) GetID() string {
	return p.id
}

// updateFiltered updates the filtered items based on filter input
func (p *Picker) updateFiltered() {
	filter := strings.ToLower(p.filterInput.Value())
	if filter == "" {
		p.filtered = p.items
		return
	}

	p.filtered = nil
	for _, item := range p.items {
		if strings.Contains(strings.ToLower(item.Label), filter) ||
			strings.Contains(strings.ToLower(item.ID), filter) ||
			strings.Contains(strings.ToLower(item.Desc), filter) {
			p.filtered = append(p.filtered, item)
		}
	}

	// Reset selection if out of bounds
	if p.selected >= len(p.filtered) {
		p.selected = len(p.filtered) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
}

// ensureVisible ensures the selected item is visible
func (p *Picker) ensureVisible() {
	if p.selected < p.scrollTop {
		p.scrollTop = p.selected
	}
	if p.selected >= p.scrollTop+p.maxVisible {
		p.scrollTop = p.selected - p.maxVisible + 1
	}
}

// Update handles picker messages
func (p *Picker) Update(msg tea.Msg) (*Picker, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	var cmds []tea.Cmd

	// Handle spinner updates when loading
	if p.loading {
		switch msg := msg.(type) {
		case spinner.TickMsg:
			var cmd tea.Cmd
			p.spinner, cmd = p.spinner.Update(msg)
			return p, cmd
		case tea.KeyPressMsg:
			if msg.String() == "esc" {
				p.Hide()
				return p, func() tea.Msg {
					return PickerCancelledMsg{PickerID: p.id}
				}
			}
		}
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if p.filtering {
			switch msg.String() {
			case "esc":
				p.filtering = false
				p.filterInput.Blur()
				return p, nil
			case "enter":
				p.filtering = false
				p.filterInput.Blur()
				return p, nil
			default:
				var cmd tea.Cmd
				p.filterInput, cmd = p.filterInput.Update(msg)
				p.updateFiltered()
				cmds = append(cmds, cmd)
				return p, tea.Batch(cmds...)
			}
		}

		switch msg.String() {
		case "esc", "q":
			p.Hide()
			return p, func() tea.Msg {
				return PickerCancelledMsg{PickerID: p.id}
			}

		case "enter":
			if len(p.filtered) > 0 && p.selected < len(p.filtered) {
				selectedItem := p.filtered[p.selected]
				p.Hide()
				return p, func() tea.Msg {
					return PickerSelectedMsg{
						PickerID: p.id,
						Item:     selectedItem,
					}
				}
			}

		case "up", "k":
			if p.selected > 0 {
				p.selected--
				p.ensureVisible()
			}

		case "down", "j":
			if p.selected < len(p.filtered)-1 {
				p.selected++
				p.ensureVisible()
			}

		case "home", "g":
			p.selected = 0
			p.scrollTop = 0

		case "end", "G":
			p.selected = len(p.filtered) - 1
			p.ensureVisible()

		case "/":
			p.filtering = true
			p.filterInput.Focus()
			return p, textinput.Blink
		}
	}

	return p, tea.Batch(cmds...)
}

// View renders the picker
func (p *Picker) View() string {
	if !p.visible {
		return ""
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Bold(true).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorPrimary).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted)

	indicatorStyle := lipgloss.NewStyle().
		Foreground(theme.ColorHighlight).
		Bold(true)

	// Build content
	var content strings.Builder

	content.WriteString(titleStyle.Render(p.title))
	content.WriteString("\n")

	if p.loading {
		content.WriteString("\n")
		content.WriteString(p.spinner.View())
		content.WriteString(" Loading...")
		content.WriteString("\n")
	} else {
		// Filter input
		if p.filtering {
			content.WriteString(theme.Styles.CommandPrefix.Render("/"))
			content.WriteString(p.filterInput.View())
			content.WriteString("\n\n")
		} else if p.filterInput.Value() != "" {
			content.WriteString(lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("Filter: "))
			content.WriteString(p.filterInput.Value())
			content.WriteString("\n\n")
		} else {
			content.WriteString("\n")
		}

		if len(p.filtered) == 0 {
			content.WriteString(lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("  No items found"))
			content.WriteString("\n")
		} else {
			// Show visible items
			end := p.scrollTop + p.maxVisible
			if end > len(p.filtered) {
				end = len(p.filtered)
			}

			for i := p.scrollTop; i < end; i++ {
				item := p.filtered[i]
				isSelected := i == p.selected

				var line string
				if isSelected {
					line = indicatorStyle.Render("► ") + selectedStyle.Render(item.Label)
				} else {
					line = "  " + itemStyle.Render(item.Label)
				}

				if item.Desc != "" {
					line += " " + descStyle.Render(item.Desc)
				}

				content.WriteString(line)
				content.WriteString("\n")
			}

			// Scroll indicator
			if len(p.filtered) > p.maxVisible {
				scrollInfo := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(
					strings.Repeat(" ", 2) + "[" + strconv.Itoa(p.selected+1) + "/" + strconv.Itoa(len(p.filtered)) + "]",
				)
				content.WriteString("\n")
				content.WriteString(scrollInfo)
			}
		}
	}

	content.WriteString("\n")
	helpText := "↑↓ navigate • enter select • / filter • esc cancel"
	content.WriteString(lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(helpText))

	// Calculate overlay size
	overlayWidth := 50
	if p.width > 0 && overlayWidth > p.width-4 {
		overlayWidth = p.width - 4
	}

	// Create overlay box
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 2).
		Width(overlayWidth)

	overlay := overlayStyle.Render(content.String())

	// Center the overlay
	overlayActualWidth := lipgloss.Width(overlay)
	overlayActualHeight := lipgloss.Height(overlay)

	paddingLeft := (p.width - overlayActualWidth) / 2
	paddingTop := (p.height - overlayActualHeight) / 2

	if paddingLeft < 0 {
		paddingLeft = 0
	}
	if paddingTop < 0 {
		paddingTop = 0
	}

	// Build lines with centering
	var result strings.Builder
	for i := 0; i < paddingTop; i++ {
		result.WriteString("\n")
	}

	overlayLines := strings.Split(overlay, "\n")
	for _, line := range overlayLines {
		result.WriteString(strings.Repeat(" ", paddingLeft))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}
