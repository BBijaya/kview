package commands

import "testing"

func TestPaletteSetSizeTinyTerminalDoesNotPanic(t *testing.T) {
	p := NewPalette(NewRegistry())
	for _, size := range [][2]int{{5, 3}, {0, 0}, {12, 8}, {39, 14}} {
		p.SetSize(size[0], size[1])
		if p.width < 20 {
			t.Errorf("SetSize(%d,%d): width = %d, want >= 20", size[0], size[1], p.width)
		}
		if p.maxVisible < 1 {
			t.Errorf("SetSize(%d,%d): maxVisible = %d, want >= 1", size[0], size[1], p.maxVisible)
		}
		p.Show()
		_ = p.View() // must not panic (strings.Repeat with negative count)
		p.Hide()
	}
}
