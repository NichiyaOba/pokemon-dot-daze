package render

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/NichiyaOba/pokemon-dot-daze/progress"
	"github.com/NichiyaOba/pokemon-dot-daze/sprite"
)

func mustRender(t *testing.T, commits int, opts Options) []byte {
	t.Helper()
	p, err := progress.Compute(sprite.Bulbasaur, commits)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	out, err := SVG(p, opts)
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	return out
}

func TestSVGIsWellFormedXML(t *testing.T) {
	for _, commits := range []int{0, 1000, 2520, 99999} {
		out := mustRender(t, commits, DefaultOptions())
		dec := xml.NewDecoder(bytes.NewReader(out))
		for {
			_, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("commits=%d: SVG is not well-formed XML: %v", commits, err)
			}
		}
	}
}

// countRects counts <rect> elements, split by whether they carry the unlit class.
func countRects(t *testing.T, out []byte) (lit, unlit int) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(out))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return lit, unlit
		}
		if err != nil {
			t.Fatalf("parsing SVG: %v", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "rect" {
			continue
		}
		isUnlit := false
		isDot := false
		for _, a := range se.Attr {
			if a.Name.Local == "class" && a.Value == "u" {
				isUnlit, isDot = true, true
			}
			if a.Name.Local == "fill" && strings.HasPrefix(a.Value, "#") {
				isDot = true
			}
		}
		if !isDot {
			continue
		}
		if isUnlit {
			unlit++
		} else {
			lit++
		}
	}
}

func TestRectCountsTrackProgress(t *testing.T) {
	total := sprite.Bulbasaur.DotCount()

	tests := []struct {
		name    string
		commits int
		wantLit int
	}{
		{"nothing revealed", 0, 0},
		{"one dot", 10, 1},
		{"hundred dots", 1000, 100},
		{"complete", total * progress.CommitsPerDot, total},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Caption off so its bar rects do not pollute the count.
			opts := DefaultOptions()
			opts.ShowCaption = false

			lit, unlit := countRects(t, mustRender(t, tt.commits, opts))

			if lit != tt.wantLit {
				t.Errorf("lit rects = %d, want %d", lit, tt.wantLit)
			}
			if want := total - tt.wantLit; unlit != want {
				t.Errorf("unlit rects = %d, want %d", unlit, want)
			}
			if lit+unlit != total {
				t.Errorf("total rects = %d, want %d (one per opaque cell)", lit+unlit, total)
			}
		})
	}
}

// Determinism underpins the CI commit-back step: an unchanged picture must
// produce an identical file so git sees no diff.
func TestSVGIsDeterministic(t *testing.T) {
	for i := 0; i < 20; i++ {
		a := mustRender(t, 1234, DefaultOptions())
		b := mustRender(t, 1234, DefaultOptions())
		if !bytes.Equal(a, b) {
			t.Fatal("two renders of the same input differ")
		}
	}
}

func TestAutoThemeEmitsMediaQuery(t *testing.T) {
	opts := DefaultOptions()
	opts.Theme = ThemeAuto
	if !bytes.Contains(mustRender(t, 100, opts), []byte("prefers-color-scheme:dark")) {
		t.Error("auto theme output has no prefers-color-scheme rule")
	}
}

func TestFixedThemesOmitMediaQuery(t *testing.T) {
	for _, theme := range []Theme{ThemeLight, ThemeDark} {
		opts := DefaultOptions()
		opts.Theme = theme
		out := mustRender(t, 100, opts)
		if bytes.Contains(out, []byte("prefers-color-scheme")) {
			t.Errorf("theme %q emitted a media query", theme)
		}
	}
}

func TestDarkThemeUsesDarkUnlitColor(t *testing.T) {
	opts := DefaultOptions()
	opts.Theme = ThemeDark
	out := mustRender(t, 100, opts)
	if !bytes.Contains(out, []byte(string(DefaultUnlitDark))) {
		t.Errorf("dark theme output does not contain %s", DefaultUnlitDark)
	}
}

func TestCaptionTogglesOutput(t *testing.T) {
	on := DefaultOptions()
	on.ShowCaption = true
	if !bytes.Contains(mustRender(t, 1000, on), []byte("Bulbasaur")) {
		t.Error("caption enabled but the sprite name is absent")
	}

	off := DefaultOptions()
	off.ShowCaption = false
	if bytes.Contains(mustRender(t, 1000, off), []byte("Bulbasaur")) {
		t.Error("caption disabled but the sprite name is present")
	}
}

func TestCaptionShowsProgress(t *testing.T) {
	out := string(mustRender(t, 1000, DefaultOptions()))
	for _, want := range []string{"#001", "Bulbasaur", "1,000", "2,520", "39%"} {
		if !strings.Contains(out, want) {
			t.Errorf("caption is missing %q", want)
		}
	}
}

