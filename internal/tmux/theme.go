// Package tmux provides theme support for Gas Town tmux sessions.
package tmux

import (
	"fmt"
	"hash/fnv"
)

// WindowStyle represents window background colors (tmux window-style).
type WindowStyle struct {
	BG string // Background color (hex or tmux color name)
	FG string // Foreground color (hex or tmux color name)
}

// Style returns the tmux window-style string.
func (w WindowStyle) Style() string {
	return fmt.Sprintf("bg=%s,fg=%s", w.BG, w.FG)
}

// Theme represents a tmux color scheme for status bar and optional window background.
type Theme struct {
	Name string // Human-readable name
	BG   string // Background color (hex or tmux color name)
	FG   string // Foreground color (hex or tmux color name)

	// Window is the optional window background style (tmux window-style).
	// nil = disabled (window uses terminal defaults).
	// If set, its BG/FG are applied as the window background.
	Window *WindowStyle `json:"window,omitempty"`
}

// DefaultPalette is the curated set of distinct, professional color themes.
// Each theme has good contrast and is visually distinct from others.
var DefaultPalette = []Theme{
	{Name: "sky", BG: "#c8dff0", FG: "#1a3a5a"},     // Light blue
	{Name: "sage", BG: "#c8e0cc", FG: "#1a3d22"},    // Light green
	{Name: "peach", BG: "#f0d0b8", FG: "#5a2a00"},   // Light peach
	{Name: "lavender", BG: "#ddd0e8", FG: "#3a1a55"}, // Light purple
	{Name: "silver", BG: "#d8dde8", FG: "#1a2040"},  // Light gray-blue
	{Name: "amber", BG: "#f5e8c0", FG: "#5a3a00"},   // Light amber
	{Name: "fog", BG: "#e0e8f0", FG: "#202840"},     // Light blue-gray
	{Name: "rose", BG: "#f0d0d5", FG: "#5a1520"},    // Light rose
	{Name: "mint", BG: "#c8e8e0", FG: "#0a3d35"},    // Light mint
	{Name: "sand", BG: "#e8dcc8", FG: "#3d2810"},    // Light sand
}

// MayorTheme returns the special theme for the Mayor session.
// Uses "default" to inherit the user's terminal colors — the Mayor
// session is the primary interactive session, so it should blend in.
func MayorTheme() Theme {
	return Theme{Name: "mayor", BG: "default", FG: "default"}
}

// DeaconTheme returns the special theme for the Deacon session.
// Light purple - ecclesiastical, distinct from Mayor's terminal default.
func DeaconTheme() Theme {
	return Theme{Name: "deacon", BG: "#e8d8f0", FG: "#3d1a55"}
}

// DogTheme returns the theme for Dog sessions.
// Light tan - earthy, loyal worker aesthetic.
func DogTheme() Theme {
	return Theme{Name: "dog", BG: "#ede5d5", FG: "#3d2810"}
}

// GetThemeByName finds a theme by name from the default palette.
// Returns nil if not found.
func GetThemeByName(name string) *Theme {
	for _, t := range DefaultPalette {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// AssignTheme picks a theme for a rig based on its name.
// Uses consistent hashing so the same rig always gets the same color.
func AssignTheme(rigName string) Theme {
	return AssignThemeFromPalette(rigName, DefaultPalette)
}

// AssignThemeFromPalette picks a theme using a custom palette.
func AssignThemeFromPalette(rigName string, palette []Theme) Theme {
	if len(palette) == 0 {
		return DefaultPalette[0]
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(rigName))
	idx := int(h.Sum32()) % len(palette)
	return palette[idx]
}

// Style returns the tmux status-style string for this theme.
func (t Theme) Style() string {
	return fmt.Sprintf("bg=%s,fg=%s", t.BG, t.FG)
}

// ListThemeNames returns the names of all themes in the default palette.
func ListThemeNames() []string {
	names := make([]string, len(DefaultPalette))
	for i, t := range DefaultPalette {
		names[i] = t.Name
	}
	return names
}
