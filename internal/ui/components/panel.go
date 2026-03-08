package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// PanelOrientation defines how panels are split
type PanelOrientation int

const (
	Horizontal PanelOrientation = iota
	Vertical
)

// Panel is a bordered panel component
type Panel struct {
	title        string
	content      string
	width        int
	height       int
	focused      bool
	showBorder   bool
	scrollY      int
	contentLines []string
}

// NewPanel creates a new panel
func NewPanel(title string) *Panel {
	return &Panel{
		title:      title,
		width:      40,
		height:     10,
		showBorder: true,
	}
}

// SetTitle sets the panel title
func (p *Panel) SetTitle(title string) {
	p.title = title
}

// SetContent sets the panel content
func (p *Panel) SetContent(content string) {
	p.content = content
	p.contentLines = strings.Split(content, "\n")
}

// SetSize sets the panel dimensions
func (p *Panel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Focus focuses the panel
func (p *Panel) Focus() {
	p.focused = true
}

// Blur unfocuses the panel
func (p *Panel) Blur() {
	p.focused = false
}

// ShowBorder enables/disables the border
func (p *Panel) ShowBorder(show bool) {
	p.showBorder = show
}

// ScrollUp scrolls the content up
func (p *Panel) ScrollUp(lines int) {
	p.scrollY -= lines
	if p.scrollY < 0 {
		p.scrollY = 0
	}
}

// ScrollDown scrolls the content down
func (p *Panel) ScrollDown(lines int) {
	maxScroll := len(p.contentLines) - p.height + 2 // Account for border
	if maxScroll < 0 {
		maxScroll = 0
	}
	p.scrollY += lines
	if p.scrollY > maxScroll {
		p.scrollY = maxScroll
	}
}

// View renders the panel
func (p *Panel) View() string {
	style := theme.Styles.Panel
	if p.focused {
		style = style.BorderForeground(theme.ColorPrimary)
	}

	// Calculate content area
	contentWidth := p.width - 2 // Account for borders
	contentHeight := p.height - 2

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Prepare content with scrolling
	var visibleContent string
	if len(p.contentLines) > 0 {
		startLine := p.scrollY
		endLine := startLine + contentHeight
		if endLine > len(p.contentLines) {
			endLine = len(p.contentLines)
		}
		if startLine < len(p.contentLines) {
			visibleLines := p.contentLines[startLine:endLine]
			// Truncate lines to fit width (use display width for UTF-8 safety)
			for i, line := range visibleLines {
				if lipgloss.Width(line) > contentWidth {
					// Truncate rune-by-rune to avoid cutting multi-byte characters
					truncated := []rune(line)
					for lipgloss.Width(string(truncated)) > contentWidth && len(truncated) > 0 {
						truncated = truncated[:len(truncated)-1]
					}
					visibleLines[i] = string(truncated)
				}
			}
			visibleContent = strings.Join(visibleLines, "\n")
		}
	} else {
		visibleContent = p.content
	}

	// Build panel
	if p.showBorder {
		style = style.Width(p.width).Height(p.height)

		// Add title if present
		if p.title != "" {
			titleStyle := theme.Styles.PanelTitle
			title := titleStyle.Render(" " + p.title + " ")
			style = style.BorderTop(true)
			// Note: lipgloss doesn't support title in border directly,
			// so we'll include it in the content
			visibleContent = title + "\n" + visibleContent
		}

		return style.Render(visibleContent)
	}

	return lipgloss.NewStyle().Width(p.width).Height(p.height).Render(visibleContent)
}

// SplitPanel manages two panels in a split view
type SplitPanel struct {
	left        *Panel
	right       *Panel
	top         *Panel
	bottom      *Panel
	orientation PanelOrientation
	ratio       float64 // 0.0 to 1.0, proportion for first panel
	width       int
	height      int
	focusLeft   bool
}

// NewSplitPanel creates a new split panel
func NewSplitPanel(orientation PanelOrientation) *SplitPanel {
	return &SplitPanel{
		orientation: orientation,
		ratio:       0.5,
		focusLeft:   true,
	}
}

// SetPanels sets the two panels (left/right or top/bottom)
func (s *SplitPanel) SetPanels(first, second *Panel) {
	if s.orientation == Horizontal {
		s.left = first
		s.right = second
	} else {
		s.top = first
		s.bottom = second
	}
}

// SetSize sets the split panel dimensions
func (s *SplitPanel) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.updatePanelSizes()
}

// SetRatio sets the split ratio
func (s *SplitPanel) SetRatio(ratio float64) {
	if ratio < 0.1 {
		ratio = 0.1
	}
	if ratio > 0.9 {
		ratio = 0.9
	}
	s.ratio = ratio
	s.updatePanelSizes()
}

// FocusFirst focuses the first panel (left or top)
func (s *SplitPanel) FocusFirst() {
	s.focusLeft = true
	if s.orientation == Horizontal {
		if s.left != nil {
			s.left.Focus()
		}
		if s.right != nil {
			s.right.Blur()
		}
	} else {
		if s.top != nil {
			s.top.Focus()
		}
		if s.bottom != nil {
			s.bottom.Blur()
		}
	}
}

// FocusSecond focuses the second panel (right or bottom)
func (s *SplitPanel) FocusSecond() {
	s.focusLeft = false
	if s.orientation == Horizontal {
		if s.left != nil {
			s.left.Blur()
		}
		if s.right != nil {
			s.right.Focus()
		}
	} else {
		if s.top != nil {
			s.top.Blur()
		}
		if s.bottom != nil {
			s.bottom.Focus()
		}
	}
}

// ToggleFocus toggles focus between panels
func (s *SplitPanel) ToggleFocus() {
	if s.focusLeft {
		s.FocusSecond()
	} else {
		s.FocusFirst()
	}
}

func (s *SplitPanel) updatePanelSizes() {
	if s.orientation == Horizontal {
		leftWidth := int(float64(s.width) * s.ratio)
		rightWidth := s.width - leftWidth
		if s.left != nil {
			s.left.SetSize(leftWidth, s.height)
		}
		if s.right != nil {
			s.right.SetSize(rightWidth, s.height)
		}
	} else {
		topHeight := int(float64(s.height) * s.ratio)
		bottomHeight := s.height - topHeight
		if s.top != nil {
			s.top.SetSize(s.width, topHeight)
		}
		if s.bottom != nil {
			s.bottom.SetSize(s.width, bottomHeight)
		}
	}
}

// View renders the split panel
func (s *SplitPanel) View() string {
	if s.orientation == Horizontal {
		leftView := ""
		rightView := ""
		if s.left != nil {
			leftView = s.left.View()
		}
		if s.right != nil {
			rightView = s.right.View()
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
	}

	topView := ""
	bottomView := ""
	if s.top != nil {
		topView = s.top.View()
	}
	if s.bottom != nil {
		bottomView = s.bottom.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, topView, bottomView)
}
