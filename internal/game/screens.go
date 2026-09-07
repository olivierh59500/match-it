package game

import (
	"image"
	"image/color"
	"log"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	assets "github.com/olivierh59500/match-it/internal/assets"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

func (g *Game) updateHighscores() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || g.consumeAnyTap() {
		g.state = "menu"
	}
	return nil
}

// Level summary (between stages)
func (g *Game) updateLevelSummary() error {
	// Any tap or Enter continues to the next stage.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || g.consumeAnyTap() {
		// Apply bonuses
		g.score += g.pendingTimeBonus
		g.score += g.pendingHelpBonus
		// Add help if none used (cap 5)
		if !g.helpUsed && g.helpCount < 5 {
			g.helpCount++
		}
		// Speed up timer
		if g.timeDelay > 20 {
			g.timeDelay -= 2
		}
		// Next stage
		g.stage++
		g.newRound()
		g.state = "play"
	}
	return nil
}

func (g *Game) drawHighscores(screen *ebiten.Image) {
	// Fill with the same menu green as main menu
	screen.Fill(assets.STWordToRGBA(0x0020))
	// Title
	title := "HIGHSCORES"
	g.drawText(screen, title, (320-len(title)*16)/2, 20)
	// Render top 10 names and scores
	if g.atlas == nil {
		return
	}
	y := 40
	for i := 0; i < len(g.hs.Entries) && i < 10; i++ {
		e := g.hs.Entries[i]
		// Name left, score right
		g.drawText(screen, strings.ToUpper(e.Name), 32, y)
		drawDigits(screen, g.atlas.Digits, e.Score, 220, y, 6)
		y += 18
	}
	if len(g.hs.Entries) == 0 {
		msg := "NO ENTRIES"
		g.drawText(screen, msg, (320-len(msg)*16)/2, y)
	}
}

func (g *Game) drawLevelSummary(screen *ebiten.Image) {
	// Use menu green as background
	screen.Fill(assets.STWordToRGBA(0x0020))
	// Use ST logical units (320x200) since drawText/drawDigits scale x2 internally
	lineH := 15
	vgap := 5
	lines := 6 // title, cleared, score, time, help, prompt
	totalH := lines*lineH + (lines-1)*vgap
	y := (200 - totalH) / 2
	// Title centered
	title := "WELL DONE!"
	g.drawText(screen, title, (320-len(title)*16)/2, y)
	y += lineH + vgap
	// Cleared stage centered
	cleared := "YOU CLEARED STAGE " + itoaDec(g.stage)
	g.drawText(screen, cleared, (320-len(cleared)*16)/2, y)
	y += lineH + vgap
	// Score & bonuses centered as label+value blocks (ST units)
	g.drawLabelValueCenteredST(screen, "SCORE:", g.score, y, 6)
	y += lineH + vgap
	g.drawLabelValueCenteredST(screen, "TIME BONUS:", g.pendingTimeBonus, y, 6)
	y += lineH + vgap
	g.drawLabelValueCenteredST(screen, "HELP BONUS:", g.pendingHelpBonus, y, 6)
	y += lineH + vgap
	// Continue prompt
	prompt := "HIT BUTTON TO GO ON!"
	g.drawText(screen, prompt, (320-len(prompt)*16)/2, y)
}

// drawLabelValueCentered centers a label (FONT2) + digits (FONT2 digits) as a single block.
func (g *Game) drawLabelValueCenteredST(screen *ebiten.Image, label string, value int, y int, max int) {
	// ST logical widths: label char =16, digit char =16; draw functions scale x2
	s := itoaDec(value)
	if len(s) > max {
		s = s[len(s)-max:]
	}
	labelW := len(label) * 16
	digitsW := len(s) * 16
	totalW := labelW + digitsW
	x := (320 - totalW) / 2
	g.drawText(screen, label, x, y)
	drawDigits(screen, g.atlas.Digits, value, x+labelW, y, max)
}

