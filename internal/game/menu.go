package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	assets "github.com/olivierh59500/match-it/internal/assets"
	"os"
	"strings"
)

func (g *Game) updateMenu() error {
	// Mouse click regions for simple menu
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !g.mouseLatch {
		g.mouseLatch = true
		x, y := ebiten.CursorPosition()
		// Scale back from 2x
		mx, my := x/2, y/2
		// Define buttons by plate rows: y bands at 37, 74, 111, 148 (height ~16)
		if inRect(mx, my, 0, 37, 320, 16) { // Start
			g.state = "play"
			g.startGame()
		} else if inRect(mx, my, 0, 74, 320, 16) { // Highscores
			g.state = "highscores"
		} else if inRect(mx, my, 0, 111, 320, 16) { // Instructions
			g.state = "instructions"
		} else if inRect(mx, my, 0, 148, 320, 16) { // Quit
			os.Exit(0)
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.mouseLatch = false
	}
	return nil
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	// Fill background with menu green (palette entry 0x020)
	green := assets.STWordToRGBA(0x0020)
	screen.Fill(green)
	// Draw menu plates (replicate original placements at lines 37, 74, 111, 148)
	if g.atlas != nil && g.atlas.MenuPlate != nil {
		ys := []int{37, 74, 111, 148}
		for _, yy := range ys {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(0, float64(yy*2))
			screen.DrawImage(ebiten.NewImageFromImage(g.atlas.MenuPlate), op)
		}
	}
	// Title at top center
	g.drawText(screen, "MATCH IT", (320-16*8)/2, 10)
	// Centered labels on plates
	items := []string{"START GAME", "HIGHSCORES", "INSTRUCTIONS", "QUIT"}
	// Slightly raise text by 1px to center vertically on the red plates.
	labelY := []int{37 + 1, 74 + 1, 111 + 1, 148 + 1}
	for i, label := range items {
		// Center in a 320px wide line: width = len(label)*16 (include spaces)
		width := len(label) * 16
		x := (320 - width) / 2
		g.drawText(screen, strings.ToUpper(label), x, labelY[i])
	}
}

func inRect(x, y, rx, ry, rw, rh int) bool {
	return x >= rx && y >= ry && x < rx+rw && y < ry+rh
}

func drawBtn(screen *ebiten.Image, x, y, w, h int) {
	img := ebiten.NewImage(w, h)
	img.Fill(color.RGBA{60, 60, 90, 150})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(x*2), float64(y*2))
	screen.DrawImage(img, op)
}
