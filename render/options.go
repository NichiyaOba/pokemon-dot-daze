// Package render draws a progress snapshot as an SVG document.
package render

import (
	"fmt"

	"github.com/NichiyaOba/pokemon-dot-daze/sprite"
)

// Theme selects how the unlit background and caption adapt to the viewer.
type Theme string

const (
	// ThemeAuto follows the viewer's prefers-color-scheme setting.
	ThemeAuto Theme = "auto"
	// ThemeLight always renders the light palette.
	ThemeLight Theme = "light"
	// ThemeDark always renders the dark palette.
	ThemeDark Theme = "dark"
)

// Defaults for Options. Exported so the CLI can document them.
const (
	DefaultCellSize = 16
	DefaultGap      = 0
	DefaultPadding  = 12
)

// Default colours for the chrome around the artwork.
const (
	DefaultUnlitLight   sprite.Color = "#E6E6E6"
	DefaultUnlitDark    sprite.Color = "#2A2A2A"
	DefaultCaptionLight sprite.Color = "#57606A"
	DefaultCaptionDark  sprite.Color = "#8B949E"
)

// Options controls SVG output. The zero value is not usable; start from
// DefaultOptions and adjust.
type Options struct {
	// CellSize is the edge length of one dot in pixels.
	CellSize int
	// Gap is the space left between adjacent dots, giving a bead-chart look.
	Gap int
	// Padding surrounds the artwork.
	Padding int
	// Theme selects the colour scheme behaviour.
	Theme Theme
	// ShowCaption draws the sprite name and progress beneath the artwork.
	ShowCaption bool

	UnlitLight   sprite.Color
	UnlitDark    sprite.Color
	CaptionLight sprite.Color
	CaptionDark  sprite.Color
}

// DefaultOptions returns the options used when the caller has no preference.
func DefaultOptions() Options {
	return Options{
		CellSize:     DefaultCellSize,
		Gap:          DefaultGap,
		Padding:      DefaultPadding,
		Theme:        ThemeAuto,
		ShowCaption:  true,
		UnlitLight:   DefaultUnlitLight,
		UnlitDark:    DefaultUnlitDark,
		CaptionLight: DefaultCaptionLight,
		CaptionDark:  DefaultCaptionDark,
	}
}

// Validate checks the options are renderable.
func (o Options) Validate() error {
	if o.CellSize <= 0 {
		return fmt.Errorf("cell size must be positive, got %d", o.CellSize)
	}
	if o.Gap < 0 {
		return fmt.Errorf("gap must not be negative, got %d", o.Gap)
	}
	if o.Padding < 0 {
		return fmt.Errorf("padding must not be negative, got %d", o.Padding)
	}
	switch o.Theme {
	case ThemeAuto, ThemeLight, ThemeDark:
	default:
		return fmt.Errorf("unknown theme %q, want one of %q, %q, %q", o.Theme, ThemeAuto, ThemeLight, ThemeDark)
	}
	colors := []struct {
		name string
		c    sprite.Color
	}{
		{"unlit light colour", o.UnlitLight},
		{"unlit dark colour", o.UnlitDark},
		{"caption light colour", o.CaptionLight},
		{"caption dark colour", o.CaptionDark},
	}
	for _, e := range colors {
		if err := (sprite.Palette{"", e.c}).Validate(); err != nil {
			return fmt.Errorf("%s: %w", e.name, err)
		}
	}
	return nil
}