func (g *Game) updateInstructions() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || g.consumeAnyTap() {
		g.state = "menu"
	}
	return nil
}

func (g *Game) drawInstructions(screen *ebiten.Image) {
	// Fill with the same menu green as main menu
	screen.Fill(assets.STWordToRGBA(0x0020))
	// Text content
	paras := []string{
		"REMOVE ALL TILES IN PAIRS.",
		"ONLY IDENTICAL PAIRS MATCH.",
		"SEASONS MATCH WITH SEASONS.",
		"FLOWERS MATCH WITH FLOWERS.",
		"A PATH WITH AT MOST TWO TURNS MUST CONNECT THE TWO TILES.",
		"TOUCH OR CLICK TWO TILES. TAP THE HELP COUNTER FOR A HINT.",
		"USE THE BOTTOM BAR FOR MENU, PAUSE, RESTART, AND MUSIC.",
		"TOUCH ANYWHERE TO RETURN.",
	}
	// Use a readable system-safe font (Goregular at 16px) and center in both axes.
	face := g.instrFace
	if face == nil {
		face = basicfont.Face7x13
	}
	lines := wrapFace(paras, face, 600)
	m := face.Metrics()
	ascent := m.Ascent.Round()
	height := m.Height.Round()
	if height == 0 {
		height = ascent + m.Descent.Round()
	}
	vgap := height / 3
	if vgap < 6 {
		vgap = 6
	}
	totalH := len(lines)*height + (len(lines)-1)*vgap
	y := (400-totalH)/2 + ascent
	for _, ln := range lines {
		w := measureFace(face, ln)
		x := (640 - w) / 2
		text.Draw(screen, ln, face, x, y, color.White)
		y += height + vgap
	}
}

// wrapChars8 wraps paragraphs into lines that fit maxW with given spacing/scale.
func wrapChars8(paras []string, spacing, scale, maxW int) []string {
	measure := func(s string) int {
		if len(s) == 0 {
			return 0
		}
		w := 0
		for range s {
			w += (8 + spacing) * scale
		}
		w -= spacing * scale
		return w
	}
	out := []string{}
	for _, p := range paras {
		words := strings.Fields(p)
		cur := ""
		for _, w := range words {
			try := w
			if cur != "" {
				try = cur + " " + w
			}
			if measure(try) <= maxW {
				cur = try
			} else {
				if cur != "" {
					out = append(out, cur)
				}
				cur = w
			}
		}
		if cur != "" {
			out = append(out, cur)
		}
	}
	return out
}

// wrapFace wraps paragraphs using a font face to a given pixel width.
func wrapFace(paras []string, face font.Face, maxW int) []string {
	measure := func(s string) int { return measureFace(face, s) }
	out := []string{}
	for _, p := range paras {
		words := strings.Fields(p)
		cur := ""
		for _, w := range words {
			try := w
			if cur != "" {
				try = cur + " " + w
			}
			if measure(try) <= maxW {
				cur = try
			} else {
				if cur != "" {
					out = append(out, cur)
				}
				cur = w
			}
		}
		if cur != "" {
			out = append(out, cur)
		}
	}
	return out
}

// measureFace returns pixel width of a string for a font face.
func measureFace(face font.Face, s string) int {
	return font.MeasureString(face, s).Round()
}

func drawDigits(screen *ebiten.Image, digits [10]*image.RGBA, val int, x, y, max int) {
	// Convert to decimal string
	s := itoaDec(val)
	if len(s) > max {
		s = s[len(s)-max:]
	}
	for i := 0; i < len(s); i++ {
		d := s[i] - '0'
		if d < 0 || d > 9 {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(x*2+i*16*2), float64(y*2))
		if digits[int(d)] != nil {
			screen.DrawImage(ebiten.NewImageFromImage(digits[int(d)]), op)
		}
	}
}

type nameKey struct {
	label string
	value string
	rect  image.Rectangle
}

var nameKeys = buildNameKeys()

