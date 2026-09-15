package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) drawBoardBase(screen *ebiten.Image) {
	if g.boardCanvas == nil {
		g.boardCanvas = ebiten.NewImage(logicalWidth, logicalHeight)
		g.boardCanvasDirty = true
	}
	if g.boardCanvasDirty {
		g.rebuildBoardCanvas()
	}
	screen.DrawImage(g.boardCanvas, nil)
}

func (g *Game) rebuildBoardCanvas() {
	g.boardCanvas.Fill(color.RGBA{0, 0, 32, 255})
	if g.atlas.BGTop != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		g.boardCanvas.DrawImage(g.atlas.BGTop, op)
	}
	if g.atlas.BGArea != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(0, 34)
		g.boardCanvas.DrawImage(g.atlas.BGArea, op)
	}

	for y := 0; y < boardRows; y++ {
		for x := 0; x < boardCols; x++ {
			if g.anim != nil && ((x == g.anim.x1 && y == g.anim.y1) || (x == g.anim.x2 && y == g.anim.y2)) {
				continue
			}
			tile := g.board.Get(x, y)
			if tile == 0 {
				continue
			}
			index := int(tile - 1)
			if index < 0 || index >= len(g.atlas.Tiles) || g.atlas.Tiles[index] == nil {
				continue
			}
			drawTileImage(g.boardCanvas, g.atlas.Tiles[index], x, y)
		}
	}
	g.boardCanvasDirty = false
}

func drawTileImage(destination, tile *ebiten.Image, boardX, boardY int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(
		float64(boardOriginX+boardX*boardTileW),
		float64(boardOriginY+boardY*boardTileH),
	)
	destination.DrawImage(tile, op)
}
