package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// buildColStyles pre-computes per-column styles with Inline(true) to avoid
// per-cell Style copy overhead from .Width() and .Align() calls.
func (t *Table) buildColStyles(widths []int) []precomputedColStyle {
	colStyles := make([]precomputedColStyle, len(t.columns))
	for i := range t.columns {
		align := t.getColumnAlign(i)
		colStyles[i] = precomputedColStyle{
			normal:   t.normalStyle.Inline(true).Width(widths[i]).Align(align),
			selected: t.selectedStyle.Inline(true).Width(widths[i]).Align(align),
			alt:      t.altStyle.Inline(true).Width(widths[i]).Align(align),
			header:   t.headerStyle.Inline(true).Width(widths[i]).Align(align),
		}
	}
	return colStyles
}

// renderRowString renders a single data row as a complete styled line.
// Uses pre-computed column styles, padding strings, and gap strings for efficiency.
func (t *Table) renderRowString(rowIdx int, isSelected bool, ctx *renderContext) string {
	if rowIdx < 0 || rowIdx >= len(t.filteredRows) {
		return ""
	}

	row := t.filteredRows[rowIdx]
	isAltRow := t.alternateRows && (rowIdx%2 == 1)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Completed rows: apply the status column's foreground color to ALL cells,
	// so the entire row is visually uniform (like k9s CompletedColor).
	isCompletedRow := !isSelected && row.Status == "Completed"
	var completedFg lipgloss.TerminalColor
	if isCompletedRow {
		completedFg = theme.ColorCompletedText
	}

	// Delta row coloring (whole-row foreground override, higher priority than completed)
	isDeltaRow := !isSelected && row.DeltaState != DeltaNone
	var deltaFg lipgloss.TerminalColor
	if isDeltaRow {
		switch row.DeltaState {
		case DeltaError:
			deltaFg = theme.ColorDeltaError
		case DeltaAdd:
			deltaFg = theme.ColorDeltaAdd
		case DeltaModify:
			deltaFg = theme.ColorDeltaModify
		}
	}
	scrollIndicatorLeft := t.maxColOffset > 0 && t.colOffset > 0
	scrollIndicatorRight := ctx.showRightIndicator

	var rowContent strings.Builder

	// Left scroll indicator (replaces 1 char at start of row number area).
	// Background must match the row state to avoid color leaks.
	if scrollIndicatorLeft {
		var indicatorBg lipgloss.TerminalColor = theme.ColorBackground
		if isSelected {
			if t.focused {
				indicatorBg = theme.ColorSelectionBg
			} else {
				indicatorBg = theme.ColorSurface
			}
		}
		rowContent.WriteString(lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(indicatorBg).Render("◀"))
	}

	// Render row number
	if t.showRowNumbers {
		numWidth := ctx.rowNumWidth - 1
		if scrollIndicatorLeft {
			numWidth-- // 1 char used by left indicator
			if numWidth < 1 {
				numWidth = 1
			}
		}
		rowNumStyle := theme.Styles.RowNumber.Width(numWidth)
		if isSelected {
			if t.focused {
				rowNumStyle = rowNumStyle.Background(theme.ColorSelectionBg).Foreground(theme.ColorSelectionFg)
			} else {
				rowNumStyle = rowNumStyle.Background(theme.ColorSurface).Foreground(theme.ColorMuted)
			}
		}
		rowContent.WriteString(rowNumStyle.Render(intToStr(rowIdx + 1)))
		if isSelected {
			selSpaceStyle := lipgloss.NewStyle().Background(theme.ColorSelectionBg)
			if !t.focused {
				selSpaceStyle = lipgloss.NewStyle().Background(theme.ColorSurface)
			}
			rowContent.WriteString(selSpaceStyle.Render(" "))
		} else {
			rowContent.WriteString(bgStyle.Render(" "))
		}
	} else if scrollIndicatorLeft {
		// No row numbers, but we already wrote the indicator; nothing more needed
	}

	// Render row indicator
	if t.showIndicator {
		if isSelected {
			selSpaceStyle := lipgloss.NewStyle().Background(theme.ColorSelectionBg)
			if !t.focused {
				selSpaceStyle = lipgloss.NewStyle().Background(theme.ColorSurface)
			}
			rowContent.WriteString(t.indicatorStyle.Render("►"))
			rowContent.WriteString(selSpaceStyle.Render(" "))
		} else {
			rowContent.WriteString(bgStyle.Render("  "))
		}
	}

	// Render cells using pre-computed column styles, with gaps between columns.
	// Render visible columns [startCol, endCol); last column may be truncated.
	for j := ctx.startCol; j < ctx.endCol; j++ {
		// Insert column gap before each column except the first visible
		if j > ctx.startCol {
			if isSelected {
				if t.focused {
					rowContent.WriteString(ctx.selGap)
				} else {
					rowContent.WriteString(ctx.selBlurGap)
				}
			} else if isAltRow {
				rowContent.WriteString(ctx.altGap)
			} else {
				rowContent.WriteString(ctx.bgGap)
			}
		}

		// Effective column width (may be truncated for last visible column)
		colW := ctx.widths[j]
		isTrunc := ctx.truncWidth > 0 && j == ctx.endCol-1
		if isTrunc {
			colW = ctx.truncWidth
		}

		value := ""
		if j < len(row.Values) {
			value = row.Values[j]
		}

		// Apply status styling to the status column
		if j == t.statusColumnIdx && row.Status != "" {
			if !isSelected {
				// Delta rows: override status cell with delta foreground
				if isDeltaRow {
					if t.showStatusIcons {
						value = theme.StatusIconPrefix(row.Status) + value
					}
					deltaStatusStyle := lipgloss.NewStyle().
						Foreground(deltaFg).
						Background(theme.ColorBackground)
					rowContent.WriteString(deltaStatusStyle.Inline(true).Width(colW).Align(t.getColumnAlign(j)).Render(theme.TruncateString(value, colW)))
					continue
				}
				// Completed rows: grey text with circle icon (not green checkmark)
				if isCompletedRow {
					completedStatusStyle := lipgloss.NewStyle().
						Foreground(theme.ColorCompletedText).
						Background(theme.ColorBackground)
					if t.showStatusIcons {
						value = theme.IconPending + " " + value
					}
					rowContent.WriteString(completedStatusStyle.Inline(true).Width(colW).Align(t.getColumnAlign(j)).Render(theme.TruncateString(value, colW)))
					continue
				}
				statusStyle := theme.StatusCellStyle(row.Status)
				if t.showStatusIcons {
					value = theme.StatusIconPrefix(row.Status) + value
				}
				rowContent.WriteString(statusStyle.Inline(true).Width(colW).Align(t.getColumnAlign(j)).Render(theme.TruncateString(value, colW)))
				continue
			}
			// For selected rows, use selected style with status color
			style := ctx.colStyles[j].selected.Inherit(theme.StatusStyle(row.Status))
			if isTrunc {
				style = style.Width(colW)
			}
			rowContent.WriteString(style.Render(theme.TruncateString(value, colW)))
			continue
		}

		if isTrunc {
			// Truncated column: override pre-computed width
			if isSelected {
				rowContent.WriteString(ctx.colStyles[j].selected.Width(colW).Render(theme.TruncateString(value, colW)))
			} else if isDeltaRow {
				style := ctx.colStyles[j].normal.Foreground(deltaFg)
				if isAltRow {
					style = ctx.colStyles[j].alt.Foreground(deltaFg)
				}
				rowContent.WriteString(style.Width(colW).Render(theme.TruncateString(value, colW)))
			} else if isCompletedRow {
				style := ctx.colStyles[j].normal.Foreground(completedFg)
				if isAltRow {
					style = ctx.colStyles[j].alt.Foreground(completedFg)
				}
				rowContent.WriteString(style.Width(colW).Render(theme.TruncateString(value, colW)))
			} else if isAltRow {
				rowContent.WriteString(ctx.colStyles[j].alt.Width(colW).Render(theme.TruncateString(value, colW)))
			} else {
				rowContent.WriteString(ctx.colStyles[j].normal.Width(colW).Render(theme.TruncateString(value, colW)))
			}
		} else {
			if isSelected {
				rowContent.WriteString(ctx.colStyles[j].selected.Render(theme.TruncateString(value, colW)))
			} else if isDeltaRow {
				style := ctx.colStyles[j].normal.Foreground(deltaFg)
				if isAltRow {
					style = ctx.colStyles[j].alt.Foreground(deltaFg)
				}
				rowContent.WriteString(style.Render(theme.TruncateString(value, colW)))
			} else if isCompletedRow {
				style := ctx.colStyles[j].normal.Foreground(completedFg)
				if isAltRow {
					style = ctx.colStyles[j].alt.Foreground(completedFg)
				}
				rowContent.WriteString(style.Render(theme.TruncateString(value, colW)))
			} else if isAltRow {
				rowContent.WriteString(ctx.colStyles[j].alt.Render(theme.TruncateString(value, colW)))
			} else {
				rowContent.WriteString(ctx.colStyles[j].normal.Render(theme.TruncateString(value, colW)))
			}
		}
	}

	// Trailing padding
	if isSelected {
		if t.focused {
			rowContent.WriteString(ctx.selPad)
		} else {
			rowContent.WriteString(ctx.selBlurPad)
		}
	} else if isAltRow {
		rowContent.WriteString(ctx.altPad)
	} else {
		rowContent.WriteString(ctx.bgPad)
	}

	// Right scroll indicator — background matches the row state
	if scrollIndicatorRight {
		var indicatorBg lipgloss.TerminalColor = theme.ColorBackground
		if isSelected {
			if t.focused {
				indicatorBg = theme.ColorSelectionBg
			} else {
				indicatorBg = theme.ColorSurface
			}
		} else if isAltRow {
			indicatorBg = theme.ColorSurfaceAlt
		}
		rowContent.WriteString(lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(indicatorBg).Render("▶"))
	}

	// Width safety net: if the rendered row is narrower than expected (e.g. due to
	// ambiguous-width characters like ◀/▶ being counted as 1 cell by go-runewidth
	// but rendered as 2 cells by some terminals), pad to targetWidth. We only pad,
	// never truncate, to avoid corrupting ANSI escape sequences.
	result := rowContent.String()
	if ctx.targetWidth > 0 {
		if actualWidth := lipgloss.Width(result); actualWidth < ctx.targetWidth {
			extra := ctx.targetWidth - actualWidth
			padStr := strings.Repeat(" ", extra)
			var bg lipgloss.TerminalColor = theme.ColorBackground
			if isSelected {
				if t.focused {
					bg = theme.ColorSelectionBg
				} else {
					bg = theme.ColorSurface
				}
			} else if isAltRow {
				bg = theme.ColorSurfaceAlt
			}
			result += lipgloss.NewStyle().Background(bg).Render(padStr)
		}
	}
	return result
}

