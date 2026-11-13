// Package frontend embeds the built React app.
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist is the vite build output.
var Dist, _ = fs.Sub(dist, "dist")