func TestCanvasScalesWithCellSize(t *testing.T) {
	tests := []struct {
		cellSize int
		wantW    int // 20 cells * cellSize + 2 * 12px padding
	}{
		{8, 184},
		{16, 344},
		{24, 504},
	}
	for _, tt := range tests {
		opts := DefaultOptions()
		opts.ShowCaption = false
		opts.CellSize = tt.cellSize
		want := []byte(fmt.Sprintf("viewBox=\"0 0 %d ", tt.wantW))
		if !bytes.Contains(mustRender(t, 100, opts), want) {
			t.Errorf("cell size %d: canvas width is not %d", tt.cellSize, tt.wantW)
		}
	}
}

// The caption is wider than the artwork at small cell sizes, so the canvas
// must grow to fit it rather than clipping the text.
func TestCanvasWidensForTheCaption(t *testing.T) {
	opts := DefaultOptions()
	opts.CellSize = 8

	p, err := progress.Compute(sprite.Bulbasaur, 1000)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	out, err := SVG(p, opts)
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}

	artOnlyWidth := sprite.Bulbasaur.Width*8 + 2*DefaultPadding
	wantAtLeast := captionWidth(captionText(p)) + 2*DefaultPadding
	if wantAtLeast <= artOnlyWidth {
		t.Fatal("test premise broken: the caption is not wider than the artwork here")
	}

	if !bytes.Contains(out, []byte(fmt.Sprintf(`width="%d"`, wantAtLeast))) {
		t.Errorf("canvas was not widened to %d to fit the caption", wantAtLeast)
	}
}

func TestGapProducesBeadLayout(t *testing.T) {
	opts := DefaultOptions()
	opts.ShowCaption = false
	opts.CellSize = 10
	opts.Gap = 2
	// 20 cells * 10 + 19 gaps * 2 = 238, + 24 padding = 262
	if !bytes.Contains(mustRender(t, 100, opts), []byte("viewBox=\"0 0 262 ")) {
		t.Error("gap was not applied to the canvas width")
	}
}

func TestSVGRejectsBadOptions(t *testing.T) {
	p, err := progress.Compute(sprite.Bulbasaur, 100)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	tests := []struct {
		name string
		mut  func(*Options)
	}{
		{"zero cell size", func(o *Options) { o.CellSize = 0 }},
		{"negative gap", func(o *Options) { o.Gap = -1 }},
		{"negative padding", func(o *Options) { o.Padding = -1 }},
		{"unknown theme", func(o *Options) { o.Theme = "neon" }},
		{"malformed unlit colour", func(o *Options) { o.UnlitLight = "grey" }},
		{"malformed caption colour", func(o *Options) { o.CaptionDark = "#GGG" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			tt.mut(&opts)
			if _, err := SVG(p, opts); err == nil {
				t.Error("SVG() = nil error, want an error")
			}
		})
	}
}

// A hand-built Progress must not panic the renderer.
func TestSVGToleratesHandBuiltProgress(t *testing.T) {
	p := progress.Progress{
		Sprite:   sprite.Bulbasaur,
		Revealed: 5, // no Sequence set
		Total:    sprite.Bulbasaur.DotCount(),
	}
	if _, err := SVG(p, DefaultOptions()); err != nil {
		t.Fatalf("SVG with a hand-built Progress returned %v", err)
	}
}

func TestBarColorFallsBackWithoutABodyIndex(t *testing.T) {
	if got := barColor(sprite.Bulbasaur); got != "#74C47A" {
		t.Errorf("barColor(Bulbasaur) = %q, want the declared body colour", got)
	}
	if got := barColor(sprite.Sprite{}); got != fallbackBarColor {
		t.Errorf("barColor(empty) = %q, want %q", got, fallbackBarColor)
	}
}

func TestSVGRejectsInvalidSprite(t *testing.T) {
	p := progress.Progress{
		Sprite: sprite.Sprite{Name: "bad", Width: 2, Height: 2, Pixels: []uint8{1}},
	}
	if _, err := SVG(p, DefaultOptions()); err == nil {
		t.Error("SVG() with an invalid sprite = nil error, want an error")
	}
}

func TestHumanize(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
		{-4321, "-4,321"},
	}
	for _, tt := range tests {
		if got := humanize(tt.in); got != tt.want {
			t.Errorf("humanize(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapeXML(t *testing.T) {
	got := escapeXML(`a & b < c > d " e ' f`)
	want := `a &amp; b &lt; c &gt; d &quot; e &apos; f`
	if got != want {
		t.Errorf("escapeXML = %q, want %q", got, want)
	}
}
