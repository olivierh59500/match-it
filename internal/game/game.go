package game

import (
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	assets "github.com/olivierh59500/match-it/internal/assets"
	audiox "github.com/olivierh59500/match-it/internal/audio"
	"github.com/olivierh59500/match-it/internal/logic"
)

// Game implements ebiten.Game. It reproduces the original flow:
// menu -> gameplay (single player) -> highscores/instructions (later). For now, we focus on gameplay.
type Game struct {
	levels *logic.LevelSet
	board  logic.Board

	// State
	helpCount int
	score     int
	timeLeft  int // game time units (start 200)
	timeDelay int // timeverzoegerung; frames per decrement (start 70, min 20)
	vblCount  int // frames since last decrement
	blumHold  int // pause count for special tiles (counts decrements to skip)

	// Path overlay
	path                 []byte
	pathFromX, pathFromY int
	pathToX, pathToY     int
	pathTimer            int // frames to display the overlay

	// Art assets (optional)
	atlas *atlas

	// simple timers
	frames   int
	helpUsed bool

	// removal animation state
	anim        *removeAnim
	removeMasks [][20]uint16
	mouseLatch  bool
	helpLatch   bool

	selOverlay *ebiten.Image
	selActive  bool
	selX, selY int

	// UI state
	state   string // "menu", "play", "highscores", "instructions", "entername"
	hs      *Highscores
	nameBuf string

	// Music
	audioCtx    *audio.Context
	audioPlayer *audio.Player
	ym          *audiox.YMPlayer

	// Controls
	paused  bool
	musicOn bool
	prevVol float64

	// Level/summary
	stage            int
	pendingTimeBonus int
	pendingHelpBonus int

	// Fonts
	instrFace font.Face
}

func New() *Game {
	ls, err := logic.LoadLevels(filepath.Join("assets", "png", "gamearea.img.png"))
	if err != nil {
		log.Printf("levels: %v", err)
	}
	hs, _ := loadHighscores("highscores.json")
	g := &Game{levels: ls, hs: hs, state: "menu"}
	g.newRound()
	g.tryLoadAtlas()
	g.initMusic()
	g.initFonts()
	g.musicOn = true
	return g
}

func (g *Game) newRound() {
	if g.levels == nil {
		return
	}
	tiles, pos := g.levels.NextBoard()
	g.board.FromLevel(tiles, pos)
	// Do not reset helpCount or score here; only at game start (menu -> play)
	g.timeLeft = 200
	if g.timeDelay == 0 {
		g.timeDelay = 70
	}
	g.vblCount = 0
	g.blumHold = 0
	g.path = nil
	g.pathTimer = 0
	g.frames = 0
	g.helpUsed = false
	g.selActive = false
}

// startGame resets global game stats and begins a new board.
func (g *Game) startGame() {
	g.score = 0
	g.helpCount = 2 // original starts at 2
	g.timeDelay = 70
	g.helpUsed = false
	g.paused = false
	g.stage = 1
	g.newRound()
}

func (g *Game) onGameOver() {
	// Submit highscore if qualifies; otherwise go to menu/highscores
	// Qualification: fewer than 10 entries or score > last
	qualifies := false
	if g.hs == nil || len(g.hs.Entries) < 10 {
		qualifies = true
	} else if g.score > g.hs.Entries[len(g.hs.Entries)-1].Score {
		qualifies = true
	}
	if qualifies {
		g.state = "entername"
		g.nameBuf = ""
	} else {
		g.state = "highscores"
	}
}

func (g *Game) initMusic() {
	if g.audioCtx != nil {
		return
	}
	g.audioCtx = audio.NewContext(44100)
	data, err := os.ReadFile("assets/music/Chambers of Shaolin - Trapped in China.ym")
	if err != nil {
		log.Printf("music load: %v", err)
		return
	}
	ym, err := audiox.NewYMPlayer(data, 44100, true)
	if err != nil {
		log.Printf("ym init: %v", err)
		return
	}
	g.ym = ym
	p, err := g.audioCtx.NewPlayer(ym)
	if err != nil {
		log.Printf("audio player: %v", err)
		_ = ym.Close()
		g.ym = nil
		return
	}
	g.audioPlayer = p
	g.audioPlayer.Play()
}

