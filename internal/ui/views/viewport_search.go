package views

import (
	"fmt"
	"regexp"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ViewportSearch is a reusable search mixin for viewport-based views.
// Embed it in any view that displays scrollable text content.
// Search highlighting is handled by the v2 viewport's built-in highlight support.
type ViewportSearch struct {
	pattern    string
	regex      *regexp.Regexp
	matchCount int
}

// ApplySearch compiles a case-insensitive regex pattern and finds all byte-range
// matches in the raw content. Returns the matches suitable for viewport.SetHighlights().
// Falls back to literal matching if the pattern is not valid regex.
func (s *ViewportSearch) ApplySearch(pattern string, rawContent string) [][]int {
	if pattern == "" {
		s.Clear()
		return nil
	}
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		re, _ = regexp.Compile(regexp.QuoteMeta(pattern))
	}
	s.pattern = pattern
	s.regex = re
	matches := re.FindAllStringIndex(rawContent, -1)
	s.matchCount = len(matches)
	return matches
}

// RecomputeMatches re-runs FindAllStringIndex with the existing compiled regex
// against new content and updates matchCount. Used when content refreshes while
// search is active.
func (s *ViewportSearch) RecomputeMatches(rawContent string) [][]int {
	if s.regex == nil {
		return nil
	}
	matches := s.regex.FindAllStringIndex(rawContent, -1)
	s.matchCount = len(matches)
	return matches
}

// Clear resets all search state.
func (s *ViewportSearch) Clear() {
	s.pattern = ""
	s.regex = nil
	s.matchCount = 0
}

// ActivePattern returns the current search pattern, or "" if none.
func (s *ViewportSearch) ActivePattern() string {
	return s.pattern
}

// HasSearch returns true if a search is active.
func (s *ViewportSearch) HasSearch() bool {
	return s.regex != nil
}

// MatchCount returns the number of occurrences matching the search pattern.
func (s *ViewportSearch) MatchCount() int {
	return s.matchCount
}

// StatusText returns a display string for the search state,
// e.g. "[/pattern/ 3 matches]". Returns "" if no search is active.
func (s *ViewportSearch) StatusText() string {
	if s.pattern == "" {
		return ""
	}
	return fmt.Sprintf("[/%s/ %d matches]", s.pattern, s.matchCount)
}

// ConfigureHighlightStyles sets the HighlightStyle and SelectedHighlightStyle
// on a viewport model using the theme's search highlight colors.
func ConfigureHighlightStyles(vp *viewport.Model) {
	vp.HighlightStyle = lipgloss.NewStyle().
		Background(theme.ColorSearchHighlightBg).
		Foreground(theme.ColorSearchHighlightFg)
	vp.SelectedHighlightStyle = lipgloss.NewStyle().
		Background(theme.ColorSearchSelectedBg).
		Foreground(theme.ColorSearchHighlightFg).
		Bold(true)
}
