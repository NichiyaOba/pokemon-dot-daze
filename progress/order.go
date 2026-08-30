// Package progress turns a contribution count into a partially revealed sprite.
package progress

import "github.com/NichiyaOba/pokemon-dot-daze/sprite"

// Point is a cell coordinate within a sprite.
type Point struct {
	X int
	Y int
}

// RevealSequence returns the non-transparent cells of s in the order they
// light up: raster scan, top row first, left to right within a row.
//
// The order is fully determined by the sprite, so the same sprite always
// produces the same sequence. Rendering depends on that determinism.
func RevealSequence(s sprite.Sprite) []Point {
	seq := make([]Point, 0, s.DotCount())
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			if s.At(x, y) == sprite.TransparentIndex {
				continue
			}
			seq = append(seq, Point{X: x, Y: y})
		}
	}
	return seq
}
