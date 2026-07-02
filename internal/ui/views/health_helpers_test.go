package views

import (
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bijaya/kview/internal/ui/theme"
)

func TestHealthCellExactVisualWidth(t *testing.T) {
	cases := []struct {
		name    string
		content string
		width   int
	}{
		{"plain ascii", "Running", 12},
		{"needs padding", "ok", 10},
		{"needs truncation", "a-very-long-status-string", 10},
		{"empty", "", 6},
		{"icon prefix healthy", theme.StatusIconPrefix("Running") + " " + "Running", 15},
		{"icon prefix error", theme.StatusIconPrefix("CrashLoopBackOff") + " " + "CrashLoopBackOff", 15},
		{"wide glyphs", "状態確認", 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := healthCell(tc.content, tc.width)
			if w := lipgloss.Width(got); w != tc.width {
				t.Errorf("healthCell(%q, %d) visual width = %d, want exact", tc.content, tc.width, w)
			}
		})
	}
}

func TestHealthHeaderAndRowCellWidthsAgree(t *testing.T) {
	// The regression: headers rendered the label at cols.X while rows
	// rendered icon + label at cols.X + icon prefix, shifting every column
	// to the right of the icon column by ~3 cells.
	const colWidth = 12
	header := healthCell("STATUS", colWidth+healthIconPad)
	row := healthCell(theme.StatusIconPrefix("CrashLoopBackOff")+" "+"CrashLoopBackOff", colWidth+healthIconPad)
	if lipgloss.Width(header) != lipgloss.Width(row) {
		t.Errorf("header cell width %d != row cell width %d", lipgloss.Width(header), lipgloss.Width(row))
	}
}