// View renders the table
func (t *Table) View() string {
	// Offset is pre-computed in movement functions via ValidateCursor().
	// Only validate here as a safety net; this should be a no-op.
	t.ValidateCursor()

	var b strings.Builder

	// Calculate visible rows
	visibleRows := t.height - 1 // Account for header
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Calculate row number width (based on total rows)
	rowNumWidth := 0
	if t.showRowNumbers {
		rowNumWidth = len(intToStr(len(t.filteredRows))) + 2 // digits + padding
		if rowNumWidth < 4 {
			rowNumWidth = 4
		}
	}

	// Calculate column widths (accounting for indicator and row numbers)
	indicatorWidth := 0
	if t.showIndicator {
		indicatorWidth = 2 // "► " or "  "
	}

	// Use cached widths if dimensions haven't changed
	widthKey := [3]int{t.width, t.height, len(t.columns)}
	var widths []int
	if t.cachedWidths != nil && t.cachedWidthKey == widthKey {
		widths = t.cachedWidths
	} else {
		widths = t.calculateColumnWidths(rowNumWidth, indicatorWidth)
		t.cachedWidths = widths
		t.cachedWidthKey = widthKey
	}

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Determine visible column range based on horizontal scroll offset.
	// Columns that fully fit are rendered at full width. The first column
	// that overflows is truncated to fill remaining space exactly, ensuring
	// every row is exactly t.width characters (no terminal background leaks).
	startCol := t.colOffset
	if startCol >= len(t.columns) {
		startCol = 0
	}

	scrollIndicatorLeft := t.maxColOffset > 0 && t.colOffset > 0
	scrollIndicatorRight := t.maxColOffset > 0 && t.colOffset < t.maxColOffset

	// Available width for columns (reserve 1 char for ▶ when scrollable)
	colAvailable := t.width - rowNumWidth - indicatorWidth
	if scrollIndicatorRight {
		colAvailable--
	}
	if colAvailable < 1 {
		colAvailable = 1
	}

	// Walk columns: full-fit until one overflows, then truncate it
	remaining := colAvailable
	endCol := startCol
	truncWidth := 0
	for c := startCol; c < len(t.columns); c++ {
		gap := 0
		if c > startCol {
			gap = columnGap
		}
		if remaining < gap+1 {
			break // not even room for gap + 1 char
		}
		remaining -= gap
		if widths[c] <= remaining {
			remaining -= widths[c]
			endCol = c + 1
		} else {
			// Partially fits — truncate to remaining width
			truncWidth = remaining
			remaining = 0
			endCol = c + 1
			break
		}
	}
	// Ensure at least one column is visible
	if endCol <= startCol && startCol < len(t.columns) {
		endCol = startCol + 1
		if widths[startCol] <= colAvailable {
			truncWidth = 0
		} else {
			truncWidth = colAvailable
			if truncWidth < 1 {
				truncWidth = 1
			}
		}
		remaining = 0
	}

	padWidth := remaining

	// Pre-compute gap strings for each row type
	gapStr := strings.Repeat(" ", columnGap)
	bgGap := bgStyle.Render(gapStr)
	selGap := lipgloss.NewStyle().Background(theme.ColorSelectionBg).Render(gapStr)
	selBlurGap := lipgloss.NewStyle().Background(theme.ColorSurface).Render(gapStr)
	altGap := lipgloss.NewStyle().Background(theme.ColorSurfaceAlt).Render(gapStr)

	// Pre-compute trailing padding strings for each row type (avoids per-row style creation)
	// Note: ▶ space is already reserved in colAvailable, so padWidth is purely for padding.
	var bgPad, selPad, selBlurPad, altPad string
	if padWidth > 0 {
		padStr := strings.Repeat(" ", padWidth)
		bgPad = bgStyle.Render(padStr)
		selPad = lipgloss.NewStyle().Background(theme.ColorSelectionBg).Render(padStr)
		selBlurPad = lipgloss.NewStyle().Background(theme.ColorSurface).Render(padStr)
		altPad = lipgloss.NewStyle().Background(theme.ColorSurfaceAlt).Render(padStr)
	}

	// Pre-compute per-column styles (eliminates ~420 Style copies per frame)
	colStyles := t.buildColStyles(widths)

	// Build render context
	ctx := &renderContext{
		colStyles:          colStyles,
		widths:             widths,
		rowNumWidth:        rowNumWidth,
		startCol:           startCol,
		endCol:             endCol,
		truncWidth:         truncWidth,
		showRightIndicator: scrollIndicatorRight,
		targetWidth:        t.width,
		bgPad:              bgPad,
		selPad:             selPad,
		selBlurPad:         selBlurPad,
		altPad:             altPad,
		bgGap:              bgGap,
		selGap:             selGap,
		selBlurGap:         selBlurGap,
		altGap:             altGap,
	}

	// Pre-compute scroll indicator strings
	mutedStyle := lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorBackground)
	leftIndicator := ""
	rightIndicator := ""
	if scrollIndicatorLeft {
		leftIndicator = mutedStyle.Render("◀")
	}
	if scrollIndicatorRight {
		rightIndicator = mutedStyle.Render("▶")
	}

	// Render header with full-width background using pre-computed column styles
	if t.showHeader {
		var headerContent strings.Builder
		// Left scroll indicator (replaces 1 char of row number / indicator space)
		if scrollIndicatorLeft {
			headerContent.WriteString(leftIndicator)
			// Add remaining row number header space
			if t.showRowNumbers {
				remaining := rowNumWidth - 1
				if remaining > 0 {
					headerContent.WriteString(bgStyle.Render(strings.Repeat(" ", remaining)))
				}
			}
			// Add indicator space to header
			if t.showIndicator {
				headerContent.WriteString(bgStyle.Render("  "))
			}
		} else {
			// Add row number header space
			if t.showRowNumbers {
				headerContent.WriteString(bgStyle.Render(strings.Repeat(" ", rowNumWidth)))
			}
			// Add indicator space to header
			if t.showIndicator {
				headerContent.WriteString(bgStyle.Render("  "))
			}
		}
		// Render visible columns [startCol, endCol), truncating the last if needed
		for i := startCol; i < endCol; i++ {
			if i > startCol {
				headerContent.WriteString(bgGap) // column gap in header
			}
			hw := widths[i]
			if truncWidth > 0 && i == endCol-1 {
				hw = truncWidth
			}
			title := t.columns[i].Title
			if i == t.sortCol {
				if t.sortAsc {
					title += " ▲"
				} else {
					title += " ▼"
				}
			}
			headerContent.WriteString(t.headerStyle.Inline(true).Width(hw).Align(t.getColumnAlign(i)).Render(theme.TruncateString(title, hw)))
		}
		// Trailing padding
		if padWidth > 0 {
			headerContent.WriteString(bgStyle.Render(strings.Repeat(" ", padWidth)))
		}
		// Right scroll indicator
		if scrollIndicatorRight {
			headerContent.WriteString(rightIndicator)
		}
		// Width safety net for header: pad to t.width if narrower than expected
		// (e.g. due to ambiguous-width scroll indicator characters ◀/▶)
		headerStr := headerContent.String()
		if hw := lipgloss.Width(headerStr); hw < t.width {
			headerStr += bgStyle.Render(strings.Repeat(" ", t.width-hw))
		}
		b.WriteString(headerStr)
		b.WriteString("\n")
	}

	// Render rows
	if len(t.filteredRows) == 0 {
		b.WriteString(t.renderEmptyState(visibleRows))
	} else {
		// Invalidate row cache if generation changed
		if t.lastCacheGen != t.cacheGen {
			t.rowCache = make(map[rowCacheKey]string)
			t.lastCacheGen = t.cacheGen
		}
		if t.rowCache == nil {
			t.rowCache = make(map[rowCacheKey]string)
		}

		endIdx := min(t.offset+visibleRows, len(t.filteredRows))

		for i := t.offset; i < endIdx; i++ {
			isSelected := i == t.cursor

			// Check row cache (includes colOffset for horizontal scroll)
			key := rowCacheKey{rowIdx: i, isSelected: isSelected, isFocused: t.focused, colOffset: t.colOffset}
			if cached, ok := t.rowCache[key]; ok {
				b.WriteString(cached)
			} else {
				rendered := t.renderRowString(i, isSelected, ctx)
				t.rowCache[key] = rendered
				b.WriteString(rendered)
			}

			b.WriteString("\n")
		}

		// Fill remaining visible rows with background-colored empty lines
		renderedRows := endIdx - t.offset
		bgFillStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
		emptyLine := bgFillStyle.Render(strings.Repeat(" ", t.width))
		for i := renderedRows; i < visibleRows; i++ {
			b.WriteString(emptyLine)
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// getColumnAlign returns the alignment for a column
func (t *Table) getColumnAlign(colIdx int) lipgloss.Position {
	if colIdx < 0 || colIdx >= len(t.columns) {
		return lipgloss.Left
	}
	col := t.columns[colIdx]

	// If explicit alignment is set, use it
	if col.Align != 0 {
		return col.Align
	}

	// Auto-detect for numeric columns
	if col.IsNumeric {
		return lipgloss.Right
	}

	// Default to left
	return lipgloss.Left
}

// renderEmptyState renders a centered empty state message
func (t *Table) renderEmptyState(visibleRows int) string {
	var b strings.Builder
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground).Width(t.width)

	// Calculate vertical centering
	contentLines := 1 // icon
	if t.emptyTitle != "" {
		contentLines++
	}
	if t.emptyMessage != "" {
		contentLines++
	}
	if t.emptyHint != "" {
		contentLines++
	}

	topPadding := (visibleRows - contentLines) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Add top padding
	for i := 0; i < topPadding; i++ {
		b.WriteString(bgStyle.Render(""))
		b.WriteString("\n")
	}

	// Render icon (centered)
	if t.emptyIcon != "" {
		iconStyle := theme.Styles.EmptyStateIcon.Width(t.width)
		b.WriteString(iconStyle.Render(t.emptyIcon))
		b.WriteString("\n")
	}

	// Render title (centered)
	if t.emptyTitle != "" {
		titleStyle := theme.Styles.EmptyStateTitle.Width(t.width)
		b.WriteString(titleStyle.Render(t.emptyTitle))
		if t.emptyMessage != "" || t.emptyHint != "" {
			b.WriteString("\n")
		}
	}

	// Render message (centered)
	if t.emptyMessage != "" {
		msgStyle := theme.Styles.EmptyStateMessage.Width(t.width)
		b.WriteString(msgStyle.Render(t.emptyMessage))
		if t.emptyHint != "" {
			b.WriteString("\n")
		}
	}

	// Render hint (centered)
	if t.emptyHint != "" {
		hintStyle := theme.Styles.EmptyStateHint.Width(t.width)
		b.WriteString(hintStyle.Render(t.emptyHint))
	}

	// Fill remaining visible rows with background
	renderedLines := topPadding + contentLines
	bgFillStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	emptyLine := bgFillStyle.Render(strings.Repeat(" ", t.width))
	for i := renderedLines; i < visibleRows; i++ {
		b.WriteString("\n")
		b.WriteString(emptyLine)
	}

	return b.String()
}

// RenderSingleRow renders one data row at the given index as a complete styled line.
// The output matches the format produced by View() for that row.
func (t *Table) RenderSingleRow(rowIdx int) string {
	if rowIdx < 0 || rowIdx >= len(t.filteredRows) {
		return ""
	}

	isSelected := rowIdx == t.cursor

	// Calculate dimensions (use cached widths if available)
	rowNumWidth := 0
	if t.showRowNumbers {
		rowNumWidth = len(intToStr(len(t.filteredRows))) + 2
		if rowNumWidth < 4 {
			rowNumWidth = 4
		}
	}
	indicatorWidth := 0
	if t.showIndicator {
		indicatorWidth = 2
	}

	widthKey := [3]int{t.width, t.height, len(t.columns)}
	var widths []int
	if t.cachedWidths != nil && t.cachedWidthKey == widthKey {
		widths = t.cachedWidths
	} else {
		widths = t.calculateColumnWidths(rowNumWidth, indicatorWidth)
		t.cachedWidths = widths
		t.cachedWidthKey = widthKey
	}

	// Determine visible column range (same logic as View)
	startCol := t.colOffset
	if startCol >= len(t.columns) {
		startCol = 0
	}

	scrollIndicatorRight := t.maxColOffset > 0 && t.colOffset < t.maxColOffset

	// Available width for columns (reserve 1 char for ▶ when scrollable)
	colAvailable := t.width - rowNumWidth - indicatorWidth
	if scrollIndicatorRight {
		colAvailable--
	}
	if colAvailable < 1 {
		colAvailable = 1
	}

	// Walk columns: full-fit until one overflows, then truncate it
	remaining := colAvailable
	endCol := startCol
	truncWidth := 0
	for c := startCol; c < len(t.columns); c++ {
		gap := 0
		if c > startCol {
			gap = columnGap
		}
		if remaining < gap+1 {
			break
		}
		remaining -= gap
		if widths[c] <= remaining {
			remaining -= widths[c]
			endCol = c + 1
		} else {
			truncWidth = remaining
			remaining = 0
			endCol = c + 1
			break
		}
	}
	if endCol <= startCol && startCol < len(t.columns) {
		endCol = startCol + 1
		if widths[startCol] <= colAvailable {
			truncWidth = 0
		} else {
			truncWidth = colAvailable
			if truncWidth < 1 {
				truncWidth = 1
			}
		}
		remaining = 0
	}

	padWidth := remaining

	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)

	// Pre-compute gap strings
	gapStr := strings.Repeat(" ", columnGap)
	bgGap := bgStyle.Render(gapStr)
	selGap := lipgloss.NewStyle().Background(theme.ColorSelectionBg).Render(gapStr)
	selBlurGap := lipgloss.NewStyle().Background(theme.ColorSurface).Render(gapStr)
	altGap := lipgloss.NewStyle().Background(theme.ColorSurfaceAlt).Render(gapStr)

	// Pre-compute padding strings
	var bgPad, selPad, selBlurPad, altPad string
	if padWidth > 0 {
		padStr := strings.Repeat(" ", padWidth)
		bgPad = bgStyle.Render(padStr)
		selPad = lipgloss.NewStyle().Background(theme.ColorSelectionBg).Render(padStr)
		selBlurPad = lipgloss.NewStyle().Background(theme.ColorSurface).Render(padStr)
		altPad = lipgloss.NewStyle().Background(theme.ColorSurfaceAlt).Render(padStr)
	}

	colStyles := t.buildColStyles(widths)
	ctx := &renderContext{
		colStyles:          colStyles,
		widths:             widths,
		rowNumWidth:        rowNumWidth,
		startCol:           startCol,
		endCol:             endCol,
		truncWidth:         truncWidth,
		showRightIndicator: scrollIndicatorRight,
		targetWidth:        t.width,
		bgPad:              bgPad,
		selPad:             selPad,
		selBlurPad:         selBlurPad,
		altPad:             altPad,
		bgGap:              bgGap,
		selGap:             selGap,
		selBlurGap:         selBlurGap,
		altGap:             altGap,
	}
	return t.renderRowString(rowIdx, isSelected, ctx)
}

// RenderVisibleRows renders all currently visible data rows as individual strings.
// Used for SyncScrollArea initialization.
func (t *Table) RenderVisibleRows() []string {
	visibleRows := t.height - 1
	if visibleRows < 1 {
		visibleRows = 1
	}
	endIdx := min(t.offset+visibleRows, len(t.filteredRows))

	var rows []string
	for i := t.offset; i < endIdx; i++ {
		rows = append(rows, t.RenderSingleRow(i))
	}

	// Fill remaining visible rows with background-colored empty lines
	if len(rows) < visibleRows {
		bgFillStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
		emptyLine := bgFillStyle.Render(strings.Repeat(" ", t.width))
		for len(rows) < visibleRows {
			rows = append(rows, emptyLine)
		}
	}

	return rows
}