func buildNameKeys() []nameKey {
	const (
		keyWidth  = 50
		keyHeight = 44
		keyGap    = 6
	)
	rows := []struct {
		letters string
		y       int
	}{
		{"QWERTYUIOP", 105},
		{"ASDFGHJKL", 155},
		{"ZXCVBNM", 205},
		{"1234567890", 255},
	}
	keys := make([]nameKey, 0, 39)
	for _, row := range rows {
		width := len(row.letters)*keyWidth + (len(row.letters)-1)*keyGap
		x := (logicalWidth - width) / 2
		for _, letter := range row.letters {
			keys = append(keys, nameKey{
				label: string(letter),
				value: string(letter),
				rect:  image.Rect(x, row.y, x+keyWidth, row.y+keyHeight),
			})
			x += keyWidth + keyGap
		}
	}
	keys = append(keys,
		nameKey{label: "SPACE", value: " ", rect: image.Rect(43, 307, 347, 357)},
		nameKey{label: "DELETE", value: "\b", rect: image.Rect(353, 307, 477, 357)},
		nameKey{label: "OK", value: "\n", rect: image.Rect(483, 307, 597, 357)},
	)
	return keys
}

func nameKeyAt(x, y int) (string, bool) {
	point := image.Pt(x, y)
	for _, key := range nameKeys {
		if point.In(key.rect) {
			return key.value, true
		}
	}
	return "", false
}

// Enter-name state: physical or on-screen keyboard input and save.
func (g *Game) updateEnterName() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = "menu"
		return nil
	}
	// Read typed characters (debounced by Ebiten), accept letters/digits/space
	for _, r := range ebiten.InputChars() {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			g.appendName(unicode.ToUpper(r))
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.nameBuf) > 0 {
		g.nameBuf = g.nameBuf[:len(g.nameBuf)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(g.nameBuf) > 0 {
		g.submitEnteredName()
		return nil
	}
	if x, y, ok := g.consumeTap(); ok {
		if value, hit := nameKeyAt(x, y); hit {
			switch value {
			case "\b":
				if len(g.nameBuf) > 0 {
					g.nameBuf = g.nameBuf[:len(g.nameBuf)-1]
				}
			case "\n":
				if len(g.nameBuf) > 0 {
					g.submitEnteredName()
				}
			default:
				g.appendName([]rune(value)[0])
			}
		}
	}
	return nil
}

func (g *Game) submitEnteredName() {
	g.hs.submit(ScoreEntry{Name: g.nameBuf, Score: g.score})
	if err := saveHighscores(g.hsPath, g.hs); err != nil {
		log.Printf("save highscores: %v", err)
	}
	g.state = "highscores"
}

func (g *Game) drawEnterName(screen *ebiten.Image) {
	screen.Fill(assets.STWordToRGBA(0x0020))
	prompt := "ENTER YOUR NAME"
	g.drawText(screen, prompt, (320-len(prompt)*16)/2, 8)
	name := strings.ToUpper(g.nameBuf) + "_"
	g.drawText(screen, name, (320-len(name)*16)/2, 34)
	g.drawNameKeyboard(screen)
}

func (g *Game) drawNameKeyboard(screen *ebiten.Image) {
	for _, key := range nameKeys {
		x := float32(key.rect.Min.X)
		y := float32(key.rect.Min.Y)
		w := float32(key.rect.Dx())
		h := float32(key.rect.Dy())
		vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{225, 225, 235, 255}, false)
		vector.DrawFilledRect(screen, x+2, y+2, w-4, h-4, color.RGBA{45, 45, 70, 255}, false)
		g.drawSystemTextCenteredInRect(screen, key.label, key.rect.Min.X, key.rect.Min.Y, key.rect.Dx(), key.rect.Dy(), color.White)
	}
}

func (g *Game) appendName(r rune) {
	if len(g.nameBuf) >= 12 {
		return
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
		g.nameBuf += string(r)
	}
}
