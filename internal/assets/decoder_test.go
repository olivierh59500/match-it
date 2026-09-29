package assets

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestTileSetPreservesAllPaletteColors(t *testing.T) {
	// Each row contains palette indices 0..15, encoded in four ST bitplanes.
	// Include the path sprites as well as the 43 playable tiles.
	raw := make([]byte, 49*160)
	for row := 0; row < 49*20; row++ {
		for plane := 0; plane < 4; plane++ {
			var bits uint16
			for x := 0; x < 16; x++ {
				if x&(1<<plane) != 0 {
					bits |= 1 << (15 - x)
				}
			}
			binary.BigEndian.PutUint16(raw[row*8+plane*2:], bits)
		}
	}
	path := filepath.Join(t.TempDir(), "tiles.img")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	palette := Palette2()
	tiles, err := TileSetFromTilesIMG(path, palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(tiles) != 49 {
		t.Fatalf("decoded %d sprites, want 49", len(tiles))
	}
	for i, tile := range tiles {
		for y := 0; y < 20; y++ {
			for x := 0; x < 16; x++ {
				if got := tile.RGBAAt(x, y); got != palette[x] {
					t.Fatalf("tile %d pixel (%d,%d): got %v, want palette[%d]=%v", i, x, y, got, x, palette[x])
				}
			}
		}
	}
}
