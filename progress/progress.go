package progress

import (
	"fmt"

	"github.com/NichiyaOba/pokemon-dot-daze/sprite"
)

// CommitsPerDot is how many GitHub contributions light one dot.
const CommitsPerDot = 10

// Progress is an immutable snapshot of how far a sprite has been revealed.
type Progress struct {
	Sprite   sprite.Sprite
	Commits  int
	Revealed int
	Total    int
	// Sequence is the reveal order; the first Revealed entries are lit.
	Sequence []Point
}

// Compute derives the progress of s at the given contribution count.
func Compute(s sprite.Sprite, commits int) (Progress, error) {
	if commits < 0 {
		return Progress{}, fmt.Errorf("contribution count must not be negative, got %d", commits)
	}
	if err := s.Validate(); err != nil {
		return Progress{}, fmt.Errorf("invalid sprite: %w", err)
	}

	seq := RevealSequence(s)
	total := len(seq)

	revealed := commits / CommitsPerDot
	if revealed > total {
		revealed = total
	}

	return Progress{
		Sprite:   s,
		Commits:  commits,
		Revealed: revealed,
		Total:    total,
		Sequence: seq,
	}, nil
}

// revealedCount clamps Revealed to the sequence actually available. Progress
// is normally built by Compute, but the package is exported and a caller can
// construct one directly; clamping keeps that a wrong picture rather than a
// panic.
func (p Progress) revealedCount() int {
	if p.Revealed < 0 {
		return 0
	}
	if p.Revealed > len(p.Sequence) {
		return len(p.Sequence)
	}
	return p.Revealed
}

// IsLit reports whether the cell at (x, y) has been revealed. Transparent
// cells are never lit.
func (p Progress) IsLit(x, y int) bool {
	n := p.revealedCount()
	for i := 0; i < n; i++ {
		if p.Sequence[i].X == x && p.Sequence[i].Y == y {
			return true
		}
	}
	return false
}

// LitMask returns a Width*Height slice, true where a cell is lit. This is the
// form renderers want: building it once is O(n) instead of O(n^2) repeated
// IsLit calls.
func (p Progress) LitMask() []bool {
	mask := make([]bool, p.Sprite.Width*p.Sprite.Height)
	n := p.revealedCount()
	for i := 0; i < n; i++ {
		pt := p.Sequence[i]
		idx := pt.Y*p.Sprite.Width + pt.X
		if idx < 0 || idx >= len(mask) {
			continue
		}
		mask[idx] = true
	}
	return mask
}

// Ratio is the fraction of dots revealed, in [0, 1].
func (p Progress) Ratio() float64 {
	if p.Total == 0 {
		return 0
	}
	return float64(p.Revealed) / float64(p.Total)
}

// Percent is Ratio as a whole-number percentage, truncated.
func (p Progress) Percent() int {
	return int(p.Ratio() * 100)
}

// IsComplete reports whether every dot is lit.
func (p Progress) IsComplete() bool {
	return p.Revealed >= p.Total
}

// CommitsRequired is the contribution count that completes the sprite.
func (p Progress) CommitsRequired() int {
	return p.Total * CommitsPerDot
}

// CommitsRemaining is how many more contributions are needed to finish.
// It is 0 once the sprite is complete.
func (p Progress) CommitsRemaining() int {
	remaining := p.CommitsRequired() - p.Commits
	if remaining < 0 {
		return 0
	}
	return remaining
}
