package game

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/olivierh59500/match-it/internal/logic"
)

// Basic mouse selection and pair removal following the original flow.

func (g *Game) handleMouse() {
    // Edge-triggered click (avoid multiple activations while button is held)
    pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
    if !pressed {
        g.mouseLatch = false
        return
    }
    if g.mouseLatch { // already handled this press
        return
    }
    g.mouseLatch = true

    x, y := ebiten.CursorPosition()
    // Translate to board coordinates using ST offsets scaled by 2 (offset 16,24; tile 16x20)
    bx := (x/2 - 16) / 16
    by := (y/2 - 24) / 20
    if bx < 0 || bx >= 18 || by < 0 || by >= 8 { return }
    vCur := g.board.Get(bx, by)
    if vCur == 0 { return }
    if !g.selActive {
        g.selActive = true
        g.selX, g.selY = bx, by
        return
    }
    // If re-click same tile: deselect
    if bx == g.selX && by == g.selY {
        g.selActive = false
        return
    }
    // Try to match and path
    vSel := g.board.Get(g.selX, g.selY)
    if ok, delay := logic.MatchInfo(vSel, vCur); ok {
        var p []byte
        if logic.FindPath(g.board.Tiles, g.selX, g.selY, bx, by, &p) {
            // Start path overlay and removal animation
            g.path = p
            g.pathFromX, g.pathFromY = g.selX, g.selY
            g.pathToX, g.pathToY = bx, by
            g.pathTimer = 15
            if g.atlas != nil && g.atlas.Tiles != nil {
                idx1 := int(vSel-1)
                idx2 := int(vCur-1)
                g.startRemoveAnim(g.selX, g.selY, bx, by, idx1, idx2)
            } else {
                // If no atlas, remove immediately
                g.board.RemovePair(g.selX, g.selY, bx, by)
            }
            if delay > 0 { g.blumHold += delay }
            g.selActive = false
            return
        }
    }
    // No path or not match -> set new selection
    g.selX, g.selY = bx, by
}
