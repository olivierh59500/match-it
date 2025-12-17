package logic

// Board encapsulates the 18x8 tile grid and the pos list used by the original help routine.
// Tiles use 0 for empty, 1..42 for tile IDs.
type Board struct {
    Tiles [18 * 8]byte // row-major
    Pos   [16 * 6]byte // position list order for help/checkmate detection
}

func (b *Board) Get(x, y int) byte { return b.Tiles[y*18+x] }
func (b *Board) Set(x, y int, v byte) { b.Tiles[y*18+x] = v }

// FromLevel copies a board + pos order from level data.
func (b *Board) FromLevel(tiles [18 * 8]byte, pos [16 * 6]byte) {
    b.Tiles = tiles
    b.Pos = pos
}

// RemovePair clears two tiles and removes their entries from Pos (sets to 0), matching makeparasforclear.
func (b *Board) RemovePair(x1, y1, x2, y2 int) {
    // clear tiles
    b.Set(x1, y1, 0)
    b.Set(x2, y2, 0)
    // zero positions in Pos buffer
    idxs := [2]byte{byte(y1*18 + x1), byte(y2*18 + x2)}
    for i := 0; i < len(b.Pos); i++ {
        if b.Pos[i] == idxs[0] {
            b.Pos[i] = 0
        } else if b.Pos[i] == idxs[1] {
            b.Pos[i] = 0
        }
    }
}

// HelpSearch replicates helpfunktion/checkmatetest scan order using Pos list.
// Returns first found pair with a valid path (using FindPath). If none found, ok=false.
func (b *Board) HelpSearch() (x1, y1, x2, y2 int, path []byte, ok bool) {
    var p []byte
    // Scan from the end, pairing earlier entries per the original.
    for i := len(b.Pos) - 1; i >= 0; i-- {
        p1 := b.Pos[i]
        if p1 == 0 {
            continue
        }
        for j := i - 1; j >= 0; j-- {
            p2 := b.Pos[j]
            if p2 == 0 {
                continue
            }
            v1 := b.Tiles[p1]
            v2 := b.Tiles[p2]
            if !MatchOK(v1, v2) {
                continue
            }
            x1, y1 = int(p1%18), int(p1/18)
            x2, y2 = int(p2%18), int(p2/18)
            if FindPath(b.Tiles, x1, y1, x2, y2, &p) {
                return x1, y1, x2, y2, append([]byte(nil), p...), true
            }
        }
    }
    return 0, 0, 0, 0, nil, false
}

