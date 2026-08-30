package sprite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBulbasaurValidates(t *testing.T) {
	if err := Bulbasaur.Validate(); err != nil {
		t.Fatalf("Bulbasaur.Validate() = %v, want nil", err)
	}
}

func TestBulbasaurDimensions(t *testing.T) {
	if Bulbasaur.Width != 20 || Bulbasaur.Height != 19 {
		t.Errorf("dimensions = %dx%d, want 20x19", Bulbasaur.Width, Bulbasaur.Height)
	}
	if got, want := len(Bulbasaur.Pixels), 20*19; got != want {
		t.Errorf("len(Pixels) = %d, want %d", got, want)
	}
}

// TestBulbasaurMatchesChartLegend is the transcription checksum. The source
// bead chart prints a per-colour tally in its legend; if the grid was copied
// correctly every count must match.
//
// Colour 1 is the one known deviation: the chart legend says 43, the grid as
// transcribed has 44. Every other colour matches exactly, so a single cell
// somewhere is read as colour 1 when the chart shows it empty. The effect is
// one extra dot (2520 contributions to complete instead of 2510).
func TestBulbasaurMatchesChartLegend(t *testing.T) {
	want := map[uint8]int{
		1: 44, // chart legend says 43 - see doc comment
		2: 65,
		3: 35,
		4: 22,
		5: 21,
		6: 2,
		7: 5,
		8: 58,
	}

	got := Bulbasaur.Histogram()

	if len(got) != len(want) {
		t.Errorf("histogram has %d colours, want %d", len(got), len(want))
	}
	for idx, wantCount := range want {
		if got[idx] != wantCount {
			t.Errorf("colour %d appears %d times, want %d", idx, got[idx], wantCount)
		}
	}
}

func TestBulbasaurDotCount(t *testing.T) {
	if got, want := Bulbasaur.DotCount(), 252; got != want {
		t.Errorf("DotCount() = %d, want %d", got, want)
	}
}

