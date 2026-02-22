package components

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// ScrollEvent records a single-row scroll that occurred during moveUp/moveDown.
// Direction is -1 (scroll up) or +1 (scroll down).
// NewLine is the fully rendered row string to insert at the boundary.
type ScrollEvent struct {
	Direction int
	NewLine   string
}

// Column represents a table column
type Column struct {
	Title     string
	Width     int
	MinWidth  int
	Flexible  bool              // If true, column can grow to fill space
	Align     lipgloss.Position // Left, Right, Center alignment
	IsNumeric bool              // Auto right-align numeric columns
}

// Row represents a table row
type Row struct {
	ID     string            // Unique identifier for the row
	Values []string          // Cell values
	Status string            // Status for color coding
	Labels map[string]string // Kubernetes labels for -l filter
}

// precomputedColStyle holds pre-built styles for a single column, avoiding
// per-cell Style copy overhead from .Width() and .Align() calls.
type precomputedColStyle struct {
	normal   lipgloss.Style
	selected lipgloss.Style
	alt      lipgloss.Style
	header   lipgloss.Style
}

// columnGap is the number of characters between columns (like tview's separator).
const columnGap = 1

// renderContext holds pre-computed styles and padding strings used by renderRowString.
// Avoids passing many parameters and allows gap strings to be pre-computed once.
type renderContext struct {
	colStyles          []precomputedColStyle
	widths             []int
	rowNumWidth        int
	startCol           int  // first visible column index (horizontal scroll)
	endCol             int  // one past last visible column index
	truncWidth         int  // width of last visible column when truncated (0 = full width)
	showRightIndicator bool // whether to show ▶ scroll indicator
	targetWidth        int  // expected total row width (t.width) for safety padding
	bgPad              string // trailing padding: normal background
	selPad             string // trailing padding: selected+focused
	selBlurPad         string // trailing padding: selected+blurred
	altPad             string // trailing padding: alternate row
	bgGap              string // column gap: normal background
	selGap             string // column gap: selected+focused
	selBlurGap         string // column gap: selected+blurred
	altGap             string // column gap: alternate row
}

// rowCacheKey identifies a cached rendered row.
type rowCacheKey struct {
	rowIdx     int
	isSelected bool
	isFocused  bool
	colOffset  int
}

// Table is a data table component
type Table struct {
	columns         []Column
	rows            []Row
	filteredRows    []Row
	cursor          int
	offset          int
	width           int
	height          int
	filter          string
	focused         bool
	showHeader      bool
	showIndicator   bool
	showRowNumbers  bool
	alternateRows   bool
	showStatusIcons bool
	selectedStyle   lipgloss.Style
	normalStyle     lipgloss.Style
	altStyle        lipgloss.Style
	headerStyle     lipgloss.Style
	indicatorStyle  lipgloss.Style
	statusColumnIdx int

	// Empty state configuration
	emptyIcon    string
	emptyTitle   string
	emptyMessage string
	emptyHint    string

	// Column width cache (avoids recalculating every frame)
	cachedWidths   []int
	cachedWidthKey [3]int // width, height, numCols

	// Content-aware column widths measured from actual data (like k9s ComputeMaxColumns)
	measuredWidths []int

	// Row render cache: caches complete rendered row strings.
	// Invalidated (via cacheGen bump) on SetRows, SetSize, Focus, Blur, SetFilter.
	rowCache     map[rowCacheKey]string
	cacheGen     uint64
	lastCacheGen uint64

	// Horizontal scroll (column-offset model, like k9s)
	colOffset    int // number of columns to skip from the left
	maxColOffset int // maximum column offset (computed in calculateColumnWidths)

	// Sorting
	sortCol int  // -1 = unsorted
	sortAsc bool // true = ascending

	// Scroll event tracking for scroll region optimization
	prevOffset     int
	scrollEvent    *ScrollEvent
	bulkScrolled   bool // true when offset changed by >1 (page up/down, home/end)
}

