package sprite

import "fmt"

// Color is an RGB value in CSS hex notation, e.g. "#74C47A".
type Color string

// TransparentIndex marks a cell that is not part of the artwork.
// It is distinct from a white pixel, which is a real colour in the palette.
const TransparentIndex uint8 = 0

// Palette maps a pixel index to a colour. Index 0 is reserved for
// TransparentIndex and is never rendered, so entry 0 is a placeholder.
type Palette []Color

// Validate reports whether the palette is usable for rendering.
func (p Palette) Validate() error {
	if len(p) < 2 {
		return fmt.Errorf("palette must define at least one colour beyond the transparent slot, got %d entries", len(p))
	}
	for i, c := range p {
		if i == int(TransparentIndex) {
			continue
		}
		if err := validateHexColor(c); err != nil {
			return fmt.Errorf("palette index %d: %w", i, err)
		}
	}
	return nil
}

func validateHexColor(c Color) error {
	if len(c) != 7 || c[0] != '#' {
		return fmt.Errorf("colour %q must be in #RRGGBB form", c)
	}
	for i := 1; i < len(c); i++ {
		if !isHexDigit(c[i]) {
			return fmt.Errorf("colour %q contains a non-hex character %q", c, c[i])
		}
	}
	return nil
}

func isHexDigit(b byte) bool {
	switch {
	case b >= '0' && b <= '9', b >= 'a' && b <= 'f', b >= 'A' && b <= 'F':
		return true
	default:
		return false
	}
}
