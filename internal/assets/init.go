package assets

import (
	"embed"
	"io/fs"
)

// Static stores static assets.
//
//go:embed static
var Static embed.FS

// Views stores dynamic templates for UI pages.
//
//go:embed views
var Views embed.FS

func InsideViews() fs.FS {
	v, err := fs.Sub(Views, "views")
	if err != nil {
		panic(err)
	}
	return v
}