// NewTable creates a new table component
func NewTable(columns []Column) *Table {
	// Find STATUS column index
	statusIdx := -1
	for i, col := range columns {
		if col.Title == "STATUS" {
			statusIdx = i
			break
		}
	}

	return &Table{
		columns:         columns,
		rows:            []Row{},
		filteredRows:    []Row{},
		cursor:          0,
		offset:          0,
		width:           80,
		height:          20,
		focused:         true,
		sortCol:         -1,
		showHeader:      true,
		showIndicator:   true,
		showRowNumbers:  true,
		alternateRows:   false,
		showStatusIcons: true,
		selectedStyle:   theme.Styles.TableRowSelectedFocused,
		normalStyle:     theme.Styles.TableRow,
		altStyle:        theme.Styles.TableRowAlt,
		headerStyle:     theme.Styles.TableHeader,
		indicatorStyle:  theme.Styles.RowIndicatorFocused,
		statusColumnIdx: statusIdx,
		emptyIcon:       theme.IconEmptyBox,
		emptyTitle:      "No resources found",
		emptyMessage:    "",
		emptyHint:       "",
	}
}

// SetColumns replaces the table's columns (for dynamic column changes like conditional NAMESPACE)
func (t *Table) SetColumns(columns []Column) {
	t.columns = columns
	t.statusColumnIdx = -1
	for i, col := range columns {
		if col.Title == "STATUS" {
			t.statusColumnIdx = i
			break
		}
	}
	t.cachedWidths = nil
	t.measuredWidths = nil
	t.colOffset = 0
	t.cacheGen++
}

// SetShowIndicator sets whether to show the row selection indicator
func (t *Table) SetShowIndicator(show bool) {
	t.showIndicator = show
}

// SetAlternateRows sets whether to use alternating row colors
func (t *Table) SetAlternateRows(alternate bool) {
	t.alternateRows = alternate
}

// SetShowStatusIcons sets whether to show status icons
func (t *Table) SetShowStatusIcons(show bool) {
	t.showStatusIcons = show
}

// SetShowRowNumbers sets whether to show row numbers
func (t *Table) SetShowRowNumbers(show bool) {
	t.showRowNumbers = show
}

// SetEmptyState configures the empty state display
func (t *Table) SetEmptyState(icon, title, message, hint string) {
	t.emptyIcon = icon
	t.emptyTitle = title
	t.emptyMessage = message
	t.emptyHint = hint
}

// SetSize sets the table dimensions
func (t *Table) SetSize(width, height int) {
	if t.width != width || t.height != height {
		t.width = width
		t.height = height
		t.cachedWidths = nil // recalculates maxColOffset, clamps colOffset
		t.cacheGen++
	}
	t.ValidateCursor()
}

