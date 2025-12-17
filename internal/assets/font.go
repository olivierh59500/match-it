package assets

import (
    "fmt"
    "image"
    "image/color"
)

// LoadFontDigits decodes FONT2.IMG (packed) into 10 digit glyphs (0..9), each 16x15 pixels,
// using transparency for the background (palette index 0).
// Layout: 320x45 (3 rows x 15 px) with 20 cells per row @16 px width.
func LoadFontDigits(fontPacked []byte, pal [16]color.RGBA) ([10]*image.RGBA, error) {
    // Backform to planar screen bytes
    buf, err := BackformDecode(fontPacked)
    if err != nil { return [10]*image.RGBA{}, err }
    if len(buf) != 45*160 {
        return [10]*image.RGBA{}, fmt.Errorf("unexpected font buffer size: %d", len(buf))
    }
    var out [10]*image.RGBA
    // For each digit, extract 16x15 subregion and convert with alpha=0 where color index==0
    for d := 0; d < 10; d++ {
        idx := 27 + d
        col := idx % 20
        row := idx / 20
        if row > 2 { return out, fmt.Errorf("font index out of range: %d", idx) }
        x0 := col * 16
        y0 := row * 15
        sub := image.NewRGBA(image.Rect(0, 0, 16, 15))
        for yy := 0; yy < 15; yy++ {
            // Pointer to the start of this line in the 320px-wide planar buffer
            line := buf[(y0+yy)*160 : (y0+yy+1)*160]
            for group := 0; group < 20; group++ {
                // planes per 16-pixel group
                p0 := uint16(line[group*8+0])<<8 | uint16(line[group*8+1])
                p1 := uint16(line[group*8+2])<<8 | uint16(line[group*8+3])
                p2 := uint16(line[group*8+4])<<8 | uint16(line[group*8+5])
                p3 := uint16(line[group*8+6])<<8 | uint16(line[group*8+7])
                // Only copy pixels for the target column group of this digit
                if group*16+15 < x0 || group*16 >= x0+16 { continue }
                for bit := 0; bit < 16; bit++ {
                    x := group*16 + bit
                    if x < x0 || x >= x0+16 { continue }
                    mask := uint16(1 << (15 - bit))
                    var cidx uint8
                    if (p0 & mask) != 0 { cidx |= 1 }
                    if (p1 & mask) != 0 { cidx |= 2 }
                    if (p2 & mask) != 0 { cidx |= 4 }
                    if (p3 & mask) != 0 { cidx |= 8 }
                    xx := x - x0
                    colr := pal[cidx]
                    if cidx == 0 { colr.A = 0 }
                    sub.SetRGBA(xx, yy, colr)
                }
            }
        }
        out[d] = sub
    }
    return out, nil
}

// LoadFontGlyphs47 decodes all 47 glyph cells used by the game font (A-Z, digits, and some punctuation).
// Returns a slice indexed by the cell index (0..46), each 16x15 RGBA with transparent background.
func LoadFontGlyphs47(fontPacked []byte, pal [16]color.RGBA) ([]*image.RGBA, error) {
    buf, err := BackformDecode(fontPacked)
    if err != nil { return nil, err }
    if len(buf) != 45*160 { return nil, fmt.Errorf("unexpected font buffer size: %d", len(buf)) }
    glyphs := make([]*image.RGBA, 47)
    for idx := 0; idx < 47; idx++ {
        col := idx % 20
        row := idx / 20
        x0 := col * 16
        y0 := row * 15
        sub := image.NewRGBA(image.Rect(0, 0, 16, 15))
        for yy := 0; yy < 15; yy++ {
            line := buf[(y0+yy)*160 : (y0+yy+1)*160]
            for group := x0 / 16; group <= (x0+15)/16; group++ {
                p0 := uint16(line[group*8+0])<<8 | uint16(line[group*8+1])
                p1 := uint16(line[group*8+2])<<8 | uint16(line[group*8+3])
                p2 := uint16(line[group*8+4])<<8 | uint16(line[group*8+5])
                p3 := uint16(line[group*8+6])<<8 | uint16(line[group*8+7])
                for bit := 0; bit < 16; bit++ {
                    x := group*16 + bit
                    if x < x0 || x >= x0+16 { continue }
                    mask := uint16(1 << (15 - bit))
                    var cidx uint8
                    if (p0 & mask) != 0 { cidx |= 1 }
                    if (p1 & mask) != 0 { cidx |= 2 }
                    if (p2 & mask) != 0 { cidx |= 4 }
                    if (p3 & mask) != 0 { cidx |= 8 }
                    xx := x - x0
                    if cidx == 0 {
                        sub.SetRGBA(xx, yy, color.RGBA{0, 0, 0, 0})
                    } else {
                        colr := pal[cidx]
                        colr.A = 255
                        sub.SetRGBA(xx, yy, colr)
                    }
                }
            }
        }
        glyphs[idx] = sub
    }
    return glyphs, nil
}

// FontIndexForChar maps a rune to the font cell index as per fonttabelle in the original code.
// Supports A-Z, 0-9, space, and '!'. Returns -1 if unsupported.
func FontIndexForChar(r rune) int {
    switch {
    case r == ' ':
        return -1
    case r >= 'A' && r <= 'Z':
        return int(r - 'A') // 0..25
    case r >= '0' && r <= '9':
        return 27 + int(r-'0') // 27..36
    case r == '!':
        return 26
    default:
        return -1
    }
}