func (g *Game) Update() error {
	// Global input (applies to all states)
	g.handleGlobalInput()
	// State machine
	switch g.state {
	case "menu":
		return g.updateMenu()
	case "highscores":
		return g.updateHighscores()
	case "instructions":
		return g.updateInstructions()
	case "entername":
		return g.updateEnterName()
	case "levelsummary":
		return g.updateLevelSummary()
	}
	// Keyboard controls for quick testing in game: H for help, R to restart.
	g.frames++
	// Timer: decrement time per timeDelay frames, pause for blumHold counts (and paused state)
	if g.state == "play" && !g.paused && g.timeLeft > 0 {
		g.vblCount++
		if g.vblCount >= g.timeDelay {
			g.vblCount = 0
			if g.blumHold > 0 {
				g.blumHold--
			} else {
				g.timeLeft--
				if g.timeLeft <= 0 {
					g.onGameOver()
				}
			}
		}
	}
	if g.state == "play" && ebiten.IsKeyPressed(ebiten.KeyR) {
		g.newRound()
	}
	if g.state == "play" && !g.paused && ebiten.IsKeyPressed(ebiten.KeyH) && !g.helpLatch {
		g.helpLatch = true
		if g.helpCount > 0 {
			x1, y1, x2, y2, p, ok := g.board.HelpSearch()
			if ok {
				g.path = p
				g.pathFromX, g.pathFromY = x1, y1
				g.pathToX, g.pathToY = x2, y2
				g.pathTimer = 15
				g.helpCount--
				g.helpUsed = true
				// start removal animation preview
				if g.atlas != nil && g.atlas.Tiles != nil {
					idx1 := int(g.board.Get(x1, y1) - 1)
					idx2 := int(g.board.Get(x2, y2) - 1)
					g.startRemoveAnim(x1, y1, x2, y2, idx1, idx2)
				}
			}
		}
	}
	if !ebiten.IsKeyPressed(ebiten.KeyH) {
		g.helpLatch = false
	}
	// ESC to menu during play
	if g.state == "play" && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = "menu"
	}
	// Pause toggle (P) during play
	if g.state == "play" && inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
	}
	// Music toggle handled globally
	// Mouse input
	if g.state == "play" && !g.paused {
		g.handleMouse()
	}

	// Animate removal if active
	if g.state == "play" && !g.paused && g.anim != nil {
		g.anim.frame++
		if g.anim.frame >= g.anim.frames {
			// Commit removal to board and end anim
			g.board.RemovePair(g.anim.x1, g.anim.y1, g.anim.x2, g.anim.y2)
			// Increment score per removed pair (matches original addq.w #1,score)
			g.score++
			g.anim = nil
		}
	}
	// Advance to level summary if cleared
	if g.state == "play" && g.isCleared() {
		g.pendingTimeBonus = g.timeLeft
		g.pendingHelpBonus = g.helpCount * 100
		g.state = "levelsummary"
		return nil
	}
	if g.pathTimer > 0 {
		g.pathTimer--
	}
	// Checkmate: no more possible pairs -> game over
	if g.state == "play" && !g.paused {
		if _, _, _, _, _, ok := g.board.HelpSearch(); !ok {
			g.onGameOver()
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw according to state
	switch g.state {
	case "menu":
		g.drawMenu(screen)
		return
	case "highscores":
		g.drawHighscores(screen)
		return
	case "instructions":
		g.drawInstructions(screen)
		return
	case "entername":
		g.drawEnterName(screen)
		return
	case "levelsummary":
		g.drawLevelSummary(screen)
		return
	}
	// Temporary: blank background if assets missing.
	screen.Fill(color.RGBA{0, 0, 32, 255})
	if g.atlas != nil && g.atlas.Tiles != nil {
		// Draw background plates if available (scaled 2x)
		if g.atlas.BGTop != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			screen.DrawImage(ebiten.NewImageFromImage(g.atlas.BGTop), op)
		}
		if g.atlas.BGArea != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(0, float64(17*2))
			screen.DrawImage(ebiten.NewImageFromImage(g.atlas.BGArea), op)
		}
		// Render tiles using atlas; ST offsets: x=16 + 16*x, y=24 + 20*y (scaled 2x)
		const originX, originY = 16, 24
		const tileW, tileH = 16, 20
		for y := 0; y < 8; y++ {
			for x := 0; x < 18; x++ {
				v := g.board.Get(x, y)
				if v == 0 {
					continue
				}
				// IDs in board are 1..42 => atlas index v-1
				idx := int(v - 1)
				if idx >= 0 && idx < len(g.atlas.Tiles) {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(2, 2)
					op.GeoM.Translate(float64((originX+x*tileW)*2), float64((originY+y*tileH)*2))
					// If animating and this cell is part of anim, draw masked frame instead of full tile
					if g.anim != nil && ((x == g.anim.x1 && y == g.anim.y1) || (x == g.anim.x2 && y == g.anim.y2)) {
						var img *ebiten.Image
						if x == g.anim.x1 && y == g.anim.y1 {
							img = g.anim.img1[g.anim.frame]
						} else {
							img = g.anim.img2[g.anim.frame]
						}
						screen.DrawImage(img, op)
					} else {
						screen.DrawImage(ebiten.NewImageFromImage(g.atlas.Tiles[idx]), op)
					}
					// Selection highlight overlay: thin border, semi-transparent
					if g.selActive && x == g.selX && y == g.selY {
						if g.selOverlay == nil {
							g.selOverlay = makeSelectionOverlay(tileW, tileH, color.RGBA{255, 255, 0, 96})
						}
						screen.DrawImage(g.selOverlay, op)
					}
				}
			}
		}
		// Path overlay using last 6 sprites (indices 43..48)
		if g.pathTimer > 0 && len(g.path) > 0 && len(g.atlas.Tiles) >= 49 {
			px, py := g.pathFromX, g.pathFromY
			prev := 0
			for i := 0; i < len(g.path); i++ {
				dir := int(g.path[i])
				// move into next cell
				switch dir {
				case 1:
					px++
				case 2:
					px--
				case 3:
					py++
				case 4:
					py--
				}
				next := 0
				if i+1 < len(g.path) {
					next = int(g.path[i+1])
				}
				glyph := pathGlyphIndex(prev, dir, next)
				idx := 43 + glyph
				if idx >= 43 && idx < 49 {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(2, 2)
					op.GeoM.Translate(float64((originX+px*tileW)*2), float64((originY+py*tileH)*2))
					screen.DrawImage(ebiten.NewImageFromImage(g.atlas.Tiles[idx]), op)
				}
				prev = dir
			}
		}
		// HUD: draw score, time, helps using FONT2 digits and help plates
		g.drawHUD(screen)
	}
	// No debug text; HUD to be implemented via original assets
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// Use a fixed logical size so Ebiten scales the backbuffer to the window size.
	// This makes the whole game zoom with window resizing.
	return 640, 400
}

