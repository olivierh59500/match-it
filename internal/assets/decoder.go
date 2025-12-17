package assets

import (
    "errors"
    "fmt"
    "image"
    "image/color"
    "os"
)

// ST 16-color palette: 12-bit RGB stored as 0x0RGB, 3 bits per channel (0..7) scaled to 0..255.
// STWordToRGBA converts a ST 12-bit palette word (0RGB) to RGBA.
func STWordToRGBA(w uint16) color.RGBA {
    r := uint8(((w>>8)&0x7) * 255 / 7)
    g := uint8(((w>>4)&0x7) * 255 / 7)
    b := uint8(((w>>0)&0x7) * 255 / 7)
    return color.RGBA{R: r, G: g, B: b, A: 255}
}

// DecodeNEO decodes a standard .NEO (NeoChrome) file (320x200x4bpp interleaved) to an RGBA image and palette.
func DecodeNEO(path string) (img *image.RGBA, pal [16]color.RGBA, err error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, pal, err
    }
    if len(raw) < 128+32000 {
        return nil, pal, fmt.Errorf("neo too small: %d", len(raw))
    }
    // Palette starts at offset 2 (after header), 16 words big-endian.
    for i := 0; i < 16; i++ {
        w := (uint16(raw[2+2*i]) << 8) | uint16(raw[3+2*i])
        pal[i] = STWordToRGBA(w)
    }
    // Bitmap starts at offset 128, 200 lines, each line 160 bytes.
    out := image.NewRGBA(image.Rect(0, 0, 320, 200))
    off := 128
    for y := 0; y < 200; y++ {
        line := raw[off : off+160]
        off += 160
        // 20 groups of 16 pixels; each group has 4 words (one per plane).
        // Memory order per group: plane0[2], plane1[2], plane2[2], plane3[2].
        for group := 0; group < 20; group++ {
            // Load 4 words for this 16-pixel group
            p0 := uint16(line[group*8+0])<<8 | uint16(line[group*8+1])
            p1 := uint16(line[group*8+2])<<8 | uint16(line[group*8+3])
            p2 := uint16(line[group*8+4])<<8 | uint16(line[group*8+5])
            p3 := uint16(line[group*8+6])<<8 | uint16(line[group*8+7])
            for bit := 0; bit < 16; bit++ {
                // ST pixels are MSB first.
                mask := uint16(1 << (15 - bit))
                // Reconstruct color index from bitplanes explicitly
                var c uint8
                if (p0 & mask) != 0 { c |= 1 }
                if (p1 & mask) != 0 { c |= 2 }
                if (p2 & mask) != 0 { c |= 4 }
                if (p3 & mask) != 0 { c |= 8 }
                x := group*16 + bit
                out.SetRGBA(x, y, pal[c])
            }
        }
    }
    return out, pal, nil
}

// ErrUnsupportedIMG is returned for unknown IMG segment sizes.
var ErrUnsupportedIMG = errors.New("unsupported IMG format for this asset")

// DecodeScreenIMG decodes raw screen-order IMG blocks used by the original code for rectangular regions.
// You must supply width in pixels (must be 320) and height in scanlines.
func DecodeScreenIMG(path string, width, height int, pal [16]color.RGBA) (*image.RGBA, error) {
    if width != 320 {
        return nil, fmt.Errorf("only 320px width supported")
    }
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    // Size must be height*160 bytes (4bpp interleaved across planes per ST framebuffer order).
    if len(raw) != height*160 {
        return nil, fmt.Errorf("size mismatch: got %d, want %d", len(raw), height*160)
    }
    return rgbaFromSTScreen(raw, width, height, pal)
}

// TileSetFromTilesIMG decodes tile atlas from TILES.IMG: 49 tiles of 8x20 pixels (43 tiles + 6 path glyphs).
// This uses the screen-order chunks (160 bytes per tile). Returns a 49-sprite slice.
func TileSetFromTilesIMG(path string, pal [16]color.RGBA) ([]*image.RGBA, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    if len(raw)%160 != 0 {
        return nil, fmt.Errorf("tiles.img unexpected size: %d", len(raw))
    }
    n := len(raw) / 160
    tiles := make([]*image.RGBA, n)
    for i := 0; i < n; i++ {
        off := i * 160
        // Each tile is 16x20 pixels, stored as five 4-scanline chunks (5*32 = 160 bytes).
        img := image.NewRGBA(image.Rect(0, 0, 16, 20))
        // Interpret the 160 bytes as five blocks, each block is 32 bytes representing 4 consecutive scanlines.
        // The original code places 8 longs per block to four successive scanlines; here we unpack line by line.
        blk := raw[off : off+160]
        for block := 0; block < 5; block++ {
            bo := block * 32
            // For each of the 4 lines in this block
            for line := 0; line < 4; line++ {
                // Each line for 16px width occupies 8 bytes (4 words, one per plane) per ST format.
                p0 := uint16(blk[bo+line*8+0])<<8 | uint16(blk[bo+line*8+1])
                p1 := uint16(blk[bo+line*8+2])<<8 | uint16(blk[bo+line*8+3])
                p2 := uint16(blk[bo+line*8+4])<<8 | uint16(blk[bo+line*8+5])
                p3 := uint16(blk[bo+line*8+6])<<8 | uint16(blk[bo+line*8+7])
                y := block*4 + line
                for bit := 0; bit < 16; bit++ {
                    mask := uint16(1 << (15 - bit))
                    var c uint8
                    if (p0 & mask) != 0 { c |= 1 }
                    if (p1 & mask) != 0 { c |= 2 }
                    if (p2 & mask) != 0 { c |= 4 }
                    if (p3 & mask) != 0 { c |= 8 }
                    x := bit
                    col := pal[c]
                    if c == 0 {
                        col.A = 0 // treat palette index 0 as transparent for tile/path sprites
                    }
                    img.SetRGBA(x, y, col)
                }
            }
        }
        tiles[i] = img
    }
    return tiles, nil
}

