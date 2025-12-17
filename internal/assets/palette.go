package assets

import "image/color"

// Palette1 returns the ST palette for the top/menu area (palette1 in ASM).
func Palette1() [16]color.RGBA {
    words := []uint16{
        0x0020, 0x0761, 0x0640, 0x0430, 0x0764, 0x0555, 0x0333, 0x0000,
        0x0012, 0x0023, 0x0034, 0x0045, 0x0056, 0x0201, 0x0403, 0x0705,
    }
    var pal [16]color.RGBA
    for i, w := range words { pal[i] = STWordToRGBA(w) }
    return pal
}

// Palette2 returns the ST palette for the ice plate and game area (palette2 in ASM).
func Palette2() [16]color.RGBA {
    words := []uint16{
        0x0020, 0x0070, 0x0300, 0x0533, 0x0333, 0x0000, 0x0124, 0x0225,
        0x0235, 0x0335, 0x0346, 0x0457, 0x0557, 0x0567, 0x0677, 0x0777,
    }
    var pal [16]color.RGBA
    for i, w := range words { pal[i] = STWordToRGBA(w) }
    return pal
}

