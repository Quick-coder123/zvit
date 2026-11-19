// Package assets embeds the game's artwork into the executable so the game
// stays a single file. Missing pictures fall back to code-drawn graphics.
package assets

import "embed"

//go:embed *.png
var FS embed.FS
