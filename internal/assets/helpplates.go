package assets

import (
    "fmt"
    "image"
    "image/color"
    "os"
)

// LoadHelpPlates decodes HELPPLAT.IMG into six 64x16 sprites, one for each help count (0..5).
// The file is organized as 6 blocks of 512 bytes. Each block is 16 lines, each line 32 bytes (64 pixels).
func LoadHelpPlates(path string, pal [16]color.RGBA) ([]*image.RGBA, error) {
    raw, err := os.ReadFile(path)
    if err != nil { return nil, err }
    if len(raw) != 6*512 {
        return nil, fmt.Errorf("helpplat size %d != 3072", len(raw))
    }
    out := make([]*image.RGBA, 6)
    for n := 0; n < 6; n++ {
        img := image.NewRGBA(image.Rect(0, 0, 64, 16))
        base := n * 512
        for y := 0; y < 16; y++ {
            line := raw[base+y*32 : base+y*32+32]
            // 4 groups of 8 bytes (one 16px group per 8 bytes)
            for group := 0; group < 4; group++ {
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
                    img.SetRGBA(x, y, pal[c])
                }
            }
        }
        out[n] = img
    }
    return out, nil
}

