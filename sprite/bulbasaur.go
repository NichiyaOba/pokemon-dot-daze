package sprite

// bulbasaurPalette follows the eight-colour legend of the source bead chart.
// Index 0 is the reserved transparent slot and is never rendered.
var bulbasaurPalette = Palette{
	"",        // 0: transparent (outside the artwork)
	"#90DA92", // 1: bright yellow-green highlight
	"#74C47A", // 2: body green
	"#55A05E", // 3: shaded green
	"#3F7A4C", // 4: deep green
	"#3C3C3C", // 5: dark grey shading
	"#B4544A", // 6: red (eye)
	"#FFFFFF", // 7: white (eye highlight, teeth)
	"#000000", // 8: black outline
}

// Bulbasaur is Pokédex #001, transcribed from a 20x19 bead chart.
//
// The transcription is guarded by TestBulbasaurMatchesChartLegend, which
// compares the per-colour histogram against the legend printed on the chart.
var Bulbasaur = Sprite{
	Dex:       1,
	Name:      "Bulbasaur",
	Width:     20,
	Height:    19,
	Palette:   bulbasaurPalette,
	BodyIndex: 2, // body green
	Pixels: []uint8{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 8, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 1, 8, 1, 8, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 8, 2, 1, 2, 8, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 8, 8, 2, 3, 3, 2, 3, 2, 8, 8, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 8, 2, 2, 3, 2, 3, 2, 3, 2, 2, 3, 8, 0,
		0, 0, 0, 8, 0, 0, 8, 4, 2, 3, 2, 2, 3, 2, 2, 3, 2, 2, 3, 8,
		0, 0, 8, 1, 8, 8, 5, 4, 3, 4, 2, 3, 2, 2, 2, 2, 3, 2, 4, 8,
		0, 0, 8, 1, 1, 1, 2, 5, 5, 4, 4, 3, 2, 2, 2, 2, 3, 4, 4, 8,
		0, 0, 8, 1, 1, 2, 3, 3, 2, 5, 4, 3, 4, 4, 4, 4, 3, 4, 4, 8,
		0, 8, 1, 1, 1, 3, 3, 3, 1, 2, 5, 5, 5, 4, 4, 4, 3, 4, 8, 0,
		0, 8, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 5, 4, 4, 3, 5, 8, 0, 0,
		8, 6, 3, 1, 1, 1, 2, 1, 1, 1, 1, 5, 2, 5, 5, 5, 2, 8, 0, 0,
		8, 1, 1, 1, 1, 3, 1, 5, 8, 5, 1, 2, 2, 2, 2, 2, 3, 3, 8, 0,
		0, 8, 1, 1, 1, 1, 1, 8, 8, 7, 7, 2, 2, 2, 5, 2, 2, 3, 3, 8,
		0, 8, 4, 1, 1, 1, 8, 6, 7, 2, 2, 2, 3, 3, 5, 2, 2, 2, 8, 0,
		0, 0, 8, 8, 2, 2, 2, 2, 2, 2, 5, 2, 3, 3, 5, 2, 1, 8, 0, 0,
		0, 0, 0, 0, 8, 8, 8, 8, 8, 8, 2, 2, 2, 2, 5, 8, 8, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 7, 2, 7, 5, 8, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 8, 8, 8, 0, 0, 0, 0, 0, 0,
	},
}
