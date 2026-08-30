package sprite

import (
	"fmt"
	"sort"
)

// registry maps a Pokédex number to its sprite. Adding a new Pokémon means
// adding one data file and one entry here.
var registry = map[int]Sprite{
	Bulbasaur.Dex: Bulbasaur,
}

// ByDex returns the sprite for a Pokédex number.
func ByDex(dex int) (Sprite, error) {
	s, ok := registry[dex]
	if !ok {
		return Sprite{}, fmt.Errorf("no sprite registered for Pokédex #%03d (available: %v)", dex, RegisteredDex())
	}
	return s, nil
}

// RegisteredDex lists the available Pokédex numbers in ascending order.
func RegisteredDex() []int {
	dex := make([]int, 0, len(registry))
	for d := range registry {
		dex = append(dex, d)
	}
	sort.Ints(dex)
	return dex
}
