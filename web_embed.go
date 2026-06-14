package main

import (
	"embed"
	"io/fs"
)

// webDist holds the built Vue frontend (web/dist). The directory must exist at
// build time with at least one file. `npm run build` populates it; a .gitkeep
// placeholder keeps the embed valid before the first build.
//
//go:embed all:web/dist
var webDist embed.FS

// webFS returns the embedded frontend rooted at web/dist so that "/" maps to
// the SPA's index.html.
func webFS() (fs.FS, error) {
	return fs.Sub(webDist, "web/dist")
}
