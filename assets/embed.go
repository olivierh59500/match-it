// Package resources exposes the runtime assets from the application binary.
// Embedding them is required on mobile, where the process has no project
// working directory to load files from.
package resources

import "embed"

// Files contains every asset needed by the game at runtime.
//
//go:embed png/tile_*.png png/obenplat.img.png png/eispla2.img.png
//go:embed png/menuplat.img.png png/font2/*.png png/help/*.png
//go:embed png/remove/removean.img.png png/gamearea.img.png
//go:embed png/malakhsoftware-pixel.png music/*.ym
var Files embed.FS