// rgbaFromSTScreen decodes a 320xH ST planar framebuffer (height*160 bytes) to RGBA using a 16-color palette.
func rgbaFromSTScreen(raw []byte, width, height int, pal [16]color.RGBA) (*image.RGBA, error) {
    out := image.NewRGBA(image.Rect(0, 0, width, height))
    off := 0
    for y := 0; y < height; y++ {
        line := raw[off : off+160]
        off += 160
        for group := 0; group < 20; group++ {
            p0 := uint16(line[group*8+0])<<8 | uint16(line[group*8+1])
            p1 := uint16(line[group*8+2])<<8 | uint16(line[group*8+3])
            p2 := uint16(line[group*8+4])<<8 | uint16(line[group*8+5])
            p3 := uint16(line[group*8+6])<<8 | uint16(line[group*8+7])
            for bit := 0; bit < 16; bit++ {
                var c uint8
                mask := uint16(1 << (15 - bit))
                if (p0 & mask) != 0 { c |= 1 }
                if (p1 & mask) != 0 { c |= 2 }
                if (p2 & mask) != 0 { c |= 4 }
                if (p3 & mask) != 0 { c |= 8 }
                x := group*16 + bit
                out.SetRGBA(x, y, pal[c])
            }
        }
    }
    return out, nil
}

// BackformDecode implements the assembly routine 'backform': it expands the special packed format
// into standard ST screen-order 4-plane words. Input and output lengths are identical.
func BackformDecode(raw []byte) ([]byte, error) {
    if len(raw)%2 != 0 {
        return nil, fmt.Errorf("odd length input")
    }
    // Number of blocks = len/8, each produces 4 output words.
    blocks := len(raw) / 8
    out := make([]byte, len(raw))
    inPos := 0
    outPos := 0
    for b := 0; b < blocks; b++ {
        var d1, d2, d3, d4 uint16 // plane accumulators
        // 4 input words per block
        for w := 0; w < 4; w++ {
            d7 := uint16(raw[inPos])<<8 | uint16(raw[inPos+1])
            inPos += 2
            // Shift in 16 bits, distributing round-robin across planes using addx semantics.
            for i := 0; i < 16; i++ {
                carry := (d7 & 0x8000) != 0
                d7 <<= 1
                switch i & 3 { // i % 4
                case 0:
                    d1 = (d1 << 1) | b2u(carry)
                case 1:
                    d2 = (d2 << 1) | b2u(carry)
                case 2:
                    d3 = (d3 << 1) | b2u(carry)
                case 3:
                    d4 = (d4 << 1) | b2u(carry)
                }
            }
        }
        // Store words: D4, D3, D2, D1
        out[outPos+0] = byte(d4 >> 8)
        out[outPos+1] = byte(d4)
        out[outPos+2] = byte(d3 >> 8)
        out[outPos+3] = byte(d3)
        out[outPos+4] = byte(d2 >> 8)
        out[outPos+5] = byte(d2)
        out[outPos+6] = byte(d1 >> 8)
        out[outPos+7] = byte(d1)
        outPos += 8
    }
    return out, nil
}

func b2u(b bool) uint16 { if b { return 1 } ; return 0 }

// DecodeBackformIMG decodes packed IMG (FONT2.IMG, EISPLA2.IMG) using Backform and returns an RGBA.
func DecodeBackformIMG(path string, width, height int, pal [16]color.RGBA) (*image.RGBA, error) {
    raw, err := os.ReadFile(path)
    if err != nil { return nil, err }
    // After backform, size must equal height*160.
    buf, err := BackformDecode(raw)
    if err != nil { return nil, err }
    if len(buf) != height*160 {
        return nil, fmt.Errorf("backform size mismatch: got %d, want %d", len(buf), height*160)
    }
    return rgbaFromSTScreen(buf, width, height, pal)
}

// DecodeBackformIMGBytes is like DecodeBackformIMG but accepts in-memory packed bytes.
func DecodeBackformIMGBytes(packed []byte, width, height int, pal [16]color.RGBA) (*image.RGBA, error) {
    buf, err := BackformDecode(packed)
    if err != nil { return nil, err }
    if len(buf) != height*160 {
        return nil, fmt.Errorf("backform size mismatch: got %d, want %d", len(buf), height*160)
    }
    return rgbaFromSTScreen(buf, width, height, pal)
}
