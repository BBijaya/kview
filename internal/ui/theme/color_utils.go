package theme

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// parseHex parses a hex color string (#RRGGBB) into r, g, b components.
func parseHex(hex string) (r, g, b uint8) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return
}

// toHex converts r, g, b components to a hex color string (#RRGGBB).
func toHex(r, g, b uint8) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// blendColor blends two hex colors by factor (0.0 = c1, 1.0 = c2).
func blendColor(c1, c2 string, factor float64) color.Color {
	r1, g1, b1 := parseHex(c1)
	r2, g2, b2 := parseHex(c2)
	r := uint8(float64(r1)*(1-factor) + float64(r2)*factor)
	g := uint8(float64(g1)*(1-factor) + float64(g2)*factor)
	b := uint8(float64(b1)*(1-factor) + float64(b2)*factor)
	return lipgloss.Color(toHex(r, g, b))
}

// lightenColor lightens a hex color by factor (0.0 = unchanged, 1.0 = white).
func lightenColor(hex string, factor float64) color.Color {
	return blendColor(hex, "#FFFFFF", factor)
}

// darkenColor darkens a hex color by factor (0.0 = unchanged, 1.0 = black).
func darkenColor(hex string, factor float64) color.Color {
	return blendColor(hex, "#000000", factor)
}

// luminance returns the relative luminance of a hex color per WCAG 2.0.
func luminance(hex string) float64 {
	r, g, b := parseHex(hex)
	linearize := func(v uint8) float64 {
		s := float64(v) / 255.0
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linearize(r) + 0.7152*linearize(g) + 0.0722*linearize(b)
}

// ContrastRatio returns the WCAG 2.0 contrast ratio between two hex colors.
// Result range [1, 21]. Higher is better.
func ContrastRatio(hex1, hex2 string) float64 {
	l1 := luminance(hex1)
	l2 := luminance(hex2)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// ValidateContrast checks critical color pairs against WCAG AA threshold (3.0:1).
// Returns warning strings for failing pairs.
func ValidateContrast(td ThemeDefinition) []string {
	const threshold = 3.0
	var warnings []string
	pairs := [][3]string{
		{td.Text, td.Background, "text/bg"},
		{td.Muted, td.Background, "muted/bg"},
		{td.Highlight, td.Background, "highlight/bg"},
		{td.Border, td.Background, "border/bg"},
	}
	if td.SelectionBg != "" && td.SelectionFg != "" {
		pairs = append(pairs, [3]string{td.SelectionFg, td.SelectionBg, "selection fg/bg"})
	}
	for _, p := range pairs {
		ratio := ContrastRatio(p[0], p[1])
		if ratio < threshold {
			warnings = append(warnings, fmt.Sprintf("%s %.1f:1", p[2], ratio))
		}
	}
	return warnings
}

// contrastForeground picks white or black foreground text based on the
// luminance of the background color. When selectionBg is explicitly set,
// uses that; otherwise derives from accent via darkenColor(accent, 0.6).
func contrastForeground(selectionBg, accent string) color.Color {
	bg := selectionBg
	if bg == "" {
		// Match the derivation in Apply(): darkenColor(accent, 0.6)
		r1, g1, b1 := parseHex(accent)
		r := uint8(float64(r1) * 0.4)
		g := uint8(float64(g1) * 0.4)
		b := uint8(float64(b1) * 0.4)
		bg = toHex(r, g, b)
	}
	if luminance(bg) > 0.179 {
		return lipgloss.Color("#000000")
	}
	return lipgloss.Color("#FFFFFF")
}
