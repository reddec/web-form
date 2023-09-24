package assets

import (
	"embed"
)

// Static stores static assets.
//
//go:embed static
var Static embed.FS

// Views stores dynamic templates for UI pages.
//
//go:embed views
var Views embed.FS