// ValidateCursor ensures cursor and offset are within valid bounds
func (t *Table) ValidateCursor() {
	if len(t.filteredRows) == 0 {
		t.cursor = 0
		t.offset = 0
		return
	}
	if t.cursor >= len(t.filteredRows) {
		t.cursor = len(t.filteredRows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	visibleRows := t.height - 1
	if visibleRows < 1 {
		visibleRows = 1
	}
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	if t.cursor >= t.offset+visibleRows {
		t.offset = t.cursor - visibleRows + 1
	}
}

// SetRows sets the table rows, preserving cursor position by ID
func (t *Table) SetRows(rows []Row) {
	// Save current selection before replacing rows
	var selectedID string
	savedCursor := t.cursor
	if t.cursor >= 0 && t.cursor < len(t.filteredRows) {
		selectedID = t.filteredRows[t.cursor].ID
	}

	t.rows = rows
	t.applyFilter()
	t.measureContentWidths()
	t.cachedWidths = nil // force width recalculation with new measured data
	t.colOffset = 0
	t.cacheGen++

	// Restore cursor to previously selected row by ID
	if selectedID != "" {
		for i, row := range t.filteredRows {
			if row.ID == selectedID {
				t.cursor = i
				t.ValidateCursor()
				return
			}
		}
	}

	// Row not found (deleted/filtered out): clamp to same index
	if savedCursor >= len(t.filteredRows) {
		t.cursor = max(0, len(t.filteredRows)-1)
	} else {
		t.cursor = savedCursor
	}
	t.ValidateCursor()
}

// SetFilter sets the filter string and resets cursor to top.
// Use this for user-initiated filter changes.
func (t *Table) SetFilter(filter string) {
	if t.filter == filter {
		return
	}
	t.filter = filter
	t.applyFilter()
	t.colOffset = 0
	t.cacheGen++
	t.cursor = 0
	t.offset = 0
}

// SetFilterPreserveCursor sets the filter string while preserving
// the cursor on the same row by ID. Used when restoring filters
// during view navigation (drill-down/back) so the user doesn't
// lose their place.
func (t *Table) SetFilterPreserveCursor(filter string) {
	var selectedID string
	savedCursor := t.cursor
	if t.cursor >= 0 && t.cursor < len(t.filteredRows) {
		selectedID = t.filteredRows[t.cursor].ID
	}

	t.filter = filter
	t.applyFilter()
	t.colOffset = 0
	t.cacheGen++

	if selectedID != "" {
		for i, row := range t.filteredRows {
			if row.ID == selectedID {
				t.cursor = i
				t.ValidateCursor()
				return
			}
		}
	}

	if savedCursor >= len(t.filteredRows) {
		t.cursor = max(0, len(t.filteredRows)-1)
	} else {
		t.cursor = savedCursor
	}
	t.ValidateCursor()
}

// GetFilter returns the current filter
func (t *Table) GetFilter() string {
	return t.filter
}

// InvalidateCache forces the row render cache to be rebuilt on the next render.
func (t *Table) InvalidateCache() {
	t.cacheGen++
}

// Focus focuses the table
func (t *Table) Focus() {
	t.focused = true
	t.selectedStyle = theme.Styles.TableRowSelectedFocused
	t.indicatorStyle = theme.Styles.RowIndicatorFocused
	t.cacheGen++
}

// Blur unfocuses the table
func (t *Table) Blur() {
	t.focused = false
	t.selectedStyle = theme.Styles.TableRowSelectedBlurred
	t.indicatorStyle = theme.Styles.RowIndicatorBlurred
	t.cacheGen++
}

// SelectedRow returns the currently selected row
func (t *Table) SelectedRow() *Row {
	if len(t.filteredRows) == 0 || t.cursor >= len(t.filteredRows) {
		return nil
	}
	return &t.filteredRows[t.cursor]
}

// SelectedValue returns the value at the given column index for the selected row
func (t *Table) SelectedValue(colIndex int) string {
	row := t.SelectedRow()
	if row == nil || colIndex < 0 || colIndex >= len(row.Values) {
		return ""
	}
	return row.Values[colIndex]
}

// SelectedIndex returns the currently selected index
func (t *Table) SelectedIndex() int {
	return t.cursor
}

// RowCount returns the number of visible rows
func (t *Table) RowCount() int {
	return len(t.filteredRows)
}

// Update handles key events
func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Up):
			t.moveUp()
		case key.Matches(msg, theme.DefaultKeyMap().Down):
			t.moveDown()
		case key.Matches(msg, theme.DefaultKeyMap().PageUp):
			t.pageUp()
		case key.Matches(msg, theme.DefaultKeyMap().PageDown):
			t.pageDown()
		case key.Matches(msg, theme.DefaultKeyMap().Home):
			t.GotoTop()
		case key.Matches(msg, theme.DefaultKeyMap().End):
			t.goToBottom()
		case key.Matches(msg, theme.DefaultKeyMap().ScrollLeft):
			if t.colOffset > 0 {
				t.colOffset--
				t.cacheGen++
			}
		case key.Matches(msg, theme.DefaultKeyMap().ScrollRight):
			if t.colOffset < t.maxColOffset {
				t.colOffset++
				t.cacheGen++
			}
		case key.Matches(msg, theme.DefaultKeyMap().SortToggle):
			if t.sortCol < 0 {
				// Unsorted → column 0 ascending
				t.sortCol = 0
				t.sortAsc = true
			} else if t.sortAsc {
				// Ascending → descending
				t.sortAsc = false
			} else {
				// Descending → unsorted
				t.sortCol = -1
			}
			t.applyFilter()
			t.cursor = 0
			t.offset = 0
			t.cacheGen++
		case key.Matches(msg, theme.DefaultKeyMap().SortColNext):
			if t.sortCol < 0 {
				// Unsorted → column 0 ascending
				t.sortCol = 0
				t.sortAsc = true
			} else if t.sortCol >= len(t.columns)-1 {
				// Past last column → unsorted
				t.sortCol = -1
			} else {
				t.sortCol++
			}
			t.applyFilter()
			t.cursor = 0
			t.offset = 0
			t.cacheGen++
		case key.Matches(msg, theme.DefaultKeyMap().SortColPrev):
			if t.sortCol < 0 {
				// Unsorted → last column ascending
				t.sortCol = len(t.columns) - 1
				t.sortAsc = true
			} else if t.sortCol <= 0 {
				// Past first column → unsorted
				t.sortCol = -1
			} else {
				t.sortCol--
			}
			t.applyFilter()
			t.cursor = 0
			t.offset = 0
			t.cacheGen++
		}
	}

	return t, nil
}

