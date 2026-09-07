package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/match-it/internal/logic"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

const (
	logicalWidth  = 640
	logicalHeight = 400

	boardOriginX = 32
	boardOriginY = 48
	boardTileW   = 32
	boardTileH   = 40
	boardCols    = 18
	boardRows    = 8

	playControlsY = 368
	playControlW  = logicalWidth / 4
)

type playControl int

const (
	playControlNone playControl = iota
	playControlMenu
	playControlPause
	playControlRestart
	playControlMusic
)

// captureTap normalizes touch and mouse input. Touch is checked first because
// some platforms also expose a synthetic mouse event for a finger press.
func (g *Game) captureTap() {
	g.tapAvailable = false
	g.touchIDs = inpututil.AppendJustPressedTouchIDs(g.touchIDs[:0])
	if len(g.touchIDs) > 0 {
		g.tapX, g.tapY = ebiten.TouchPosition(g.touchIDs[0])
		g.tapAvailable = true
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.tapX, g.tapY = ebiten.CursorPosition()
		g.tapAvailable = true
	}
}

func (g *Game) consumeTap() (x, y int, ok bool) {
	if !g.tapAvailable {
		return 0, 0, false
	}
	g.tapAvailable = false
	return g.tapX, g.tapY, true
}

func (g *Game) consumeAnyTap() bool {
	_, _, ok := g.consumeTap()
	return ok
}

func playControlAt(x, y int) playControl {
	if x < 0 || x >= logicalWidth || y < playControlsY || y >= logicalHeight {
		return playControlNone
	}
	switch x / playControlW {
	case 0:
		return playControlMenu
	case 1:
		return playControlPause
	case 2:
		return playControlRestart
	case 3:
		return playControlMusic
	default:
		return playControlNone
	}
}

func boardCellAt(x, y int) (boardX, boardY int, ok bool) {
	boardWidth := boardCols * boardTileW
	boardHeight := boardRows * boardTileH
	if x < boardOriginX || x >= boardOriginX+boardWidth ||
		y < boardOriginY || y >= boardOriginY+boardHeight {
		return 0, 0, false
	}
	return (x - boardOriginX) / boardTileW, (y - boardOriginY) / boardTileH, true
}

func (g *Game) updatePlayInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.paused = false
		g.state = "menu"
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.paused = false
		g.newRound()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) && !g.paused {
		g.useHelp()
	}

	x, y, ok := g.consumeTap()
	if !ok {
		return
	}
	switch playControlAt(x, y) {
	case playControlMenu:
		g.paused = false
		g.state = "menu"
		return
	case playControlPause:
		g.paused = !g.paused
		return
	case playControlRestart:
		g.paused = false
		g.newRound()
		return
	case playControlMusic:
		g.toggleMusic()
		return
	}

	// The original help plate doubles as a large touch target.
	if x >= 512 && x < logicalWidth && y >= 0 && y < boardOriginY {
		if !g.paused {
			g.useHelp()
		}
		return
	}
	if !g.paused {
		g.handleBoardTap(x, y)
	}
}

func (g *Game) useHelp() {
	if g.helpCount <= 0 || g.anim != nil {
		return
	}
	x1, y1, x2, y2, path, ok := g.board.HelpSearch()
	if !ok {
		return
	}
	g.path = path
	g.pathFromX, g.pathFromY = x1, y1
	g.pathToX, g.pathToY = x2, y2
	g.pathTimer = 15
	g.helpCount--
	g.helpUsed = true
	if g.atlas != nil && g.atlas.Tiles != nil {
		idx1 := int(g.board.Get(x1, y1) - 1)
		idx2 := int(g.board.Get(x2, y2) - 1)
		g.startRemoveAnim(x1, y1, x2, y2, idx1, idx2)
	}
}

// handleBoardTap performs tile selection and pair removal for a logical screen
// coordinate. It is shared by touchscreens and the desktop mouse.
func (g *Game) handleBoardTap(x, y int) {
	if g.anim != nil {
		return
	}
	bx, by, ok := boardCellAt(x, y)
	if !ok {
		return
	}
	vCur := g.board.Get(bx, by)
	if vCur == 0 {
		return
	}
	if !g.selActive {
		g.selActive = true
		g.selX, g.selY = bx, by
		return
	}
	if bx == g.selX && by == g.selY {
		g.selActive = false
		return
	}

	vSel := g.board.Get(g.selX, g.selY)
	if matches, delay := logic.MatchInfo(vSel, vCur); matches {
		var path []byte
		if logic.FindPath(g.board.Tiles, g.selX, g.selY, bx, by, &path) {
			g.path = path
			g.pathFromX, g.pathFromY = g.selX, g.selY
			g.pathToX, g.pathToY = bx, by
			g.pathTimer = 15
			if g.atlas != nil && g.atlas.Tiles != nil {
				g.startRemoveAnim(g.selX, g.selY, bx, by, int(vSel-1), int(vCur-1))
			} else {
				g.board.RemovePair(g.selX, g.selY, bx, by)
				g.score++
			}
			if delay > 0 {
				g.blumHold += delay
			}
			g.selActive = false
			return
		}
	}
	// A non-matching tile becomes the new selection.
	g.selX, g.selY = bx, by
}