// itoa2 prints a small 2-digit tile id.
func itoa2(b byte) string {
	if b < 10 {
		return "0" + string('0'+b)
	}
	return string([]byte{'0' + (b / 10), '0' + (b % 10)})
}

// pathGlyphIndex maps (prev, curr, next) directions to glyph index 0..5 located at tile indices 43..48.
// Encoding per original:
// 0: horizontal, 1: vertical, 2..5: corners (see draw_weg logic)
func pathGlyphIndex(prev, curr, next int) int {
	switch curr {
	case 1: // right
		switch next {
		case 3:
			return 4
		case 4:
			return 3
		}
		return 0
	case 2: // left
		switch next {
		case 3:
			return 5
		case 4:
			return 2
		}
		return 0
	case 3: // down
		switch next {
		case 1:
			return 2
		case 2:
			return 3
		}
		return 1
	case 4: // up
		switch next {
		case 1:
			return 5
		case 2:
			return 4
		}
		return 1
	}
	return 0
}

func (g *Game) isCleared() bool {
	for _, v := range g.board.Tiles {
		if v != 0 {
			return false
		}
	}
	return true
}

// drawHUD renders score (X=3,Y=1), time (X=10,Y=1), and help plate at X=256px,Y=0 using scaled coordinates.
func (g *Game) drawHUD(screen *ebiten.Image) {
	if g.atlas == nil {
		return
	}
	// Digits scale and placement match ST: each digit 16x15; positions in pixels, then scaled 2x.
	drawNum := func(val, x16, y int, max int) {
		if g.atlas.Digits[0] == nil {
			return
		}
		s := itoaDec(val)
		if len(s) > max {
			s = s[len(s)-max:]
		}
		op := &ebiten.DrawImageOptions{}
		for i := 0; i < len(s); i++ {
			d := s[i] - '0'
			if d < 0 || d > 9 {
				continue
			}
			op.GeoM.Reset()
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(float64(x16*2+i*16*2), float64(y*2))
			if g.state == "play" { // show HUD only during game
				screen.DrawImage(ebiten.NewImageFromImage(g.atlas.Digits[int(d)]), op)
			}
		}
	}
	// Score at X=3*16=48, Y=1
	drawNum(g.score, 3*16, 1, 5)
	// Time at X=10*16=160, Y=1 (3 digits)
	if g.timeLeft < 0 {
		g.timeLeft = 0
	}
	drawNum(g.timeLeft, 10*16, 1, 3)
	// Help plate at X=256, Y=0
	if len(g.atlas.HelpPlates) == 6 && g.state == "play" {
		idx := g.helpCount
		if idx < 0 {
			idx = 0
		}
		if idx > 5 {
			idx = 5
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(256*2), 0)
		screen.DrawImage(ebiten.NewImageFromImage(g.atlas.HelpPlates[idx]), op)
	}
}

