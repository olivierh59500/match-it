package logic

import (
	"fmt"
	"image/png"
	"os"
)

// LevelSet holds the 64 predefined boards bundled in GAMEAREA.IMG.
// Each level block is 240 bytes: 144 bytes for matchitbuff (18*8), then 96 bytes for posbuff (16*6).
type LevelSet struct {
	Boards   [64][18 * 8]byte
	PosList  [64][16 * 6]byte
	LevelTab [64]byte // randomized permutation as in make_leveltab
	level    int      // current rotating index [0..63]
	rng      *parkMiller
}

// LoadLevels loads the level table from a PNG dump of GAMEAREA.IMG (240x64 grayscale)
// and prepares the level permutation like make_leveltab.
func LoadLevels(path string) (*LevelSet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode gamearea png: %w", err)
	}
	rect := img.Bounds()
	if rect.Dx() != 240 || rect.Dy() != 64 {
		return nil, fmt.Errorf("unexpected gamearea size: %dx%d", rect.Dx(), rect.Dy())
	}
	raw := make([]byte, 0, 64*240)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			raw = append(raw, byte(r>>8))
		}
	}
	ls := &LevelSet{rng: newRNG()}
	// Parse 64 blocks of 240 bytes
	for i := 0; i < 64; i++ {
		off := i * 240
		copy(ls.Boards[i][:], raw[off:off+144])
		copy(ls.PosList[i][:], raw[off+144:off+240])
	}
	// Build random permutation LevelTab (64 entries), matching make_leveltab.
	// Fill with -1 then pick unused slots randomly.
	for i := range ls.LevelTab {
		ls.LevelTab[i] = 0xFF
	}
	for n := 63; n >= 0; n-- {
		for {
			idx := ls.rng.Intn(64)
			if ls.LevelTab[idx] == 0xFF {
				ls.LevelTab[idx] = byte(n)
				break
			}
		}
	}
	// Original picks initial level index via two RNG calls, multiply, and mask &63.
	a := int(ls.rng.NextUint32())
	b := int(ls.rng.NextUint32())
	ls.level = (a * b) & 63
	return ls, nil
}

// NextBoard emulates make_buff: advances level, maps through LevelTab, and returns
// the board and position list copies.
func (ls *LevelSet) NextBoard() (board [18 * 8]byte, pos [16 * 6]byte) {
	ls.level = (ls.level + 1) & 63
	idx := ls.LevelTab[ls.level]
	board = ls.Boards[idx]
	pos = ls.PosList[idx]
	return
}

// Debug helper to render a board into rows for logs.
func BoardString(b [18 * 8]byte) string {
	out := make([]byte, 0, 18*8*3)
	for y := 0; y < 8; y++ {
		for x := 0; x < 18; x++ {
			v := b[y*18+x]
			if v == 0 {
				out = append(out, []byte(" . ")...)
			} else {
				out = append(out, byte('0'+(v/10)))
				out = append(out, byte('0'+(v%10)))
				out = append(out, ' ')
			}
		}
		if y != 7 {
			out = append(out, '\n')
		}
	}
	return string(out)
}
