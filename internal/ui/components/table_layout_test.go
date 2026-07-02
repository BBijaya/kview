package components

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func layoutTable(columns []Column, rows []Row, width int) *Table {
	t := NewTable(columns)
	t.SetSize(width, 20)
	t.SetRows(rows)
	return t
}

func TestCalculateColumnWidthsFlexibleAbsorbsSlack(t *testing.T) {
	// NAME is flexible; AGE and READY are fixed. On a wide terminal the
	// leftover space must go to NAME, not be spread over the fixed columns.
	tbl := layoutTable(
		[]Column{
			{Title: "NAME", Width: 10, MinWidth: 10, Flexible: true},
			{Title: "READY", Width: 5},
			{Title: "AGE", Width: 5},
		},
		[]Row{
			{Values: []string{"web-abc", "1/1", "5d"}},
			{Values: []string{"db-xyz", "1/1", "3d"}},
		},
		120,
	)

	widths := tbl.calculateColumnWidths(0, 0)
	if len(widths) != 3 {
		t.Fatalf("widths length = %d, want 3", len(widths))
	}
	// Fixed columns stay at content width: max(title, content+1)
	if widths[1] != 5 { // "READY" title = 5 > "1/1"+1 = 4
		t.Errorf("READY width = %d, want 5 (content width, no bloat)", widths[1])
	}
	if widths[2] != 3 { // "AGE" = 3 > "5d"+1 = 3
		t.Errorf("AGE width = %d, want 3 (content width, no bloat)", widths[2])
	}
	// Flexible NAME absorbs everything else
	total := widths[0] + widths[1] + widths[2]
	available := 120 - 2*columnGap
	if total != available {
		t.Errorf("total widths = %d, want %d (flexible column fills width)", total, available)
	}
}

func TestCalculateColumnWidthsNoFlexibleFallsBackToEqual(t *testing.T) {
	tbl := layoutTable(
		[]Column{
			{Title: "A", Width: 5},
			{Title: "B", Width: 5},
		},
		[]Row{{Values: []string{"aaa", "bbb"}}},
		40,
	)

	widths := tbl.calculateColumnWidths(0, 0)
	// Equal distribution with trailing slot: both columns grow by the same
	// amount (±1 for remainder)
	diff := widths[0] - widths[1]
	if diff < -1 || diff > 1 {
		t.Errorf("no-flexible columns diverged: %v (want near-equal growth)", widths)
	}
}

func TestMeasureContentWidthsHighWaterMark(t *testing.T) {
	tbl := layoutTable(
		[]Column{
			{Title: "NAME", Width: 10, Flexible: true},
			{Title: "RESTARTS", Width: 8},
		},
		[]Row{{Values: []string{"a-very-long-pod-name-here", "10"}}},
		80,
	)

	wide := make([]int, len(tbl.measuredWidths))
	copy(wide, tbl.measuredWidths)

	// Refresh with shorter data: measured widths must not shrink
	tbl.SetRows([]Row{{Values: []string{"short", "9"}}})
	for i, w := range tbl.measuredWidths {
		if w < wide[i] {
			t.Errorf("measuredWidths[%d] shrank %d -> %d after refresh; high-water mark violated", i, wide[i], w)
		}
	}

	// Longer data must still grow the mark
	tbl.SetRows([]Row{{Values: []string{"an-even-longer-pod-name-than-before-xxxx", "9"}}})
	if tbl.measuredWidths[0] <= wide[0] {
		t.Errorf("measuredWidths[0] = %d, want > %d (growth still allowed)", tbl.measuredWidths[0], wide[0])
	}

	// SetColumns resets the mark (view switch)
	tbl.SetColumns([]Column{
		{Title: "NAME", Width: 10, Flexible: true},
		{Title: "RESTARTS", Width: 8},
	})
	tbl.SetRows([]Row{{Values: []string{"short", "9"}}})
	if got, want := tbl.measuredWidths[0], lipgloss.Width("short")+1; got != want {
		t.Errorf("after SetColumns reset, measuredWidths[0] = %d, want %d", got, want)
	}
}

func TestCalculateColumnWidthsOverflowStillShrinksFlexible(t *testing.T) {
	// Content wider than terminal: flexible column must shrink (respecting
	// MinWidth) and the fixed column keeps its width.
	tbl := layoutTable(
		[]Column{
			{Title: "NAME", Width: 10, MinWidth: 8, Flexible: true},
			{Title: "AGE", Width: 5},
		},
		[]Row{{Values: []string{"an-extremely-long-name-that-overflows-everything", "5d"}}},
		30,
	)

	widths := tbl.calculateColumnWidths(0, 0)
	if widths[0] >= 49 { // original content width
		t.Errorf("flexible NAME width = %d, want shrunk below content width", widths[0])
	}
	if widths[0] < 8 {
		t.Errorf("flexible NAME width = %d, violates MinWidth 8", widths[0])
	}
}

func TestColOffsetResetsWhenColumnsFitAgain(t *testing.T) {
	// Regression: scroll right on a narrow terminal, then widen the
	// terminal so everything fits. The stale offset used to hide the
	// leading columns (including NAME) with no scroll indicator.
	cols := []Column{
		{Title: "NAME", Width: 20},
		{Title: "READY", Width: 8},
		{Title: "STATUS", Width: 12},
		{Title: "AGE", Width: 6},
	}
	tbl := layoutTable(cols, []Row{
		{Values: []string{"a-fairly-long-pod-name", "1/1", "Running", "5d"}},
	}, 25)

	tbl.calculateColumnWidths(0, 0)
	if tbl.maxColOffset == 0 {
		t.Fatal("fixture not narrow enough: expected horizontal scroll to engage")
	}

	// User scrolls all the way right
	tbl.colOffset = tbl.maxColOffset

	// Terminal widens; everything fits now
	tbl.SetSize(120, 20)
	tbl.calculateColumnWidths(0, 0)

	if tbl.maxColOffset != 0 {
		t.Fatalf("maxColOffset = %d, want 0 on wide terminal", tbl.maxColOffset)
	}
	if tbl.colOffset != 0 {
		t.Errorf("colOffset = %d, want 0 — leading columns are hidden with no indicator", tbl.colOffset)
	}
}

func TestColumnStabilityAcrossRefresh(t *testing.T) {
	// The regression that motivated the change: a value changing length in
	// one fixed column must not move the widths of other fixed columns.
	cols := []Column{
		{Title: "NAME", Width: 10, MinWidth: 10, Flexible: true},
		{Title: "READY", Width: 5},
		{Title: "RESTARTS", Width: 8},
		{Title: "AGE", Width: 5},
	}
	tbl := layoutTable(cols, []Row{
		{Values: []string{"web-abc", "1/1", "9", "5d"}},
	}, 100)
	before := tbl.calculateColumnWidths(0, 0)

	// RESTARTS grows 9 -> 10 (one char longer)
	tbl.SetRows([]Row{
		{Values: []string{"web-abc", "1/1", "10", "5d"}},
	})
	after := tbl.calculateColumnWidths(0, 0)

	// READY and AGE (fixed, unrelated) must not move
	if before[1] != after[1] {
		t.Errorf("READY width moved %d -> %d on unrelated data change", before[1], after[1])
	}
	if before[3] != after[3] {
		t.Errorf("AGE width moved %d -> %d on unrelated data change", before[3], after[3])
	}
}