// initFonts loads a readable system-safe font for instructions and other UI text.
func (g *Game) initFonts() {
	ft, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Printf("font parse: %v", err)
		return
	}
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Printf("font face: %v", err)
		return
	}
	g.instrFace = face
}

// itoaDec converts int to decimal string without leading zeros.
func itoaDec(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	b := make([]byte, 0, 10)
	for n > 0 {
		b = append(b, byte('0'+n%10))
		n /= 10
	}
	// reverse
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// drawText draws uppercase text with the 16x15 font at pixel coordinates (x,y), scaled 2x.
func (g *Game) drawText(screen *ebiten.Image, s string, x, y int) {
	if g.atlas == nil || len(g.atlas.Font47) == 0 {
		return
	}
	xx := x
	for _, r := range s {
		if r == ' ' {
			xx += 16
			continue
		}
		idx := assets.FontIndexForChar(r)
		if idx >= 0 && idx < len(g.atlas.Font47) {
			img := g.atlas.Font47[idx]
			if img != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(2, 2)
				op.GeoM.Translate(float64(xx*2), float64(y*2))
				screen.DrawImage(ebiten.NewImageFromImage(img), op)
			}
		}
		xx += 16
	}
}

// drawSmallText draws text using the menu font at 1x scale (not 2x), to approximate the small instruction font.
func (g *Game) drawSmallText(screen *ebiten.Image, s string, x, y int, spacing int) {
	if g.atlas == nil || len(g.atlas.Font47) == 0 {
		return
	}
	xx := x
	for _, r := range s {
		if r == ' ' {
			xx += 8 + spacing
			continue
		}
		idx := assets.FontIndexForChar(r)
		if idx >= 0 && idx < len(g.atlas.Font47) {
			img := g.atlas.Font47[idx]
			if img != nil {
				op := &ebiten.DrawImageOptions{}
				// scale down to ~8px width by scaling 0.5
				op.GeoM.Scale(0.5, 0.5)
				op.GeoM.Translate(float64(xx), float64(y))
				screen.DrawImage(ebiten.NewImageFromImage(img), op)
			}
		}
		xx += 8 + spacing
	}
}

