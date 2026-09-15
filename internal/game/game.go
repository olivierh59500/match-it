package game

import (
	"bytes"
	"image/color"
	"log"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	resources "github.com/olivierh59500/match-it/assets"
	assets "github.com/olivierh59500/match-it/internal/assets"
	audiox "github.com/olivierh59500/match-it/internal/audio"
	"github.com/olivierh59500/match-it/internal/logic"
	"golang.org/x/image/font/gofont/goregular"
)

const audioSampleRate = 48000

type cachedMove struct {
	x1, y1 int
	x2, y2 int
	path   []byte
	ok     bool
}

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
	pathTimer            int // frames to display the overlay
	availableMove        cachedMove
	remainingTiles       int

	// Art assets (optional)
	atlas            *atlas
	boardCanvas      *ebiten.Image
	boardCanvasDirty bool

	// simple timers
	frames   int
	helpUsed bool

	// removal animation state
	anim *removeAnim

	selActive  bool
	selX, selY int

	// UI state
	state   string // "menu", "play", "highscores", "instructions", "entername"
	hs      *Highscores
	nameBuf string
	hsPath  string

	// Pointer input. Touches and mouse clicks are normalized to one logical tap
	// per update so every screen follows the same interaction path.
	tapAvailable bool
	tapX, tapY   int
	touchIDs     []ebiten.TouchID
	inputChars   []rune

	// Music
	audioCtx    *audio.Context
	audioPlayer *audio.Player
	ym          *audiox.YMPlayer

	// Controls
	paused  bool
	musicOn bool

	// Level/summary
	stage            int
	pendingTimeBonus int
	pendingHelpBonus int

	// Fonts
	instrFace         text.Face
	instructionLines  []string
	instructionWidths []int

	// Splash
	splash      *ebiten.Image
	splashStart time.Time
}

func New() *Game {
	levelFile, err := resources.Files.Open("png/gamearea.img.png")
	var ls *logic.LevelSet
	if err == nil {
		ls, err = logic.DecodeLevels(levelFile)
		_ = levelFile.Close()
	}
	if err != nil {
		log.Printf("levels: %v", err)
	}
	const hsPath = "highscores.json"
	hs, _ := loadHighscores(hsPath)
	g := &Game{levels: ls, hs: hs, hsPath: hsPath, state: "splash"}
	g.newRound()
	g.tryLoadAtlas()
	g.initFonts()
	g.musicOn = true
	// Load splash image if available
	if img, err := loadPNG("png/malakhsoftware-pixel.png"); err == nil {
		g.splash = ebiten.NewImageFromImage(img)
	}
	g.splashStart = time.Now()
	return g
}

// SetDataDir switches persistent data to an application-owned directory. The
// Android launcher calls this before the first frame because a bound Go
// library does not have a useful project working directory.
func (g *Game) SetDataDir(dir string) {
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "highscores.json")
	hs, err := loadHighscores(path)
	if err != nil {
		log.Printf("highscores: %v", err)
		return
	}
	g.hsPath = path
	g.hs = hs
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
	g.anim = nil
	g.boardCanvasDirty = true
	g.remainingTiles = 0
	for _, tile := range g.board.Tiles {
		if tile != 0 {
			g.remainingTiles++
		}
	}
	g.refreshAvailableMove()
}

func (g *Game) refreshAvailableMove() {
	if g.remainingTiles == 0 {
		g.availableMove = cachedMove{}
		return
	}
	x1, y1, x2, y2, path, ok := g.board.HelpSearch()
	g.availableMove = cachedMove{x1: x1, y1: y1, x2: x2, y2: y2, path: path, ok: ok}
}

