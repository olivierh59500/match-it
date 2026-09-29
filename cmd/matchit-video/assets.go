package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	resources "github.com/olivierh59500/match-it/assets"
	assetdecoder "github.com/olivierh59500/match-it/internal/assets"
	"github.com/olivierh59500/match-it/internal/logic"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	videoWidth  = 640
	videoHeight = 400
	boardCols   = 18
	boardRows   = 8
	boardX      = 32
	boardY      = 48
	tileWidth   = 32
	tileHeight  = 40
)

type demoAssets struct {
	tiles         [49]image.Image
	removalFrames [43][15]*image.RGBA
	digits        [10]image.Image
	glyphs        [47]image.Image
	help          [6]image.Image
	backgroundTop image.Image
	background    image.Image
	menuPlate     image.Image
	splash        image.Image
	face          font.Face
}

func loadDemoAssets() (*demoAssets, error) {
	a := &demoAssets{}
	var err error
	for i := range a.tiles {
		if a.tiles[i], err = readEmbeddedPNG(fmt.Sprintf("png/tile_%02d.png", i)); err != nil {
			return nil, err
		}
	}
	for i := range a.digits {
		if a.digits[i], err = readEmbeddedPNG(fmt.Sprintf("png/font2/digit_%d.png", i)); err != nil {
			return nil, err
		}
	}
	for i := range a.glyphs {
		if a.glyphs[i], err = readEmbeddedPNG(fmt.Sprintf("png/font2/glyph_%02d.png", i)); err != nil {
			return nil, err
		}
	}
	for i := range a.help {
		if a.help[i], err = readEmbeddedPNG(fmt.Sprintf("png/help/help_%d.png", i)); err != nil {
			return nil, err
		}
	}
	if a.backgroundTop, err = readEmbeddedPNG("png/obenplat.img.png"); err != nil {
		return nil, err
	}
	if a.background, err = readEmbeddedPNG("png/eispla2.img.png"); err != nil {
		return nil, err
	}
	if a.menuPlate, err = readEmbeddedPNG("png/menuplat.img.png"); err != nil {
		return nil, err
	}
	if a.splash, err = readEmbeddedPNG("png/malakhsoftware-pixel.png"); err != nil {
		return nil, err
	}

	fontData, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	a.face, err = opentype.NewFace(fontData, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	maskFile, err := resources.Files.Open("png/remove/removean.img.png")
	if err != nil {
		return nil, err
	}
	masks, decodeErr := assetdecoder.DecodeRemoveMasks(maskFile)
	closeErr := maskFile.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	for tile := range a.removalFrames {
		for frame := range a.removalFrames[tile] {
			a.removalFrames[tile][frame] = maskTile(a.tiles[tile], &masks[frame])
		}
	}
	return a, nil
}

func readEmbeddedPNG(name string) (image.Image, error) {
	f, err := resources.Files.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	img, decodeErr := png.Decode(f)
	closeErr := f.Close()
	if decodeErr != nil {
		return nil, fmt.Errorf("decode %s: %w", name, decodeErr)
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return img, nil
}

func maskTile(source image.Image, mask *[20]uint16) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, 16, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 16; x++ {
			if mask[y]&(1<<uint(15-x)) == 0 {
				continue
			}
			out.Set(x, y, source.At(source.Bounds().Min.X+x, source.Bounds().Min.Y+y))
		}
	}
	return out
}

func (a *demoAssets) renderSplash(dst *image.RGBA) {
	fill(dst, color.White)
	b := a.splash.Bounds()
	draw.Draw(dst, image.Rect((videoWidth-b.Dx())/2, (videoHeight-b.Dy())/2, (videoWidth+b.Dx())/2, (videoHeight+b.Dy())/2), a.splash, b.Min, draw.Over)
}

func (a *demoAssets) renderMenu(dst *image.RGBA) {
	fill(dst, assetdecoder.STWordToRGBA(0x0020))
	for _, y := range []int{37, 74, 111, 148} {
		drawScaled(dst, a.menuPlate, 0, y*2, 2)
	}
	a.drawRetroText(dst, "MATCH IT", (320-16*8)/2, 10)
	items := []string{"START GAME", "HIGHSCORES", "INSTRUCTIONS", "QUIT"}
	for i, label := range items {
		x := (320 - len(label)*16) / 2
		a.drawRetroText(dst, label, x, []int{38, 75, 112, 149}[i])
	}
}