// Note: intToStr is defined in tabs.go to avoid duplication

func (t *Table) moveUp() {
	if t.cursor > 0 {
		oldOffset := t.offset
		t.cursor--
		t.ValidateCursor()
		if t.offset == oldOffset-1 {
			// Scrolled up by 1: the new top row needs to be inserted
			t.scrollEvent = &ScrollEvent{
				Direction: -1,
				NewLine:   t.RenderSingleRow(t.offset),
			}
		}
		t.prevOffset = t.offset
	}
}

func (t *Table) moveDown() {
	if t.cursor < len(t.filteredRows)-1 {
		oldOffset := t.offset
		t.cursor++
		t.ValidateCursor()
		if t.offset == oldOffset+1 {
			// Scrolled down by 1: the new bottom row needs to be inserted
			visibleRows := t.height - 1
			if visibleRows < 1 {
				visibleRows = 1
			}
			bottomIdx := t.offset + visibleRows - 1
			if bottomIdx >= len(t.filteredRows) {
				bottomIdx = len(t.filteredRows) - 1
			}
			t.scrollEvent = &ScrollEvent{
				Direction: +1,
				NewLine:   t.RenderSingleRow(bottomIdx),
			}
		}
		t.prevOffset = t.offset
	}
}

func (t *Table) pageUp() {
	oldOffset := t.offset
	visibleRows := t.height - 1
	t.cursor -= visibleRows
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ValidateCursor()
	if t.offset != oldOffset {
		t.bulkScrolled = true
	}
	t.prevOffset = t.offset
}

func (t *Table) pageDown() {
	oldOffset := t.offset
	visibleRows := t.height - 1
	t.cursor += visibleRows
	if t.cursor >= len(t.filteredRows) {
		t.cursor = len(t.filteredRows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ValidateCursor()
	if t.offset != oldOffset {
		t.bulkScrolled = true
	}
	t.prevOffset = t.offset
}

// GotoTop resets cursor and offset to the top
func (t *Table) GotoTop() {
	oldOffset := t.offset
	t.cursor = 0
	t.offset = 0
	if t.offset != oldOffset {
		t.bulkScrolled = true
	}
	t.prevOffset = t.offset
}

func (t *Table) goToBottom() {
	if len(t.filteredRows) > 0 {
		oldOffset := t.offset
		t.cursor = len(t.filteredRows) - 1
		t.ValidateCursor()
		if t.offset != oldOffset {
			t.bulkScrolled = true
		}
		t.prevOffset = t.offset
	}
}

// ConsumeBulkScroll returns and clears the bulk scroll flag (set on page up/down/home/end).
func (t *Table) ConsumeBulkScroll() bool {
	v := t.bulkScrolled
	t.bulkScrolled = false
	return v
}

// ConsumeScrollEvent returns and clears the pending scroll event, if any.
func (t *Table) ConsumeScrollEvent() *ScrollEvent {
	ev := t.scrollEvent
	t.scrollEvent = nil
	return ev
}

// ClearSort resets the sort state to unsorted.
func (t *Table) ClearSort() {
	t.sortCol = -1
	t.applyFilter()
	t.cacheGen++
}

// SortedColumn returns the current sort column and direction.
// col is -1 when unsorted.
func (t *Table) SortedColumn() (col int, asc bool) {
	return t.sortCol, t.sortAsc
}
