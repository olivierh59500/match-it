package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/olivierh59500/match-it/internal/assets"
)

// Simple utility to convert the original Atari ST assets to PNGs for Ebiten usage.
// Output goes under assets/png/.
func main() {
	// Helper: resolve originals either from repo root or old/ subdir.
	find := func(name string) string {
		if _, err := os.Stat(filepath.Join("old", name)); err == nil {
			return filepath.Join("old", name)
		}
		return name
	}

	// Decode palette from tiles.neo and reuse for IMG blocks.
	_, pal, err := assets.DecodeNEO(find("tiles.neo"))
	if err != nil {
		log.Fatalf("decode tiles.neo: %v", err)
	}
	if err := os.MkdirAll("assets/png", 0o755); err != nil {
		log.Fatal(err)
	}

	// Convert full-screen blocks
	convertScreen := func(name string, h int, p [16]color.RGBA) {
		img, err := assets.DecodeScreenIMG(find(name), 320, h, p)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		f, err := os.Create(fmt.Sprintf("assets/png/%s.png", name))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote assets/png/%s.png", name)
	}
	pal1 := assets.Palette1()
	convertScreen("obenplat.img", 16, pal1)
	convertScreen("menuplat.img", 16, pal1)
	// EISPLA2.IMG is packed and needs Backform decode (per assembly backform routine)
	// Use in-game palette2 from the assembly constants for best colors.
	eisPal := assets.Palette2()
	if img, err := assets.DecodeBackformIMG(find("eispla2.img"), 320, 181, eisPal); err == nil {
		f, err := os.Create("assets/png/eispla2.img.png")
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			log.Fatal(err)
		}
		_ = f.Close()
		log.Printf("wrote assets/png/eispla2.img.png")
	} else {
		log.Fatalf("eispla2: %v", err)
	}

	// Convert tile set (43 tiles + 6 path glyphs)
	tiles, err := assets.TileSetFromTilesIMG(find("tiles.img"), eisPal)
	if err != nil {
		log.Fatalf("tiles: %v", err)
	}
	for i, t := range tiles {
		f, err := os.Create(fmt.Sprintf("assets/png/tile_%02d.png", i))
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, t); err != nil {
			log.Fatal(err)
		}
		_ = f.Close()
	}
	log.Printf("wrote %d tile PNGs", len(tiles))

	// Convert budgie splash (full screen, from budgie.img) using same palette
	// Note: budgie.img has its own palette in code; this reuse may differ slightly.
	if img, err := assets.DecodeScreenIMG(find("budgie.img"), 320, 200, pal); err == nil {
		f, _ := os.Create("assets/png/budgie.png")
		_ = png.Encode(f, img)
		_ = f.Close()
	}

	// Export FONT2 digits and glyphs as PNGs
	if err := os.MkdirAll(filepath.Join("assets", "png", "font2"), 0o755); err != nil {
		log.Fatal(err)
	}
	if raw, err := os.ReadFile(find("font2.img")); err == nil {
		if digits, err := assets.LoadFontDigits(raw, pal1); err == nil {
			for i, d := range digits {
				if d == nil {
					continue
				}
				f, err := os.Create(fmt.Sprintf("assets/png/font2/digit_%d.png", i))
				if err != nil {
					log.Fatal(err)
				}
				_ = png.Encode(f, d)
				_ = f.Close()
			}
		}
		if glyphs, err := assets.LoadFontGlyphs47(raw, pal1); err == nil {
			for i, g := range glyphs {
				if g == nil {
					continue
				}
				f, err := os.Create(fmt.Sprintf("assets/png/font2/glyph_%02d.png", i))
				if err != nil {
					log.Fatal(err)
				}
				_ = png.Encode(f, g)
				_ = f.Close()
			}
		}
	}

	// Export help plates (6 counters) as PNGs
	if err := os.MkdirAll(filepath.Join("assets", "png", "help"), 0o755); err != nil {
		log.Fatal(err)
	}
	if plates, err := assets.LoadHelpPlates(find("helpplat.img"), pal1); err == nil {
		for i, p := range plates {
			f, err := os.Create(fmt.Sprintf("assets/png/help/help_%d.png", i))
			if err != nil {
				log.Fatal(err)
			}
			_ = png.Encode(f, p)
			_ = f.Close()
		}
	} else {
		log.Fatalf("helpplat: %v", err)
	}

	// Export chars8 full sheet (32xN glyphs of 8x8); tolerate partial data
	if data, err := assets.LoadChars8FromAsm(find("match_it.s")); err == nil {
		blocks := len(data) / 256 // each block: 8 rows * 32 bytes
		if blocks > 8 {
			blocks = 8
		}
		if blocks > 0 {
			w, h := 32*8, blocks*8
			sheet := image.NewRGBA(image.Rect(0, 0, w, h))
			for blk := 0; blk < blocks; blk++ {
				for col := 0; col < 32; col++ {
					x0 := col * 8
					y0 := blk * 8
					base := blk*256 + col
					for row := 0; row < 8; row++ {
						idx := base + row*32
						if idx < 0 || idx >= len(data) {
							continue
						}
						b := data[idx]
						for bit := 0; bit < 8; bit++ {
							if (b & (1 << (7 - uint(bit)))) != 0 {
								sheet.SetRGBA(x0+bit, y0+row, color.RGBA{255, 255, 255, 255})
							} else {
								sheet.SetRGBA(x0+bit, y0+row, color.RGBA{0, 0, 0, 0})
							}
						}
					}
				}
			}
			if err := os.MkdirAll(filepath.Join("assets", "png"), 0o755); err == nil {
				if f, err := os.Create("assets/png/chars8_sheet.png"); err == nil {
					_ = png.Encode(f, sheet)
					_ = f.Close()
				}
			}
		} else {
			log.Printf("chars8: no data parsed; skipping PNG export")
		}
	}

	// Export gamearea (levels) to PNG: 64 rows x 240 bytes -> 240x64 grayscale
	rawGA, err := os.ReadFile(find("gamearea.img"))
	if err != nil {
		log.Fatalf("gamearea: %v", err)
	}
	if len(rawGA) != 64*240 {
		log.Fatalf("gamearea: unexpected size %d", len(rawGA))
	}
	gaImg := image.NewRGBA(image.Rect(0, 0, 240, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 240; x++ {
			v := rawGA[y*240+x]
			gaImg.SetRGBA(x, y, color.RGBA{v, v, v, 0xFF})
		}
	}
	if f, err := os.Create("assets/png/gamearea.img.png"); err == nil {
		_ = png.Encode(f, gaImg)
		_ = f.Close()
		log.Printf("wrote assets/png/gamearea.img.png")
	} else {
		log.Fatalf("gamearea write: %v", err)
	}

	// Export remove animation masks to PNG (16x20 per frame, stacked vertically)
	rawRem, err := os.ReadFile(find("removean.img"))
	if err != nil {
		log.Fatalf("removean: %v", err)
	}
	if len(rawRem) != 15*20*2 {
		log.Fatalf("removean: unexpected size %d", len(rawRem))
	}
	remImg := image.NewRGBA(image.Rect(0, 0, 16, 20*15))
	off := 0
	for fidx := 0; fidx < 15; fidx++ {
		for row := 0; row < 20; row++ {
			word := binary.BigEndian.Uint16(rawRem[off : off+2])
			off += 2
			for x := 0; x < 16; x++ {
				bit := (word >> (15 - x)) & 1
				if bit != 0 {
					remImg.SetRGBA(x, fidx*20+row, color.RGBA{255, 255, 255, 255})
				} else {
					remImg.SetRGBA(x, fidx*20+row, color.RGBA{0, 0, 0, 0})
				}
			}
		}
	}
	if err := os.MkdirAll(filepath.Join("assets", "png", "remove"), 0o755); err != nil {
		log.Fatal(err)
	}
	if f, err := os.Create(filepath.Join("assets", "png", "remove", "removean.img.png")); err == nil {
		_ = png.Encode(f, remImg)
		_ = f.Close()
		log.Printf("wrote assets/png/remove/removean.img.png")
	} else {
		log.Fatalf("remove write: %v", err)
	}
}

// palette functions moved to internal/assets/palette.go