type gameRenderState struct {
	board       *logic.Board
	score       int
	timeLeft    int
	helpCount   int
	first       *tilePoint
	move        *demoMove
	removeFrame int
	cursor      cursorState
	pulse       int
}

func (a *demoAssets) renderGame(dst *image.RGBA, state gameRenderState) {
	fill(dst, color.RGBA{0, 0, 32, 255})
	drawScaled(dst, a.backgroundTop, 0, 0, 2)
	drawScaled(dst, a.background, 0, 34, 2)

	for y := 0; y < boardRows; y++ {
		for x := 0; x < boardCols; x++ {
			tile := state.board.Get(x, y)
			if tile == 0 {
				continue
			}
			img := a.tiles[int(tile-1)]
			if state.move != nil && state.removeFrame >= 0 {
				if x == state.move.x1 && y == state.move.y1 {
					img = a.removalFrames[int(state.move.tile1-1)][state.removeFrame]
				} else if x == state.move.x2 && y == state.move.y2 {
					img = a.removalFrames[int(state.move.tile2-1)][state.removeFrame]
				}
			}
			drawScaled(dst, img, boardX+x*tileWidth, boardY+y*tileHeight, 2)
		}
	}

	if state.move != nil && state.removeFrame >= 0 {
		a.drawPath(dst, state.move)
		drawSelection(dst, state.move.x1, state.move.y1, "1", false, state.pulse)
		drawSelection(dst, state.move.x2, state.move.y2, "2", false, state.pulse)
	} else if state.first != nil {
		drawSelection(dst, state.first.x, state.first.y, "1", true, state.pulse)
	}

	a.drawNumber(dst, state.score, 3*16, 1, 5)
	a.drawNumber(dst, state.timeLeft, 10*16, 1, 3)
	help := state.helpCount
	if help < 0 {
		help = 0
	}
	if help > 5 {
		help = 5
	}
	drawScaled(dst, a.help[help], 512, 0, 2)

	fillRect(dst, image.Rect(0, 368, 640, 400), color.RGBA{20, 20, 35, 235})
	labels := []string{"MENU", "PAUSE", "RESTART", "MUSIC"}
	for i, label := range labels {
		if i > 0 {
			fillRect(dst, image.Rect(i*160, 372, i*160+1, 396), color.RGBA{180, 180, 200, 180})
		}
		a.drawSystemCentered(dst, label, image.Rect(i*160, 368, (i+1)*160, 400), color.White)
	}
	drawCursor(dst, state.cursor)
}

func (a *demoAssets) drawPath(dst *image.RGBA, move *demoMove) {
	x, y := move.x1, move.y1
	for i, direction := range move.path {
		switch direction {
		case 1:
			x++
		case 2:
			x--
		case 3:
			y++
		case 4:
			y--
		}
		next := byte(0)
		if i+1 < len(move.path) {
			next = move.path[i+1]
		}
		glyph := pathGlyphIndex(direction, next)
		drawScaled(dst, a.tiles[43+glyph], boardX+x*tileWidth, boardY+y*tileHeight, 2)
	}
}

func (a *demoAssets) renderSummary(dst *image.RGBA, stage, score, timeBonus, helpBonus int) {
	fill(dst, assetdecoder.STWordToRGBA(0x0020))
	lineHeight, gap := 15, 5
	y := (200 - (6*lineHeight + 5*gap)) / 2
	a.drawCenteredRetro(dst, "WELL DONE!", y)
	y += lineHeight + gap
	a.drawCenteredRetro(dst, "YOU CLEARED STAGE "+itoa(stage), y)
	y += lineHeight + gap
	a.drawLabelNumber(dst, "SCORE:", score, y, 6)
	y += lineHeight + gap
	a.drawLabelNumber(dst, "TIME BONUS:", timeBonus, y, 6)
	y += lineHeight + gap
	a.drawLabelNumber(dst, "HELP BONUS:", helpBonus, y, 6)
	y += lineHeight + gap
	a.drawCenteredRetro(dst, "HIT BUTTON TO GO ON!", y)
}

