package components

import (
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// measureContentWidths scans header titles and all row data to compute
// content-aware column widths, like k9s's ComputeMaxColumns.
func (t *Table) measureContentWidths() {
	t.measuredWidths = make([]int, len(t.columns))
	// Phase 1: header title widths
	for i, col := range t.columns {
		t.measuredWidths[i] = lipgloss.Width(col.Title)
	}
	// Phase 2: scan all rows, track max content width + 1 padding (like k9s colPadding=1)
	for _, row := range t.rows {
		for j, val := range row.Values {
			if j < len(t.measuredWidths) {
				w := lipgloss.Width(val) + 1
				if w > t.measuredWidths[j] {
					t.measuredWidths[j] = w
				}
			}
		}
	}
	// Account for status icons (2 chars: icon + space)
	if t.showStatusIcons && t.statusColumnIdx >= 0 && t.statusColumnIdx < len(t.measuredWidths) {
		t.measuredWidths[t.statusColumnIdx] += 2
	}
}

// applyFilter filters rows based on the current filter string, then sorts.
// Supports multiple filter modes via prefix:
//   - "!" prefix: inverse modifier (composes with all modes: !-f, !-l, !term)
//   - "-f " prefix: fuzzy match (characters appear in order, not contiguously)
//   - "-l " prefix: label selector filter (key=value, key!=value, key, !key)
//   - no prefix: substring filter (default)
func (t *Table) applyFilter() {
	if t.filter == "" {
		t.filteredRows = t.rows
	} else {
		// Extract ! inverse prefix before mode dispatch
		filter := t.filter
		invert := false
		if strings.HasPrefix(filter, "!") {
			invert = true
			filter = filter[1:]
		}

		if filter == "" {
			// "!" alone — show all rows
			t.filteredRows = t.rows
		} else if strings.HasPrefix(filter, "-f ") || filter == "-f" {
			// Fuzzy filter mode
			term := strings.TrimPrefix(filter, "-f")
			term = strings.TrimSpace(term)
			if term == "" {
				t.filteredRows = t.rows
			} else {
				term = strings.ToLower(term)
				t.filteredRows = make([]Row, 0, len(t.rows))
				for _, row := range t.rows {
					matched := false
					for _, val := range row.Values {
						if fuzzyMatch(strings.ToLower(val), term) {
							matched = true
							break
						}
					}
					if invert != matched {
						t.filteredRows = append(t.filteredRows, row)
					}
				}
			}
		} else if strings.HasPrefix(filter, "-l ") || filter == "-l" {
			// Label selector filter mode
			selector := strings.TrimPrefix(filter, "-l")
			selector = strings.TrimSpace(selector)
			if selector == "" {
				t.filteredRows = t.rows
			} else {
				t.filteredRows = make([]Row, 0, len(t.rows))
				for _, row := range t.rows {
					matched := matchLabelSelector(row.Labels, selector)
					if invert != matched {
						t.filteredRows = append(t.filteredRows, row)
					}
				}
			}
		} else {
			// Substring filter
			filter = strings.ToLower(filter)
			t.filteredRows = make([]Row, 0, len(t.rows))
			for _, row := range t.rows {
				matched := false
				for _, val := range row.Values {
					if strings.Contains(strings.ToLower(val), filter) {
						matched = true
						break
					}
				}
				if invert != matched {
					t.filteredRows = append(t.filteredRows, row)
				}
			}
		}
	}

	// Delta error filter: keep only unhealthy rows
	if t.deltaFilterActive {
		filtered := make([]Row, 0, len(t.filteredRows))
		for _, row := range t.filteredRows {
			if row.DeltaState == DeltaError {
				filtered = append(filtered, row)
			}
		}
		t.filteredRows = filtered
	}

	t.sortFilteredRows()
}

// fuzzyMatch returns true if all characters of term appear in value in order.
func fuzzyMatch(value, term string) bool {
	vi := 0
	for ti := 0; ti < len(term); ti++ {
		found := false
		for vi < len(value) {
			if value[vi] == term[ti] {
				vi++
				found = true
				break
			}
			vi++
		}
		if !found {
			return false
		}
	}
	return true
}

// matchLabelSelector checks if labels match a comma-separated label selector.
// Supported selector expressions:
//   - key=value  (key must equal value)
//   - key!=value (key must not equal value, or key absent)
//   - key        (key must exist)
//   - !key       (key must not exist)
func matchLabelSelector(labels map[string]string, selector string) bool {
	parts := strings.Split(selector, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if idx := strings.Index(part, "!="); idx >= 0 {
			// key!=value: key must not equal value (or be absent)
			key := strings.TrimSpace(part[:idx])
			val := strings.TrimSpace(part[idx+2:])
			if v, ok := labels[key]; ok && v == val {
				return false
			}
		} else if idx := strings.Index(part, "="); idx >= 0 {
			// key=value: key must equal value
			key := strings.TrimSpace(part[:idx])
			val := strings.TrimSpace(part[idx+1:])
			if v, ok := labels[key]; !ok || v != val {
				return false
			}
		} else if strings.HasPrefix(part, "!") {
			// !key: key must not exist
			key := strings.TrimSpace(part[1:])
			if _, ok := labels[key]; ok {
				return false
			}
		} else {
			// key: key must exist
			if _, ok := labels[part]; !ok {
				return false
			}
		}
	}
	return true
}

// sortFilteredRows sorts filteredRows in place based on current sort state.
// Always copies the slice first to avoid mutating the backing rows slice
// (when filter is empty, filteredRows == rows).
func (t *Table) sortFilteredRows() {
	if t.sortCol < 0 || len(t.filteredRows) <= 1 {
		return
	}

	col := t.sortCol
	asc := t.sortAsc

	// Copy to avoid mutating the original rows slice
	sorted := make([]Row, len(t.filteredRows))
	copy(sorted, t.filteredRows)

	sort.SliceStable(sorted, func(i, j int) bool {
		a := ""
		b := ""
		if col < len(sorted[i].Values) {
			a = sorted[i].Values[col]
		}
		if col < len(sorted[j].Values) {
			b = sorted[j].Values[col]
		}

		// Try numeric comparison first
		af, aErr := strconv.ParseFloat(a, 64)
		bf, bErr := strconv.ParseFloat(b, 64)
		if aErr == nil && bErr == nil {
			if asc {
				return af < bf
			}
			return af > bf
		}

		// Fall back to case-insensitive string comparison
		al := strings.ToLower(a)
		bl := strings.ToLower(b)
		if asc {
			return al < bl
		}
		return al > bl
	})

	t.filteredRows = sorted
}

// calculateColumnWidths computes column widths using content-aware sizing.
func (t *Table) calculateColumnWidths(rowNumWidth, indicatorWidth int) []int {
	// Content-aware column sizing: start with measured content widths,
	// then distribute remaining space equally across all columns so the
	// table fills the terminal width with even spacing.
	// When content overflows, flexible columns are shrunk proportionally.
	// If shrinking is insufficient (MinWidth constraints), horizontal
	// scrolling is enabled as a fallback.

	n := len(t.columns)
	if n == 0 {
		t.maxColOffset = 0
		return nil
	}

	totalGapWidth := 0
	if n > 1 {
		totalGapWidth = (n - 1) * columnGap
	}

	// Phase 1: content-aware base widths from measureContentWidths()
	widths := make([]int, n)
	for i, col := range t.columns {
		if i < len(t.measuredWidths) && t.measuredWidths[i] > 0 {
			widths[i] = t.measuredWidths[i]
		} else {
			widths[i] = col.Width // fallback when no data yet
		}
		if widths[i] < 1 {
			widths[i] = 1
		}
	}

	available := t.width - rowNumWidth - indicatorWidth - totalGapWidth
	if available < n {
		available = n
	}

	totalUsed := 0
	for _, w := range widths {
		totalUsed += w
	}

	if totalUsed < available {
		// Phase 2: distribute remaining space equally across all columns
		// plus one trailing slot, so the gap after the last column matches.
		remaining := available - totalUsed
		slots := n + 1 // n columns + 1 trailing padding
		perSlot := remaining / slots
		extra := remaining % slots
		for i := range widths {
			widths[i] += perSlot
			if i < extra {
				widths[i]++
			}
		}
		// The leftover (perSlot + any remaining extra) becomes trailing
		// padding automatically via padWidth in View().
		t.maxColOffset = 0
	} else if totalUsed > available {
		// Phase 3: overflow — shrink flexible columns proportionally to fit.
		flexibleCount := 0
		flexibleTotal := 0
		fixedTotal := 0
		for i, col := range t.columns {
			if col.Flexible {
				flexibleCount++
				flexibleTotal += widths[i]
			} else {
				fixedTotal += widths[i]
			}
		}

		if flexibleCount > 0 && flexibleTotal > 0 {
			flexAvailable := available - fixedTotal
			if flexAvailable < flexibleCount {
				flexAvailable = flexibleCount
			}
			distributed := 0
			remaining := flexibleCount
			for i, col := range t.columns {
				if col.Flexible {
					remaining--
					if remaining == 0 {
						widths[i] = flexAvailable - distributed
					} else {
						share := widths[i] * flexAvailable / flexibleTotal
						if col.MinWidth > 0 && share < col.MinWidth {
							share = col.MinWidth
						}
						widths[i] = share
						distributed += share
					}
					if widths[i] < 1 {
						widths[i] = 1
					}
				}
			}
		}

		// Phase 4: check if shrinking was sufficient. If total still
		// overflows (e.g. MinWidth constraints or no flexible columns),
		// enable horizontal scrolling as a fallback.
		postShrinkTotal := 0
		for _, w := range widths {
			postShrinkTotal += w
		}
		viewportWidth := t.width - rowNumWidth - indicatorWidth
		totalWithGaps := postShrinkTotal + totalGapWidth
		if totalWithGaps > viewportWidth {
			// Still overflows after shrinking — enable horizontal scroll.
			// Compute maxColOffset: smallest offset where columns [offset:]
			// plus their gaps fit within viewport.
			t.maxColOffset = 0
			for i := 1; i < n; i++ {
				colSum := 0
				for j := i; j < n; j++ {
					if j > i {
						colSum += columnGap
					}
					colSum += widths[j]
				}
				if colSum <= viewportWidth {
					t.maxColOffset = i
					break
				}
				t.maxColOffset = i
			}
			if t.colOffset > t.maxColOffset {
				t.colOffset = t.maxColOffset
			}
		} else {
			// Shrinking was sufficient — no scroll needed
			t.maxColOffset = 0
			t.colOffset = 0
		}
	} else {
		// Exact fit
		t.maxColOffset = 0
	}

	return widths
}
