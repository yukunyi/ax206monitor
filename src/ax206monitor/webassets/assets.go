package webassets

import (
	"embed"
	"io/fs"
)

//go:embed webdist/*
var embeddedAssets embed.FS

func EmbeddedFS() (fs.FS, error) {
	return fs.Sub(embeddedAssets, "webdist")
}
