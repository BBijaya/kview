package ui

import (
	"charm.land/bubbles/v2/key"

	"github.com/bijaya/kview/internal/ui/theme"
)

// KeyMap re-exports the theme KeyMap type
type KeyMap = theme.KeyMap

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return theme.DefaultKeyMap()
}

// ShortHelp returns keybindings for the short help view
func ShortHelp(k KeyMap) []key.Binding {
	return k.ShortHelp()
}

// FullHelp returns keybindings for the full help view
func FullHelp(k KeyMap) [][]key.Binding {
	return k.FullHelp()
}
