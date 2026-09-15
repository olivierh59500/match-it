package game

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	resources "github.com/olivierh59500/match-it/assets"
	assetdecoder "github.com/olivierh59500/match-it/internal/assets"
)

// removeAnim references preloaded frames instead of creating textures when a
// pair is selected.
type removeAnim struct {
	x1, y1 int
	x2, y2 int
	tile1  int
	tile2  int
	frame  int
}

func (g *Game) loadRemovalFrames(a *atlas, tileSources []*image.RGBA) {
	f, err := resources.Files.Open("png/remove/removean.img.png")
	if err != nil {
		log.Printf("removal masks: %v", err)
		return
	}
	masks, decodeErr := assetdecoder.DecodeRemoveMasks(f)
	_ = f.Close()
	if decodeErr != nil {
		log.Printf("removal masks: %v", decodeErr)
		return
	}

	const tileWidth, tileHeight = 16, 20
	sheet := image.NewRGBA(image.Rect(0, 0, playableTileCount*tileWidth, removeFrameCount*tileHeight))
	for tile := 0; tile < playableTileCount; tile++ {
		source := tileSources[tile]
		if source == nil {
			continue
		}
		for frame := 0; frame < removeFrameCount; frame++ {
			for y := 0; y < tileHeight; y++ {
				mask := masks[frame][y]
				for x := 0; x < tileWidth; x++ {
					if mask&(1<<uint(15-x)) == 0 {
						continue
					}
					sheet.SetRGBA(tile*tileWidth+x, frame*tileHeight+y, source.RGBAAt(x, y))
				}
			}
		}
	}

	a.RemovalSheet = ebiten.NewImageFromImage(sheet)
	for tile := 0; tile < playableTileCount; tile++ {
		for frame := 0; frame < removeFrameCount; frame++ {
			rect := image.Rect(
				tile*tileWidth,
				frame*tileHeight,
				(tile+1)*tileWidth,
				(frame+1)*tileHeight,
			)
			a.RemovalFrames[tile][frame] = a.RemovalSheet.SubImage(rect).(*ebiten.Image)
		}
	}
}

func (g *Game) startRemoveAnim(x1, y1, x2, y2, tile1, tile2 int) {
	if g.atlas == nil || tile1 < 0 || tile1 >= playableTileCount || tile2 < 0 || tile2 >= playableTileCount ||
		g.atlas.RemovalFrames[tile1][0] == nil || g.atlas.RemovalFrames[tile2][0] == nil {
		g.commitPair(x1, y1, x2, y2)
		return
	}
	g.anim = &removeAnim{
		x1:    x1,
		y1:    y1,
		x2:    x2,
		y2:    y2,
		tile1: tile1,
		tile2: tile2,
	}
	g.boardCanvasDirty = true
}
