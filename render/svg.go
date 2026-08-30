package render

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/NichiyaOba/pokemon-dot-daze/progress"
	"github.com/NichiyaOba/pokemon-dot-daze/sprite"
)

// Caption block metrics, in pixels.
const (
	captionGap      = 10
	captionBarH     = 6
	captionTextGap  = 6
	captionFontSize = 13
	captionLineH    = 16
	captionHeight   = captionGap + captionBarH + captionTextGap + captionLineH
)

const fontStack = "ui-monospace,SFMono-Regular,Menlo,Consolas,monospace"

// captionCharWidthNum/Den approximate the advance width of one monospace glyph
// as a fraction of the font size. 0.6em is the usual figure for the stack above
// and errs slightly wide, so the caption is never clipped.
const (
	captionCharWidthNum = 6
	captionCharWidthDen = 10
)

// captionWidth estimates the rendered width of the caption in pixels.
func captionWidth(text string) int {
	return len([]rune(text)) * captionFontSize * captionCharWidthNum / captionCharWidthDen
}

// SVG renders p as a standalone SVG document.
//
// Output is deterministic: the same progress and options always produce
// byte-identical output. CI relies on this so an unchanged picture produces
// no commit.
func SVG(p progress.Progress, opts Options) ([]byte, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid render options: %w", err)
	}
	if err := p.Sprite.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sprite: %w", err)
	}

	step := opts.CellSize + opts.Gap
	artW := p.Sprite.Width*step - opts.Gap
	artH := p.Sprite.Height*step - opts.Gap

	contentW := artW
	if opts.ShowCaption {
		// The caption is usually wider than the artwork at small cell sizes;
		// sizing the canvas to the artwork alone clips it.
		if w := captionWidth(captionText(p)); w > contentW {
			contentW = w
		}
	}

	totalW := contentW + opts.Padding*2
	totalH := artH + opts.Padding*2
	if opts.ShowCaption {
		totalH += captionHeight
	}

	var b bytes.Buffer
	b.Grow(p.Sprite.Width*p.Sprite.Height*64 + 512)

	writeHeader(&b, totalW, totalH)
	writeStyle(&b, opts)
	writeDots(&b, p, opts, step)
	if opts.ShowCaption {
		writeCaption(&b, p, opts, contentW, artH)
	}
	b.WriteString("</svg>\n")

	return b.Bytes(), nil
}

func writeHeader(b *bytes.Buffer, w, h int) {
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="`)
	b.WriteString(strconv.Itoa(w))
	b.WriteString(`" height="`)
	b.WriteString(strconv.Itoa(h))
	b.WriteString(`" viewBox="0 0 `)
	b.WriteString(strconv.Itoa(w))
	b.WriteString(" ")
	b.WriteString(strconv.Itoa(h))
	b.WriteString(`" role="img" shape-rendering="crispEdges">`)
	b.WriteString("\n")
}

// writeStyle emits the colours that depend on the viewer's colour scheme.
// Keeping the unlit fill in CSS rather than on every rect keeps the file small,
// since unlit dots dominate early on.
func writeStyle(b *bytes.Buffer, opts Options) {
	unlit, caption := opts.UnlitLight, opts.CaptionLight
	if opts.Theme == ThemeDark {
		unlit, caption = opts.UnlitDark, opts.CaptionDark
	}

	b.WriteString("<style>\n")
	fmt.Fprintf(b, ".u{fill:%s}.bar{fill:%s}\n", unlit, unlit)
	fmt.Fprintf(b, ".cap{fill:%s;font-family:%s;font-size:%dpx;font-weight:600}\n", caption, fontStack, captionFontSize)

	if opts.Theme == ThemeAuto {
		b.WriteString("@media(prefers-color-scheme:dark){")
		fmt.Fprintf(b, ".u{fill:%s}.bar{fill:%s}", opts.UnlitDark, opts.UnlitDark)
		fmt.Fprintf(b, ".cap{fill:%s}", opts.CaptionDark)
		b.WriteString("}\n")
	}
	b.WriteString("</style>\n")
}

func writeDots(b *bytes.Buffer, p progress.Progress, opts Options, step int) {
	mask := p.LitMask()

	b.WriteString(`<g transform="translate(`)
	b.WriteString(strconv.Itoa(opts.Padding))
	b.WriteString(",")
	b.WriteString(strconv.Itoa(opts.Padding))
	b.WriteString(`)">`)
	b.WriteString("\n")

	for y := 0; y < p.Sprite.Height; y++ {
		for x := 0; x < p.Sprite.Width; x++ {
			color, opaque := p.Sprite.ColorAt(x, y)
			if !opaque {
				continue
			}
			writeRect(b, x*step, y*step, opts.CellSize, color, mask[y*p.Sprite.Width+x])
		}
	}

	b.WriteString("</g>\n")
}

func writeRect(b *bytes.Buffer, x, y, size int, color sprite.Color, lit bool) {
	b.WriteString(`<rect x="`)
	b.WriteString(strconv.Itoa(x))
	b.WriteString(`" y="`)
	b.WriteString(strconv.Itoa(y))
	b.WriteString(`" width="`)
	b.WriteString(strconv.Itoa(size))
	b.WriteString(`" height="`)
	b.WriteString(strconv.Itoa(size))
	if lit {
		b.WriteString(`" fill="`)
		b.WriteString(string(color))
		b.WriteString(`"/>`)
	} else {
		b.WriteString(`" class="u"/>`)
	}
	b.WriteString("\n")
}

func writeCaption(b *bytes.Buffer, p progress.Progress, opts Options, contentW, artH int) {
	top := opts.Padding + artH + captionGap

	b.WriteString(`<g transform="translate(`)
	b.WriteString(strconv.Itoa(opts.Padding))
	b.WriteString(",")
	b.WriteString(strconv.Itoa(top))
	b.WriteString(`)">`)
	b.WriteString("\n")

	// Track, then the filled portion on top of it.
	fmt.Fprintf(b, `<rect x="0" y="0" width="%d" height="%d" rx="%d" class="bar"/>`+"\n",
		contentW, captionBarH, captionBarH/2)

	if filled := int(float64(contentW) * p.Ratio()); filled > 0 {
		fmt.Fprintf(b, `<rect x="0" y="0" width="%d" height="%d" rx="%d" fill="%s"/>`+"\n",
			filled, captionBarH, captionBarH/2, barColor(p.Sprite))
	}

	textY := captionBarH + captionTextGap + captionFontSize
	fmt.Fprintf(b, `<text x="0" y="%d" class="cap">%s</text>`+"\n", textY, escapeXML(captionText(p)))

	b.WriteString("</g>\n")
}

// fallbackBarColor is used when a sprite declares no usable body colour.
const fallbackBarColor sprite.Color = "#74C47A"

// barColor uses the sprite's own body colour so the bar matches the artwork.
func barColor(s sprite.Sprite) sprite.Color {
	if c, ok := s.BodyColor(); ok {
		return c
	}
	return fallbackBarColor
}

func captionText(p progress.Progress) string {
	return fmt.Sprintf("#%03d %s  %s / %s  (%d%%)",
		p.Sprite.Dex,
		p.Sprite.Name,
		humanize(p.Commits),
		humanize(p.CommitsRequired()),
		p.Percent(),
	)
}

// humanize formats n with thousands separators, e.g. 12345 -> "12,345".
func humanize(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	var out strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(r)
	}
	if neg {
		return "-" + out.String()
	}
	return out.String()
}

var xmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&apos;",
)

func escapeXML(s string) string { return xmlEscaper.Replace(s) }