func (g *Game) commitPair(x1, y1, x2, y2 int) {
	if g.board.Get(x1, y1) == 0 || g.board.Get(x2, y2) == 0 {
		g.anim = nil
		return
	}
	g.board.RemovePair(x1, y1, x2, y2)
	g.remainingTiles -= 2
	if g.remainingTiles < 0 {
		g.remainingTiles = 0
	}
	g.score++
	g.anim = nil
	g.boardCanvasDirty = true
	g.refreshAvailableMove()
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
	g.audioCtx = audio.NewContext(audioSampleRate)
	data, err := resources.Files.ReadFile("music/Chambers of Shaolin - Trapped in China.ym")
	if err != nil {
		log.Printf("music load: %v", err)
		return
	}
	ym, err := audiox.NewYMPlayer(data, audioSampleRate, true)
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
	g.captureTap()
	g.inputChars = ebiten.AppendInputChars(g.inputChars[:0])
	// Global input (applies to all states)
	g.handleGlobalInput()
	// State machine
	switch g.state {
	case "splash":
		if time.Since(g.splashStart) >= 3*time.Second || g.consumeAnyTap() {
			g.state = "menu"
			g.initMusic()
		}
		return nil
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
	if g.state == "play" {
		g.updatePlayInput()
	}

	// Animate removal if active
	if g.state == "play" && !g.paused && g.anim != nil {
		g.anim.frame++
		if g.anim.frame >= removeFrameCount {
			// Commit removal to board and end anim
			g.commitPair(g.anim.x1, g.anim.y1, g.anim.x2, g.anim.y2)
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
	if g.state == "play" && !g.paused && g.anim == nil && g.remainingTiles > 0 && !g.availableMove.ok {
		g.onGameOver()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw according to state
	switch g.state {
	case "splash":
		screen.Fill(color.White)
		if g.splash != nil {
			b := g.splash.Bounds()
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64((640-b.Dx())/2), float64((400-b.Dy())/2))
			screen.DrawImage(g.splash, op)
		}
		return
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
	if g.atlas != nil && g.atlas.Tiles != nil {
		g.drawBoardBase(screen)
		if g.anim != nil {
			drawTileImage(screen, g.atlas.RemovalFrames[g.anim.tile1][g.anim.frame], g.anim.x1, g.anim.y1)
			drawTileImage(screen, g.atlas.RemovalFrames[g.anim.tile2][g.anim.frame], g.anim.x2, g.anim.y2)
		}
		// Path overlay using last 6 sprites (indices 43..48)
		if g.pathTimer > 0 && len(g.path) > 0 && len(g.atlas.Tiles) >= 49 {
			px, py := g.pathFromX, g.pathFromY
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
				glyph := pathGlyphIndex(dir, next)
				idx := 43 + glyph
				if idx >= 43 && idx < 49 && g.atlas.Tiles[idx] != nil {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(2, 2)
					op.GeoM.Translate(float64(boardOriginX+px*boardTileW), float64(boardOriginY+py*boardTileH))
					screen.DrawImage(g.atlas.Tiles[idx], op)
				}
			}
		}
		// Keep selected endpoints above both the tiles and path so selection is
		// unmistakable on small, bright mobile screens.
		if g.anim != nil {
			g.drawTileSelection(screen, g.anim.x1, g.anim.y1, "1", false)
			g.drawTileSelection(screen, g.anim.x2, g.anim.y2, "2", false)
		} else if g.selActive {
			g.drawTileSelection(screen, g.selX, g.selY, "1", true)
		}
		// HUD: draw score, time, helps using FONT2 digits and help plates
		g.drawHUD(screen)
		g.drawPlayControls(screen)
		return
	}
	screen.Fill(color.RGBA{0, 0, 32, 255})
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// Use a fixed logical size so Ebiten scales the backbuffer to the window size.
	// This makes the whole game zoom with window resizing.
	return logicalWidth, logicalHeight
}

// itoa2 prints a small 2-digit tile id.
func itoa2(b byte) string {
	if b < 10 {
		return "0" + string('0'+b)
	}
	return string([]byte{'0' + (b / 10), '0' + (b % 10)})
}

// pathGlyphIndex maps current and next directions to glyph index 0..5 located at tile indices 43..48.
// Encoding per original:
// 0: horizontal, 1: vertical, 2..5: corners (see draw_weg logic)
func pathGlyphIndex(curr, next int) int {
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
	return g.remainingTiles == 0
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
			if d > 9 {
				continue
			}
			op.GeoM.Reset()
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(float64(x16*2+i*16*2), float64(y*2))
			if g.state == "play" { // show HUD only during game
				screen.DrawImage(g.atlas.Digits[int(d)], op)
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
		screen.DrawImage(g.atlas.HelpPlates[idx], op)
	}
}

// initFonts loads a readable system-safe font for instructions and other UI text.
func (g *Game) initFonts() {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Printf("font source: %v", err)
		return
	}
	face := &text.GoTextFace{Source: source, Size: 16}
	g.instrFace = face
	g.instructionLines = wrapFace(instructionParagraphs, face, 600)
	g.instructionWidths = make([]int, len(g.instructionLines))
	for i, line := range g.instructionLines {
		g.instructionWidths[i] = measureFace(face, line)
	}
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
				screen.DrawImage(img, op)
			}
		}
		xx += 16
	}
}

// handleGlobalInput processes inputs that should affect all states (e.g., music toggle).
func (g *Game) handleGlobalInput() {
	// Music toggle (M/m): mute/unmute by volume across all screens
	toggle := inpututil.IsKeyJustPressed(ebiten.KeyM)
	for _, r := range g.inputChars {
		if r == 'm' || r == 'M' {
			toggle = true
			break
		}
	}
	if toggle {
		g.toggleMusic()
	}
}

func (g *Game) toggleMusic() {
	if g.audioPlayer == nil {
		return
	}
	if g.musicOn {
		g.audioPlayer.SetVolume(0)
		g.musicOn = false
		return
	}
	g.audioPlayer.SetVolume(1)
	g.musicOn = true
}
