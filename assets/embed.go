// Package resources exposes the runtime assets from the application binary.
// Embedding them is required on mobile, where the process has no project
// working directory to load files from.
package resources

import "embed"

// Files contains every asset needed by the game at runtime.
//
//go:embed png music
var Files embed.FS
