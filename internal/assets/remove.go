package assets

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// LoadRemoveMasks reads REMOVEAN.IMG which contains 15 frames of 20-word masks.
// Each frame has 20 scanlines; each word has 16 bits mapping to 16 pixels (MSB=leftmost).
func LoadRemoveMasks(path string) ([][20]uint16, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("remove masks decode: %w", err)
	}
	b := img.Bounds()
	if b.Dx() != 16 || b.Dy()%20 != 0 {
		return nil, fmt.Errorf("remove masks unexpected size %dx%d", b.Dx(), b.Dy())
	}
	frames := b.Dy() / 20
	if frames < 15 {
		return nil, fmt.Errorf("remove masks need 15 frames, got %d", frames)
	}
	masks := make([][20]uint16, 15)
	for fidx := 0; fidx < 15; fidx++ {
		for y := 0; y < 20; y++ {
			var word uint16
			for x := 0; x < 16; x++ {
				_, _, _, a := img.At(b.Min.X+x, b.Min.Y+fidx*20+y).RGBA()
				if a >= 0x8000 {
					word |= 1 << (15 - x)
				}
			}
			masks[fidx][y] = word
		}
	}
	return masks, nil
}

// MaskImage generates a 16x20 alpha image from a 20-word mask (1=opaque, 0=transparent).
func MaskImage(mask *[20]uint16) *image.Alpha {
	img := image.NewAlpha(image.Rect(0, 0, 16, 20))
	for y := 0; y < 20; y++ {
		m := mask[y]
		for x := 0; x < 16; x++ {
			bit := (m >> (15 - x)) & 1
			if bit != 0 {
				img.SetAlpha(x, y, color.Alpha{A: 0xFF})
			} else {
				img.SetAlpha(x, y, color.Alpha{A: 0x00})
			}
		}
	}
	return img
}
