//go:build android || ios

// Package matchitmobile exposes Match'it to native mobile launchers.
package matchitmobile

import (
	"github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/olivierh59500/match-it/internal/game"
)

var matchit = game.New()

func init() {
	mobile.SetGame(matchit)
}

// SetDataDir selects the native application's private files directory for
// persistent data such as high scores.
func SetDataDir(path string) {
	matchit.SetDataDir(path)
}

// Dummy makes the package bindable even if SetDataDir is optimized away by a
// future launcher implementation.
func Dummy() {}