func (g *Game) drawPlayControls(screen *ebiten.Image) {
	if g.paused {
		vector.DrawFilledRect(screen, 0, boardOriginY, logicalWidth, playControlsY-boardOriginY, color.RGBA{0, 0, 0, 170}, false)
		g.drawCenteredSystemText(screen, "PAUSED", 210, color.White)
	}

	vector.DrawFilledRect(screen, 0, playControlsY, logicalWidth, logicalHeight-playControlsY, color.RGBA{20, 20, 35, 235}, false)
	labels := []string{"MENU", "PAUSE", "RESTART", "MUSIC"}
	if g.paused {
		labels[1] = "RESUME"
	}
	if !g.musicOn {
		labels[3] = "MUSIC OFF"
	}
	for i, label := range labels {
		if i > 0 {
			x := float32(i * playControlW)
			vector.DrawFilledRect(screen, x, playControlsY+4, 1, logicalHeight-playControlsY-8, color.RGBA{180, 180, 200, 180}, false)
		}
		g.drawSystemTextCenteredInRect(screen, label, i*playControlW, playControlsY, playControlW, logicalHeight-playControlsY, color.White)
	}
}

// drawTileSelection draws a high-contrast, pulsing frame and an order badge.
// The optional tint is used for the first selection only; removal frames stay
// transparent so their dissolve animation remains visible.
func (g *Game) drawTileSelection(screen *ebiten.Image, boardX, boardY int, label string, tint bool) {
	x := float32(boardOriginX + boardX*boardTileW)
	y := float32(boardOriginY + boardY*boardTileH)
	w := float32(boardTileW)
	h := float32(boardTileH)

	phase := g.frames % 30
	if phase > 15 {
		phase = 30 - phase
	}
	gold := color.RGBA{255, uint8(170 + phase*5), 0, 255}
	if tint {
		vector.DrawFilledRect(screen, x+4, y+4, w-8, h-8, color.RGBA{255, 210, 0, uint8(35 + phase)}, false)
	}
	drawFrame(screen, x, y, w, h, 4, color.RGBA{0, 0, 0, 230})
	drawFrame(screen, x+2, y+2, w-4, h-4, 2, gold)

	// A numbered badge makes the selection order explicit without depending on
	// color alone.
	vector.DrawFilledRect(screen, x+3, y+3, 15, 18, gold, false)
	vector.DrawFilledRect(screen, x+5, y+5, 11, 14, color.RGBA{15, 15, 20, 255}, false)
	text.Draw(screen, label, basicfont.Face7x13, int(x)+7, int(y)+17, color.White)
}

func drawFrame(screen *ebiten.Image, x, y, width, height, thickness float32, clr color.Color) {
	vector.DrawFilledRect(screen, x, y, width, thickness, clr, false)
	vector.DrawFilledRect(screen, x, y+height-thickness, width, thickness, clr, false)
	vector.DrawFilledRect(screen, x, y+thickness, thickness, height-2*thickness, clr, false)
	vector.DrawFilledRect(screen, x+width-thickness, y+thickness, thickness, height-2*thickness, clr, false)
}

func (g *Game) drawCenteredSystemText(screen *ebiten.Image, label string, baselineY int, clr color.Color) {
	face := g.instrFace
	if face == nil {
		face = basicfont.Face7x13
	}
	w := font.MeasureString(face, label).Round()
	text.Draw(screen, label, face, (logicalWidth-w)/2, baselineY, clr)
}

func (g *Game) drawSystemTextCenteredInRect(screen *ebiten.Image, label string, x, y, w, h int, clr color.Color) {
	face := g.instrFace
	if face == nil {
		face = basicfont.Face7x13
	}
	metrics := face.Metrics()
	tw := font.MeasureString(face, label).Round()
	th := metrics.Height.Round()
	baseline := y + (h-th)/2 + metrics.Ascent.Round()
	text.Draw(screen, label, face, x+(w-tw)/2, baseline, clr)
}