// smallTextWidth returns the pixel width of the small text with given spacing.
func (g *Game) smallTextWidth(s string, spacing int) int {
	w := 0
	for _, r := range s {
		if r == ' ' {
			w += 8 + spacing
		} else if assets.FontIndexForChar(r) >= 0 {
			w += 8 + spacing
		}
	}
	if w > 0 {
		w -= spacing
	}
	return w
}

// makeSelectionOverlay builds a thin border rectangle image.
func makeSelectionOverlay(w, h int, c color.RGBA) *ebiten.Image {
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	// 1px border
	for x := 0; x < w; x++ {
		rgba.SetRGBA(x, 0, c)
		rgba.SetRGBA(x, h-1, c)
	}
	for y := 0; y < h; y++ {
		rgba.SetRGBA(0, y, c)
		rgba.SetRGBA(w-1, y, c)
	}
	return ebiten.NewImageFromImage(rgba)
}

// handleGlobalInput processes inputs that should affect all states (e.g., music toggle).
func (g *Game) handleGlobalInput() {
	// Music toggle (M/m): mute/unmute by volume across all screens
	toggle := inpututil.IsKeyJustPressed(ebiten.KeyM)
	for _, r := range ebiten.InputChars() {
		if r == 'm' || r == 'M' {
			toggle = true
			break
		}
	}
	if toggle && g.ym != nil {
		if g.musicOn {
			g.prevVol = g.ym.GetVolume()
			g.ym.SetVolume(0)
			g.musicOn = false
		} else {
			v := g.prevVol
			if v <= 0 {
				v = 0.7
			}
			g.ym.SetVolume(v)
			g.musicOn = true
		}
	}
}

// drawChars8Centered draws a line of text using the original chars8 font at scale and spacing, centered at given y.
func (g *Game) drawChars8Centered(screen *ebiten.Image, s string, y int, spacing int, scale int) {
	img, w, _ := g.renderChars8Line(s, spacing, scale)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((640-w)/2), float64(y))
	screen.DrawImage(img, op)
}

// renderChars8Line renders a single line of chars8 text into an offscreen image and returns it with width/height.
func (g *Game) renderChars8Line(s string, spacing, scale int) (*ebiten.Image, int, int) {
	if g.atlas == nil || g.atlas.CharSheet == nil {
		return nil, 0, 0
	}
	// measure width
	w := 0
	for range s {
		w += (8 + spacing) * scale
	}
	if len(s) > 0 {
		w -= spacing * scale
	}
	if w <= 0 {
		w = 1
	}
	h := 8 * scale
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	x := 0
	for _, r := range s {
		if r == ' ' {
			x += (8 + spacing) * scale
			continue
		}
		// chars8 encodes 64 glyphs laid out in ASCII order starting at space (0x20)
		c := int(r) - 32 // map ASCII to glyph index (space=0)
		if c < 0 || c >= 64 {
			x += (8 + spacing) * scale
			continue
		}
		col := c & 31
		blk := (c >> 5) & 0x7
		x0 := col * 8
		y0 := blk * 8
		for row := 0; row < 8; row++ {
			yy := y0 + row
			if yy < 0 || yy >= g.atlas.CharSheet.Bounds().Dy() {
				continue
			}
			for bit := 0; bit < 8; bit++ {
				xx := x0 + bit
				if xx < 0 || xx >= g.atlas.CharSheet.Bounds().Dx() {
					continue
				}
				if g.atlas.CharSheet.RGBAAt(xx, yy).A > 0 {
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							rgba.Set(x+bit*scale+dx, row*scale+dy, color.White)
						}
					}
				}
			}
		}
		x += (8 + spacing) * scale
	}
	return ebiten.NewImageFromImage(rgba), w, h
}
