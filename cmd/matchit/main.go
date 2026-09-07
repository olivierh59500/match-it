package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/match-it/internal/game"
)

// Entry point: sets up the Ebiten window and runs the game loop.
func main() {
	g := game.New()

	ebiten.SetWindowTitle("Match'it (Go + Ebiten)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(640, 400) // 2x ST resolution for clarity

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
