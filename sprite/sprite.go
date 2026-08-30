package sprite

import "fmt"

// Sprite is an immutable pixel-art definition. All methods return values and
// never mutate the receiver, so a package-level sprite can be shared freely.
type Sprite struct {
	Dex     int
	Name    string
	Width   int
	Height  int
	Palette Palette
	// Pixels holds Width*Height palette indices in row-major order.
	// TransparentIndex means the cell is outside the artwork.
	Pixels []uint8
	// BodyIndex is the palette entry that best represents the sprite, used
	// for chrome such as the progress bar. Zero means "no preference".
	BodyIndex uint8
}

// BodyColor returns the sprite's representative colour, falling back to the
// first non-transparent palette entry when BodyIndex is unset or out of range.
func (s Sprite) BodyColor() (Color, bool) {
	if s.BodyIndex != TransparentIndex && int(s.BodyIndex) < len(s.Palette) {
		return s.Palette[s.BodyIndex], true
	}
	if len(s.Palette) > 1 {
		return s.Palette[1], true
	}
	return "", false
}

// At returns the palette index at (x, y), or TransparentIndex when the
// coordinates fall outside the sprite.
func (s Sprite) At(x, y int) uint8 {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return TransparentIndex
	}
	i := y*s.Width + x
	// Guard the slice too: At is exported and callers are not required to
	// have run Validate first.
	if i >= len(s.Pixels) {
		return TransparentIndex
	}
	return s.Pixels[i]
}

// ColorAt returns the colour at (x, y) and whether the cell is part of the
// artwork. Transparent cells report ok == false.
func (s Sprite) ColorAt(x, y int) (Color, bool) {
	idx := s.At(x, y)
	if idx == TransparentIndex || int(idx) >= len(s.Palette) {
		return "", false
	}
	return s.Palette[idx], true
}

// DotCount returns the number of non-transparent cells, i.e. how many dots
// must be lit for the sprite to be complete.
func (s Sprite) DotCount() int {
	n := 0
	for _, idx := range s.Pixels {
		if idx != TransparentIndex {
			n++
		}
	}
	return n
}

// Histogram counts how many cells use each palette index, skipping
// transparent cells. It is the checksum used to verify the transcription
// against the source bead chart.
func (s Sprite) Histogram() map[uint8]int {
	counts := make(map[uint8]int, len(s.Palette))
	for _, idx := range s.Pixels {
		if idx == TransparentIndex {
			continue
		}
		counts[idx]++
	}
	return counts
}

// Validate checks the sprite's internal consistency. Callers should run this
// at start-up so malformed data fails fast rather than rendering garbage.
func (s Sprite) Validate() error {
	if s.Width <= 0 || s.Height <= 0 {
		return fmt.Errorf("sprite %q has non-positive dimensions %dx%d", s.Name, s.Width, s.Height)
	}
	if want := s.Width * s.Height; len(s.Pixels) != want {
		return fmt.Errorf("sprite %q has %d pixels, want %d for a %dx%d grid", s.Name, len(s.Pixels), want, s.Width, s.Height)
	}
	if err := s.Palette.Validate(); err != nil {
		return fmt.Errorf("sprite %q: %w", s.Name, err)
	}
	for i, idx := range s.Pixels {
		if int(idx) >= len(s.Palette) {
			return fmt.Errorf("sprite %q pixel %d references palette index %d, but the palette has %d entries", s.Name, i, idx, len(s.Palette))
		}
	}
	return nil
}

// asciiDigits maps a palette index to its character in ASCII output. Hex keeps
// one character per cell for palettes of up to 16 colours.
const asciiDigits = "0123456789abcdef"

// ASCII renders the sprite as one line per row, using the palette index as a
// hex digit and '.' for transparent cells. Indices beyond the digit table
// render as '?'. Used for golden tests and debugging.
func (s Sprite) ASCII() string {
	buf := make([]byte, 0, s.Height*(s.Width+1))
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			idx := s.At(x, y)
			if idx == TransparentIndex {
				buf = append(buf, '.')
				continue
			}
			if int(idx) >= len(asciiDigits) {
				buf = append(buf, '?')
				continue
			}
			buf = append(buf, asciiDigits[idx])
		}
		buf = append(buf, '\n')
	}
	return string(buf)
}
