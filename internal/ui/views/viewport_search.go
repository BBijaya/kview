package views

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ViewportSearch is a reusable search mixin for viewport-based views.
// Embed it in any view that displays scrollable text content.
type ViewportSearch struct {
	pattern string
	regex   *regexp.Regexp
	matches []int // line indices with matches
	cursor  int   // current match index (-1 = none)
}

// ApplySearch compiles a case-insensitive regex pattern, scans lines for matches,
// and sets the cursor to the first match. Falls back to literal matching if the
// pattern is not valid regex.
func (s *ViewportSearch) ApplySearch(pattern string, lines []string) {
	if pattern == "" {
		s.Clear()
		return
	}
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		re, _ = regexp.Compile(regexp.QuoteMeta(pattern))
	}
	s.pattern = pattern
	s.regex = re
	s.matches = nil
	s.cursor = -1
	for i, line := range lines {
		if re.MatchString(line) {
			s.matches = append(s.matches, i)
		}
	}
	if len(s.matches) > 0 {
		s.cursor = 0
	}
}

// Clear resets all search state.
func (s *ViewportSearch) Clear() {
	s.pattern = ""
	s.regex = nil
	s.matches = nil
	s.cursor = -1
}

// ActivePattern returns the current search pattern, or "" if none.
func (s *ViewportSearch) ActivePattern() string {
	return s.pattern
}

// HasSearch returns true if a search is active.
func (s *ViewportSearch) HasSearch() bool {
	return s.regex != nil
}

// MatchCount returns the number of lines matching the search pattern.
func (s *ViewportSearch) MatchCount() int {
	return len(s.matches)
}

// NextMatch advances the cursor to the next match and returns the line offset,
// or -1 if there are no matches.
func (s *ViewportSearch) NextMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	s.cursor = (s.cursor + 1) % len(s.matches)
	return s.matches[s.cursor]
}

// PrevMatch moves the cursor to the previous match and returns the line offset,
// or -1 if there are no matches.
func (s *ViewportSearch) PrevMatch() int {
	if len(s.matches) == 0 {
		return -1
	}
	s.cursor--
	if s.cursor < 0 {
		s.cursor = len(s.matches) - 1
	}
	return s.matches[s.cursor]
}

// CurrentMatchOffset returns the line offset of the current match,
// or -1 if there are no matches.
func (s *ViewportSearch) CurrentMatchOffset() int {
	if s.cursor < 0 || s.cursor >= len(s.matches) {
		return -1
	}
	return s.matches[s.cursor]
}

// HighlightContent applies search highlighting to raw content.
// Matches are rendered with the search highlight style.
func (s *ViewportSearch) HighlightContent(rawContent string) string {
	if s.regex == nil {
		return rawContent
	}

	highlightStyle := lipgloss.NewStyle().
		Background(theme.ColorSearchHighlightBg).
		Foreground(theme.ColorSearchHighlightFg)

	lines := strings.Split(rawContent, "\n")
	for i, line := range lines {
		matches := s.regex.FindAllStringIndex(line, -1)
		if len(matches) == 0 {
			continue
		}
		// Build highlighted line from right to left to preserve indices
		for j := len(matches) - 1; j >= 0; j-- {
			m := matches[j]
			matched := line[m[0]:m[1]]
			line = line[:m[0]] + highlightStyle.Render(matched) + line[m[1]:]
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// StatusText returns a display string for the search state,
// e.g. "[/pattern/ 3 matches]". Returns "" if no search is active.
func (s *ViewportSearch) StatusText() string {
	if s.pattern == "" {
		return ""
	}
	return fmt.Sprintf("[/%s/ %d matches]", s.pattern, len(s.matches))
}