// TestBulbasaurASCIIGolden guards against cells moving around, which the
// histogram alone cannot detect.
func TestBulbasaurASCIIGolden(t *testing.T) {
	path := filepath.Join("testdata", "bulbasaur.txt")

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, []byte(Bulbasaur.ASCII()), 0o644); err != nil {
			t.Fatalf("writing golden file: %v", err)
		}
		t.Log("golden file updated")
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file (re-run with UPDATE_GOLDEN=1 to create it): %v", err)
	}
	if got := Bulbasaur.ASCII(); got != string(want) {
		t.Errorf("ASCII mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAtOutOfBoundsIsTransparent(t *testing.T) {
	cases := []struct{ x, y int }{{-1, 0}, {0, -1}, {20, 0}, {0, 19}, {100, 100}}
	for _, c := range cases {
		if got := Bulbasaur.At(c.x, c.y); got != TransparentIndex {
			t.Errorf("At(%d, %d) = %d, want TransparentIndex", c.x, c.y, got)
		}
	}
}

func TestColorAtReportsTransparency(t *testing.T) {
	// (0,0) is outside the artwork; (12,0) is the black outline of an ear.
	if _, ok := Bulbasaur.ColorAt(0, 0); ok {
		t.Error("ColorAt(0, 0) reported a colour, want transparent")
	}
	c, ok := Bulbasaur.ColorAt(12, 0)
	if !ok {
		t.Fatal("ColorAt(12, 0) reported transparent, want a colour")
	}
	if c != "#000000" {
		t.Errorf("ColorAt(12, 0) = %q, want #000000", c)
	}
}

func TestValidateRejectsBadSprites(t *testing.T) {
	tests := []struct {
		name   string
		sprite Sprite
	}{
		{"zero width", Sprite{Name: "x", Width: 0, Height: 2, Palette: bulbasaurPalette, Pixels: []uint8{}}},
		{"pixel count mismatch", Sprite{Name: "x", Width: 2, Height: 2, Palette: bulbasaurPalette, Pixels: []uint8{1, 2}}},
		{"index out of palette range", Sprite{Name: "x", Width: 1, Height: 1, Palette: bulbasaurPalette, Pixels: []uint8{99}}},
		{"palette too small", Sprite{Name: "x", Width: 1, Height: 1, Palette: Palette{""}, Pixels: []uint8{0}}},
		{"malformed colour", Sprite{Name: "x", Width: 1, Height: 1, Palette: Palette{"", "red"}, Pixels: []uint8{1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.sprite.Validate(); err == nil {
				t.Error("Validate() = nil, want an error")
			}
		})
	}
}

func TestByDex(t *testing.T) {
	s, err := ByDex(1)
	if err != nil {
		t.Fatalf("ByDex(1) returned %v", err)
	}
	if s.Name != "Bulbasaur" {
		t.Errorf("ByDex(1).Name = %q, want Bulbasaur", s.Name)
	}
	if _, err := ByDex(999); err == nil {
		t.Error("ByDex(999) = nil error, want an error")
	}
}

func TestRegisteredDexIsSorted(t *testing.T) {
	dex := RegisteredDex()
	if len(dex) == 0 {
		t.Fatal("RegisteredDex() is empty")
	}
	for i := 1; i < len(dex); i++ {
		if dex[i-1] >= dex[i] {
			t.Errorf("RegisteredDex() not ascending: %v", dex)
			break
		}
	}
}

func TestAllRegisteredSpritesValidate(t *testing.T) {
	for _, d := range RegisteredDex() {
		s, err := ByDex(d)
		if err != nil {
			t.Fatalf("ByDex(%d): %v", d, err)
		}
		if err := s.Validate(); err != nil {
			t.Errorf("sprite #%03d: %v", d, err)
		}
	}
}

func TestBodyColor(t *testing.T) {
	tests := []struct {
		name   string
		sprite Sprite
		want   Color
		wantOK bool
	}{
		{"declared body index", Bulbasaur, "#74C47A", true},
		{"unset index falls back to the first colour",
			Sprite{Palette: Palette{"", "#ABCDEF", "#123456"}}, "#ABCDEF", true},
		{"index beyond the palette falls back",
			Sprite{Palette: Palette{"", "#ABCDEF"}, BodyIndex: 9}, "#ABCDEF", true},
		{"no palette at all", Sprite{}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.sprite.BodyColor()
			if ok != tt.wantOK {
				t.Fatalf("BodyColor() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("BodyColor() = %q, want %q", got, tt.want)
			}
		})
	}
}

// At and ASCII are exported and must not panic on sprites that never passed
// Validate.
func TestAtToleratesUnvalidatedSprites(t *testing.T) {
	short := Sprite{Name: "short", Width: 2, Height: 2, Pixels: []uint8{1}}
	if got := short.At(1, 1); got != TransparentIndex {
		t.Errorf("At(1, 1) on a short pixel slice = %d, want TransparentIndex", got)
	}
	if _, ok := short.ColorAt(0, 0); ok {
		t.Error("ColorAt reported a colour for an index outside the palette")
	}
	_ = short.ASCII() // must not panic
}

func TestASCIIUsesHexDigits(t *testing.T) {
	s := Sprite{
		Width:   3,
		Height:  1,
		Palette: make(Palette, 20),
		Pixels:  []uint8{0, 10, 15},
	}
	if got, want := s.ASCII(), ".af\n"; got != want {
		t.Errorf("ASCII() = %q, want %q", got, want)
	}
}

func TestASCIIMarksIndicesBeyondTheDigitTable(t *testing.T) {
	s := Sprite{
		Width:   1,
		Height:  1,
		Palette: make(Palette, 30),
		Pixels:  []uint8{20},
	}
	if got, want := s.ASCII(), "?\n"; got != want {
		t.Errorf("ASCII() = %q, want %q", got, want)
	}
}
