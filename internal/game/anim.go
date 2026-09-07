package game

import (
	"image"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
	resources "github.com/olivierh59500/match-it/assets"
	"github.com/olivierh59500/match-it/internal/assets"
)

// removeAnim holds state for a two-tile removal animation using REMOVEAN.IMG masks.
type removeAnim struct {
	x1, y1 int
	x2, y2 int
	idx1   int // tile index (0-based) for the sprite atlas
	idx2   int
	frame  int
	frames int
	img1   []*ebiten.Image // precomputed masked frames for tile1
	img2   []*ebiten.Image // precomputed masked frames for tile2
}

func (g *Game) startRemoveAnim(x1, y1, x2, y2, idx1, idx2 int) {
	if g.atlas == nil || g.atlas.Tiles == nil {
		return
	}
	// Load masks once.
	if g.removeMasks == nil {
		f, err := resources.Files.Open("png/remove/removean.img.png")
		if err != nil {
			g.board.RemovePair(x1, y1, x2, y2)
			return
		}
		masks, decodeErr := assets.DecodeRemoveMasks(f)
		_ = f.Close()
		if decodeErr == nil {
			g.removeMasks = masks
		} else {
			// Fallback: no animation
			g.board.RemovePair(x1, y1, x2, y2)
			return
		}
	}
	// Precompute masked frames for both tiles.
	base1 := g.atlas.Tiles[idx1]
	base2 := g.atlas.Tiles[idx2]
	imgs1 := make([]*ebiten.Image, 15)
	imgs2 := make([]*ebiten.Image, 15)
	for f := 0; f < 15; f++ {
		m := assets.MaskImage(&g.removeMasks[f])
		imgs1[f] = ebiten.NewImageFromImage(applyMask(base1, m))
		imgs2[f] = ebiten.NewImageFromImage(applyMask(base2, m))
	}
	g.anim = &removeAnim{
		x1: x1, y1: y1, x2: x2, y2: y2, idx1: idx1, idx2: idx2,
		frame: 0, frames: 15, img1: imgs1, img2: imgs2,
	}
}

// applyMask multiplies the alpha channel of src by mask (mask alpha 0/255) and returns a new RGBA image.
func applyMask(src image.Image, mask *image.Alpha) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	// Apply mask by zeroing alpha where mask is 0.
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			a := mask.AlphaAt(x-b.Min.X, y-b.Min.Y).A
			if a == 0 {
				c := out.RGBAAt(x, y)
				c.A = 0
				out.SetRGBA(x, y, c)
			}
		}
	}
	return out
}
