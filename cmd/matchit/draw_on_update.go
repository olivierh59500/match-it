package main

import "github.com/hajimehoshi/ebiten/v2"

// drawOnUpdateGame avoids rebuilding an identical frame when the monitor runs
// faster than Ebitengine's update rate. This wrapper is desktop-only because
// the mobile launcher binds the core game directly.
type drawOnUpdateGame struct {
	game      ebiten.Game
	needsDraw bool
}

func newDrawOnUpdateGame(game ebiten.Game) *drawOnUpdateGame {
	return &drawOnUpdateGame{game: game}
}

func (g *drawOnUpdateGame) Update() error {
	if err := g.game.Update(); err != nil {
		return err
	}
	g.needsDraw = true
	return nil
}

func (g *drawOnUpdateGame) Draw(screen *ebiten.Image) {
	if !g.needsDraw {
		return
	}
	g.game.Draw(screen)
	g.needsDraw = false
}

func (g *drawOnUpdateGame) Layout(width, height int) (int, int) {
	return g.game.Layout(width, height)
}
