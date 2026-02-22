package commands

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ClosePaletteMsg is sent when the palette is closed
type ClosePaletteMsg struct{}

// Palette key bindings
type paletteKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
}

func defaultPaletteKeyMap() paletteKeyMap {
	return paletteKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
		),
	}
}

// paletteStyles returns computed palette styles from current theme colors.
// Called lazily so it picks up theme changes from Apply()/ComputeStyles().
func paletteStylesComputed() struct {
	Container    lipgloss.Style
	Input        lipgloss.Style
	Item         lipgloss.Style
	SelectedItem lipgloss.Style
	NoResults    lipgloss.Style
	Separator    lipgloss.Style
} {
	return struct {
		Container    lipgloss.Style
		Input        lipgloss.Style
		Item         lipgloss.Style
		SelectedItem lipgloss.Style
		NoResults    lipgloss.Style
		Separator    lipgloss.Style
	}{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorPrimary).
			BorderBackground(theme.ColorBackground).
			Background(theme.ColorBackground).
			Padding(0, 1),
		Input: lipgloss.NewStyle().
			Foreground(theme.ColorText).
			Background(theme.ColorSurface).
			Padding(0, 1),
		Item: lipgloss.NewStyle().
			Foreground(theme.ColorText).
			Background(theme.ColorBackground).
			Padding(0, 1),
		SelectedItem: lipgloss.NewStyle().
			Background(theme.ColorPrimary).
			Foreground(theme.ColorText).
			Padding(0, 1),
		NoResults: lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Background(theme.ColorBackground),
		Separator: lipgloss.NewStyle().
			Foreground(theme.ColorBorder).
			Background(theme.ColorBackground),
	}
}

// Palette is the command palette component
type Palette struct {
	registry   *Registry
	input      textinput.Model
	matches    []Command
	cursor     int
	width      int
	height     int
	maxVisible int
	visible    bool
	keys       paletteKeyMap
}

// NewPalette creates a new command palette
func NewPalette(registry *Registry) *Palette {
	inputBg := theme.ColorSurface
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.CharLimit = 100
	ti.Width = 50
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(inputBg)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.ColorText).Background(inputBg)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.ColorText).Background(inputBg)
	ti.Cursor.Style = lipgloss.NewStyle().Background(inputBg)

	p := &Palette{
		registry:   registry,
		input:      ti,
		maxVisible: 10,
		width:      60,
		keys:       defaultPaletteKeyMap(),
	}
	p.updateMatches()

	return p
}

// Show shows the palette
func (p *Palette) Show() {
	p.visible = true
	p.input.SetValue("")
	p.input.Focus()
	p.cursor = 0
	p.updateMatches()
}

// Hide hides the palette
func (p *Palette) Hide() {
	p.visible = false
	p.input.Blur()
}

// IsVisible returns whether the palette is visible
func (p *Palette) IsVisible() bool {
	return p.visible
}

// SetSize sets the palette dimensions
func (p *Palette) SetSize(width, height int) {
	p.width = min(width-10, 70)
	p.height = height
	p.maxVisible = min(10, height-6)
	p.input.Width = p.width - 4
}

// Update handles input events
func (p *Palette) Update(msg tea.Msg) (*Palette, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.keys.Escape):
			p.Hide()
			return p, func() tea.Msg { return ClosePaletteMsg{} }

		case key.Matches(msg, p.keys.Up):
			if p.cursor > 0 {
				p.cursor--
			}
			return p, nil

		case key.Matches(msg, p.keys.Down):
			if p.cursor < len(p.matches)-1 {
				p.cursor++
			}
			return p, nil

		case key.Matches(msg, p.keys.Enter):
			if len(p.matches) > 0 && p.cursor < len(p.matches) {
				selected := p.matches[p.cursor]
				p.Hide()
				if selected.Action != nil {
					result := selected.Action()
					return p, func() tea.Msg { return result }
				}
			}
			return p, nil
		}
	}

	// Update text input
	prevValue := p.input.Value()
	p.input, cmd = p.input.Update(msg)

	// If input changed, update matches
	if p.input.Value() != prevValue {
		p.updateMatches()
		p.cursor = 0
	}

	return p, cmd
}

// View renders the palette
func (p *Palette) View() string {
	if !p.visible {
		return ""
	}

	ps := paletteStylesComputed()
	var b strings.Builder

	// Input box
	inputStyle := ps.Input.Width(p.width - 2)
	b.WriteString(inputStyle.Render("> " + p.input.View()))
	b.WriteString("\n")

	// Separator
	sep := ps.Separator.Width(p.width - 2).Render(strings.Repeat("─", p.width-2))
	b.WriteString(sep)
	b.WriteString("\n")

	// Matches
	if len(p.matches) == 0 {
		noResults := ps.NoResults.Width(p.width - 2).Render("No matching commands")
		b.WriteString(noResults)
	} else {
		// Calculate visible range
		startIdx := 0
		if p.cursor >= p.maxVisible {
			startIdx = p.cursor - p.maxVisible + 1
		}
		endIdx := min(startIdx+p.maxVisible, len(p.matches))

		for i := startIdx; i < endIdx; i++ {
			cmd := p.matches[i]

			style := ps.Item
			if i == p.cursor {
				style = ps.SelectedItem
			}

			// Format: Name (shortcut) - Description
			line := cmd.Name
			if cmd.Shortcut != "" {
				line += " [" + cmd.Shortcut + "]"
			}

			// Truncate if needed
			maxLen := p.width - 4
			if len(line) > maxLen {
				line = line[:maxLen-3] + "..."
			}

			b.WriteString(style.Width(p.width - 2).Render(line))
			if i < endIdx-1 {
				b.WriteString("\n")
			}
		}

		// Show scroll indicator if there are more items
		if len(p.matches) > p.maxVisible {
			b.WriteString("\n")
			indicator := ps.NoResults.Width(p.width - 2).
				Align(lipgloss.Center).
				Render("↑↓ " + intToStr(len(p.matches)) + " commands")
			b.WriteString(indicator)
		}
	}

	// Wrap in container
	container := ps.Container.Width(p.width).Render(b.String())
	return container
}

// ViewCentered renders the palette centered on the screen
func (p *Palette) ViewCentered(screenWidth, screenHeight int) string {
	if !p.visible {
		return ""
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	content := p.View()
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	// Calculate centering
	padLeft := (screenWidth - contentWidth) / 2
	padTop := (screenHeight - contentHeight) / 3 // Slightly above center

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	emptyLine := bgStyle.Render(strings.Repeat(" ", screenWidth))
	leftPad := bgStyle.Render(strings.Repeat(" ", padLeft))
	rightPadWidth := screenWidth - padLeft - contentWidth
	if rightPadWidth < 0 {
		rightPadWidth = 0
	}

	// Build centered content
	var lines []string
	for i := 0; i < padTop; i++ {
		lines = append(lines, emptyLine)
	}

	for _, line := range strings.Split(content, "\n") {
		rightPad := bgStyle.Render(strings.Repeat(" ", rightPadWidth))
		lines = append(lines, leftPad+line+rightPad)
	}

	// Fill remaining lines below the palette
	usedLines := padTop + contentHeight
	for i := usedLines; i < screenHeight; i++ {
		lines = append(lines, emptyLine)
	}

	return strings.Join(lines, "\n")
}

func (p *Palette) updateMatches() {
	query := p.input.Value()
	if query == "" {
		p.matches = p.registry.All()
	} else {
		p.matches = p.registry.FuzzySearch(query)
	}
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