func (a *demoAssets) drawRetroText(dst *image.RGBA, text string, x, y int) {
	position := x
	for _, r := range strings.ToUpper(text) {
		if r != ' ' {
			index := assetdecoder.FontIndexForChar(r)
			if index >= 0 && index < len(a.glyphs) {
				drawScaled(dst, a.glyphs[index], position*2, y*2, 2)
			}
		}
		position += 16
	}
}

func (a *demoAssets) drawCenteredRetro(dst *image.RGBA, text string, y int) {
	a.drawRetroText(dst, text, (320-len(text)*16)/2, y)
}

func (a *demoAssets) drawNumber(dst *image.RGBA, value, x, y, maxDigits int) {
	text := itoa(value)
	if len(text) > maxDigits {
		text = text[len(text)-maxDigits:]
	}
	for i := range text {
		digit := int(text[i] - '0')
		if digit >= 0 && digit < len(a.digits) {
			drawScaled(dst, a.digits[digit], x*2+i*32, y*2, 2)
		}
	}
}

func (a *demoAssets) drawLabelNumber(dst *image.RGBA, label string, value, y, maxDigits int) {
	number := itoa(value)
	if len(number) > maxDigits {
		number = number[len(number)-maxDigits:]
	}
	x := (320 - (len(label)+len(number))*16) / 2
	a.drawRetroText(dst, label, x, y)
	a.drawNumber(dst, value, x+len(label)*16, y, maxDigits)
}

func (a *demoAssets) drawSystemCentered(dst *image.RGBA, text string, rect image.Rectangle, clr color.Color) {
	metrics := a.face.Metrics()
	width := font.MeasureString(a.face, text).Round()
	height := metrics.Height.Round()
	baseline := rect.Min.Y + (rect.Dy()-height)/2 + metrics.Ascent.Round()
	drawer := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(clr),
		Face: a.face,
		Dot:  fixed.P(rect.Min.X+(rect.Dx()-width)/2, baseline),
	}
	drawer.DrawString(text)
}

func drawSelection(dst *image.RGBA, x, y int, label string, tint bool, pulse int) {
	left, top := boardX+x*tileWidth, boardY+y*tileHeight
	phase := pulse % 30
	if phase > 15 {
		phase = 30 - phase
	}
	gold := color.RGBA{255, uint8(170 + phase*5), 0, 255}
	if tint {
		fillAlphaRect(dst, image.Rect(left+4, top+4, left+tileWidth-4, top+tileHeight-4), color.RGBA{255, 210, 0, uint8(35 + phase)})
	}
	drawFrame(dst, left, top, tileWidth, tileHeight, 4, color.RGBA{0, 0, 0, 230})
	drawFrame(dst, left+2, top+2, tileWidth-4, tileHeight-4, 2, gold)
	fillRect(dst, image.Rect(left+3, top+3, left+18, top+21), gold)
	fillRect(dst, image.Rect(left+5, top+5, left+16, top+19), color.RGBA{15, 15, 20, 255})
	drawBasicCentered(dst, label, image.Rect(left+3, top+3, left+18, top+21), color.White)
}

func drawFrame(dst *image.RGBA, x, y, width, height, thickness int, clr color.Color) {
	fillAlphaRect(dst, image.Rect(x, y, x+width, y+thickness), clr)
	fillAlphaRect(dst, image.Rect(x, y+height-thickness, x+width, y+height), clr)
	fillAlphaRect(dst, image.Rect(x, y+thickness, x+thickness, y+height-thickness), clr)
	fillAlphaRect(dst, image.Rect(x+width-thickness, y+thickness, x+width, y+height-thickness), clr)
}

