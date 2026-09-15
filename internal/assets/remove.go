package assets

import (
	"fmt"
	"image/png"
	"io"
)

// DecodeRemoveMasks decodes the removal animation from a PNG stream.
func DecodeRemoveMasks(r io.Reader) ([][20]uint16, error) {
	img, err := png.Decode(r)
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
