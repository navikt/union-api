package web

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var rawStaticFS embed.FS

// StaticFS is rooted at the static/ directory, ready to hand to http.FileServer.
var StaticFS = mustSub(rawStaticFS, "static")

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err) // unreachable: "static" is guaranteed to exist at compile time
	}
	return sub
}