func drawBasicCentered(dst *image.RGBA, text string, rect image.Rectangle, clr color.Color) {
	face := basicfont.Face7x13
	width := font.MeasureString(face, text).Round()
	metrics := face.Metrics()
	baseline := rect.Min.Y + (rect.Dy()-metrics.Height.Round())/2 + metrics.Ascent.Round()
	drawer := font.Drawer{Dst: dst, Src: image.NewUniform(clr), Face: face, Dot: fixed.P(rect.Min.X+(rect.Dx()-width)/2, baseline)}
	drawer.DrawString(text)
}

func drawScaled(dst draw.Image, src image.Image, x, y, scale int) {
	b := src.Bounds()
	rect := image.Rect(x, y, x+b.Dx()*scale, y+b.Dy()*scale)
	xdraw.NearestNeighbor.Scale(dst, rect, src, b, draw.Over, nil)
}

func fill(dst draw.Image, clr color.Color) {
	draw.Draw(dst, dst.Bounds(), image.NewUniform(clr), image.Point{}, draw.Src)
}

func fillRect(dst draw.Image, rect image.Rectangle, clr color.Color) {
	draw.Draw(dst, rect, image.NewUniform(clr), image.Point{}, draw.Src)
}

func fillAlphaRect(dst draw.Image, rect image.Rectangle, clr color.Color) {
	draw.Draw(dst, rect, image.NewUniform(clr), image.Point{}, draw.Over)
}

type cursorState struct {
	x, y     float64
	click    bool
	progress float64
}

var cursorPixels = []string{
	"X...........",
	"XX..........",
	"XOX.........",
	"XOOX........",
	"XOOOX.......",
	"XOOOOX......",
	"XOOOOOX.....",
	"XOOOOOOX....",
	"XOOOOOOOX...",
	"XOOOOXXXXX..",
	"XOOXOOX.....",
	"XOX.XOOX....",
	"XX..XOOX....",
	"X....XOOX...",
	".....XOOX...",
	"......XX....",
}

func drawCursor(dst *image.RGBA, cursor cursorState) {
	left, top := int(math.Round(cursor.x)), int(math.Round(cursor.y))
	for y, row := range cursorPixels {
		for x, pixel := range row {
			switch pixel {
			case 'X':
				dst.Set(left+x, top+y, color.Black)
			case 'O':
				dst.Set(left+x, top+y, color.White)
			}
		}
	}
	if cursor.click {
		radius := 5 + int(cursor.progress*10)
		alpha := uint8(255 * (1 - cursor.progress))
		drawCircleOutline(dst, left, top, radius, color.RGBA{255, 180, 0, alpha})
	}
}

func drawCircleOutline(dst *image.RGBA, cx, cy, radius int, clr color.RGBA) {
	inner := float64(radius) - 1.25
	outer := float64(radius) + 1.25
	for y := -radius - 2; y <= radius+2; y++ {
		for x := -radius - 2; x <= radius+2; x++ {
			distance := math.Hypot(float64(x), float64(y))
			if distance >= inner && distance <= outer {
				blendPixel(dst, cx+x, cy+y, clr)
			}
		}
	}
}

func blendPixel(dst *image.RGBA, x, y int, source color.RGBA) {
	if !image.Pt(x, y).In(dst.Bounds()) {
		return
	}
	destination := dst.RGBAAt(x, y)
	alpha := uint16(source.A)
	inverse := 255 - alpha
	dst.SetRGBA(x, y, color.RGBA{
		R: uint8((uint16(source.R)*alpha + uint16(destination.R)*inverse) / 255),
		G: uint8((uint16(source.G)*alpha + uint16(destination.G)*inverse) / 255),
		B: uint8((uint16(source.B)*alpha + uint16(destination.B)*inverse) / 255),
		A: 255,
	})
}

func pathGlyphIndex(current, next byte) int {
	switch current {
	case 1:
		if next == 3 {
			return 4
		}
		if next == 4 {
			return 3
		}
		return 0
	case 2:
		if next == 3 {
			return 5
		}
		if next == 4 {
			return 2
		}
		return 0
	case 3:
		if next == 1 {
			return 2
		}
		if next == 2 {
			return 3
		}
		return 1
	case 4:
		if next == 1 {
			return 5
		}
		if next == 2 {
			return 4
		}
		return 1
	default:
		return 0
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}
