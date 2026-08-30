package progress

import (
	"testing"

	"github.com/NichiyaOba/pokemon-dot-daze/sprite"
)

const bulbasaurDots = 252

func mustCompute(t *testing.T, commits int) Progress {
	t.Helper()
	p, err := Compute(sprite.Bulbasaur, commits)
	if err != nil {
		t.Fatalf("Compute(Bulbasaur, %d) returned %v", commits, err)
	}
	return p
}

func TestComputeRevealedBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		commits  int
		revealed int
	}{
		{"zero commits lights nothing", 0, 0},
		{"nine commits is still short of one dot", 9, 0},
		{"ten commits lights exactly one dot", 10, 1},
		{"eleven commits still lights one dot", 11, 1},
		{"one short of complete", bulbasaurDots*CommitsPerDot - 10, bulbasaurDots - 1},
		{"exactly complete", bulbasaurDots * CommitsPerDot, bulbasaurDots},
		{"overflow clamps to total", bulbasaurDots*CommitsPerDot + 5000, bulbasaurDots},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := mustCompute(t, tt.commits)
			if p.Revealed != tt.revealed {
				t.Errorf("Revealed = %d, want %d", p.Revealed, tt.revealed)
			}
			if p.Total != bulbasaurDots {
				t.Errorf("Total = %d, want %d", p.Total, bulbasaurDots)
			}
		})
	}
}

func TestComputeRejectsNegativeCommits(t *testing.T) {
	if _, err := Compute(sprite.Bulbasaur, -1); err == nil {
		t.Error("Compute with -1 commits = nil error, want an error")
	}
}

func TestComputeRejectsInvalidSprite(t *testing.T) {
	bad := sprite.Sprite{Name: "bad", Width: 2, Height: 2, Pixels: []uint8{1}}
	if _, err := Compute(bad, 10); err == nil {
		t.Error("Compute with an invalid sprite = nil error, want an error")
	}
}

func TestRatioAndPercent(t *testing.T) {
	if got := mustCompute(t, 0).Ratio(); got != 0 {
		t.Errorf("Ratio at 0 commits = %v, want 0", got)
	}
	full := mustCompute(t, bulbasaurDots*CommitsPerDot)
	if got := full.Ratio(); got != 1 {
		t.Errorf("Ratio when complete = %v, want 1", got)
	}
	if got := full.Percent(); got != 100 {
		t.Errorf("Percent when complete = %d, want 100", got)
	}
	half := mustCompute(t, bulbasaurDots/2*CommitsPerDot)
	if got := half.Percent(); got != 50 {
		t.Errorf("Percent at half = %d, want 50", got)
	}
}

func TestIsComplete(t *testing.T) {
	if mustCompute(t, bulbasaurDots*CommitsPerDot-1).IsComplete() {
		t.Error("IsComplete() = true one commit short of the requirement")
	}
	if !mustCompute(t, bulbasaurDots*CommitsPerDot).IsComplete() {
		t.Error("IsComplete() = false at the exact requirement")
	}
}

func TestCommitsRequiredAndRemaining(t *testing.T) {
	p := mustCompute(t, 100)
	if got, want := p.CommitsRequired(), bulbasaurDots*CommitsPerDot; got != want {
		t.Errorf("CommitsRequired() = %d, want %d", got, want)
	}
	if got, want := p.CommitsRemaining(), bulbasaurDots*CommitsPerDot-100; got != want {
		t.Errorf("CommitsRemaining() = %d, want %d", got, want)
	}
	over := mustCompute(t, bulbasaurDots*CommitsPerDot+999)
	if got := over.CommitsRemaining(); got != 0 {
		t.Errorf("CommitsRemaining() past completion = %d, want 0", got)
	}
}

func TestRevealSequenceCoversEveryOpaqueCell(t *testing.T) {
	seq := RevealSequence(sprite.Bulbasaur)

	if len(seq) != bulbasaurDots {
		t.Errorf("len(sequence) = %d, want %d", len(seq), bulbasaurDots)
	}

	seen := make(map[Point]bool, len(seq))
	for _, pt := range seq {
		if seen[pt] {
			t.Errorf("point %+v appears more than once", pt)
		}
		seen[pt] = true
		if sprite.Bulbasaur.At(pt.X, pt.Y) == sprite.TransparentIndex {
			t.Errorf("point %+v is transparent but appears in the sequence", pt)
		}
	}

	// Every opaque cell must be present.
	for y := 0; y < sprite.Bulbasaur.Height; y++ {
		for x := 0; x < sprite.Bulbasaur.Width; x++ {
			if sprite.Bulbasaur.At(x, y) == sprite.TransparentIndex {
				continue
			}
			if !seen[Point{X: x, Y: y}] {
				t.Errorf("opaque cell (%d, %d) is missing from the sequence", x, y)
			}
		}
	}
}

func TestRevealSequenceIsRasterOrdered(t *testing.T) {
	seq := RevealSequence(sprite.Bulbasaur)
	for i := 1; i < len(seq); i++ {
		prev, cur := seq[i-1], seq[i]
		if cur.Y < prev.Y {
			t.Fatalf("row went backwards at index %d: %+v then %+v", i, prev, cur)
		}
		if cur.Y == prev.Y && cur.X <= prev.X {
			t.Fatalf("column not increasing within row at index %d: %+v then %+v", i, prev, cur)
		}
	}
}

func TestLitMaskMatchesIsLit(t *testing.T) {
	p := mustCompute(t, 500)
	mask := p.LitMask()

	for y := 0; y < p.Sprite.Height; y++ {
		for x := 0; x < p.Sprite.Width; x++ {
			if got, want := mask[y*p.Sprite.Width+x], p.IsLit(x, y); got != want {
				t.Fatalf("LitMask disagrees with IsLit at (%d, %d): %v vs %v", x, y, got, want)
			}
		}
	}
}

func TestLitMaskCountMatchesRevealed(t *testing.T) {
	p := mustCompute(t, 500)
	lit := 0
	for _, v := range p.LitMask() {
		if v {
			lit++
		}
	}
	if lit != p.Revealed {
		t.Errorf("mask has %d lit cells, want %d", lit, p.Revealed)
	}
}

// The first dot to light must be the top-left-most opaque cell, which for
// Bulbasaur is the left ear tip at (12, 0).
func TestFirstDotIsTopLeftMost(t *testing.T) {
	seq := RevealSequence(sprite.Bulbasaur)
	if want := (Point{X: 12, Y: 0}); seq[0] != want {
		t.Errorf("first dot = %+v, want %+v", seq[0], want)
	}
}

// A hand-built Progress (not produced by Compute) must degrade to a wrong
// picture rather than panicking.
func TestHandBuiltProgressDoesNotPanic(t *testing.T) {
	tests := []struct {
		name string
		p    Progress
	}{
		{"revealed without a sequence", Progress{Sprite: sprite.Bulbasaur, Revealed: 5}},
		{"negative revealed", Progress{Sprite: sprite.Bulbasaur, Revealed: -3}},
		{"revealed beyond the sequence", Progress{
			Sprite:   sprite.Bulbasaur,
			Sequence: []Point{{X: 0, Y: 0}},
			Revealed: 99,
		}},
		{"sequence pointing outside the sprite", Progress{
			Sprite:   sprite.Bulbasaur,
			Sequence: []Point{{X: 500, Y: 500}},
			Revealed: 1,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.p.LitMask()
			_ = tt.p.IsLit(0, 0)
		})
	}
}
