package game

import (
	"fmt"
	"image"
	"image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	resources "github.com/olivierh59500/match-it/assets"
)

const (
	tileImageCount    = 49
	playableTileCount = 43
	removeFrameCount  = 15
)

// atlas contains GPU-ready images. CPU images exist only while loading, so no
// PNG is decoded or uploaded again from Draw.
type atlas struct {
	Tiles         []*ebiten.Image // tile_00.png .. tile_48.png
	BGTop         *ebiten.Image   // obenplat.img.png
	BGArea        *ebiten.Image   // eispla2.img.png
	MenuPlate     *ebiten.Image   // menuplat.img.png
	Digits        [10]*ebiten.Image
	HelpPlates    []*ebiten.Image
	Font47        []*ebiten.Image
	RemovalSheet  *ebiten.Image
	RemovalFrames [playableTileCount][removeFrameCount]*ebiten.Image
}

func loadPNG(name string) (*image.RGBA, error) {
	f, err := resources.Files.Open(name)
	if err != nil {
		return nil, err
	}
	m, err := png.Decode(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if img, ok := m.(*image.RGBA); ok {
		return img, nil
	}
	b := m.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, m.At(x, y))
		}
	}
	return out, nil
}

func loadGPUImage(name string) (*ebiten.Image, error) {
	img, err := loadPNG(name)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

func (g *Game) tryLoadAtlas() {
	a := &atlas{
		Tiles:      make([]*ebiten.Image, tileImageCount),
		HelpPlates: make([]*ebiten.Image, 6),
		Font47:     make([]*ebiten.Image, 47),
	}
	tileSources := make([]*image.RGBA, tileImageCount)

	for i := 0; i < tileImageCount; i++ {
		name := "png/" + sprintfTile(i)
		img, err := loadPNG(name)
		if err != nil {
			log.Printf("asset %s: %v", name, err)
			continue
		}
		tileSources[i] = img
		a.Tiles[i] = ebiten.NewImageFromImage(img)
	}

	var err error
	if a.BGTop, err = loadGPUImage("png/obenplat.img.png"); err != nil {
		log.Printf("asset BGTop: %v", err)
	}
	if a.BGArea, err = loadGPUImage("png/eispla2.img.png"); err != nil {
		log.Printf("asset BGArea: %v", err)
	}
	if a.MenuPlate, err = loadGPUImage("png/menuplat.img.png"); err != nil {
		log.Printf("asset MenuPlate: %v", err)
	}

	for i := range a.Digits {
		name := fmt.Sprintf("png/font2/digit_%d.png", i)
		if a.Digits[i], err = loadGPUImage(name); err != nil {
			log.Printf("asset %s: %v", name, err)
		}
	}
	for i := range a.Font47 {
		name := fmt.Sprintf("png/font2/glyph_%02d.png", i)
		if a.Font47[i], err = loadGPUImage(name); err != nil {
			log.Printf("asset %s: %v", name, err)
		}
	}
	for i := range a.HelpPlates {
		name := fmt.Sprintf("png/help/help_%d.png", i)
		if a.HelpPlates[i], err = loadGPUImage(name); err != nil {
			log.Printf("asset %s: %v", name, err)
		}
	}

	g.loadRemovalFrames(a, tileSources)
	g.atlas = a
	g.boardCanvasDirty = true
}

func sprintfTile(i int) string { return "tile_" + itoa2(byte(i)) + ".png" }
