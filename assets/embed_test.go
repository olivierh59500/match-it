package resources

import (
	"bytes"
	"fmt"
	"image/png"
	"testing"
)

func TestRuntimeAssetsAreEmbedded(t *testing.T) {
	for _, name := range []string{
		"png/gamearea.img.png",
		"png/tile_00.png",
		"png/font2/glyph_00.png",
		"png/remove/removean.img.png",
		"music/Chambers of Shaolin - Trapped in China.ym",
	} {
		data, err := Files.ReadFile(name)
		if err != nil {
			t.Errorf("read embedded %q: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("embedded %q is empty", name)
		}
	}
}

// A normal tile must cover the background completely. Transparency belongs to
// the separate removal masks, not to the green parts of a tile's artwork.
func TestPlayableTilePNGsAreOpaque(t *testing.T) {
	for i := 0; i < 43; i++ {
		name := fmt.Sprintf("png/tile_%02d.png", i)
		t.Run(name, func(t *testing.T) {
			data, err := Files.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			bounds := img.Bounds()
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					_, _, _, alpha := img.At(x, y).RGBA()
					if alpha != 0xffff {
						t.Fatalf("pixel (%d,%d) has alpha %d: the board background can alter the tile's colors", x, y, alpha)
					}
				}
			}
		})
	}
}
