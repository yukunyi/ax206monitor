package main

import (
	"io/fs"
	"metrics_render_sender/webassets"
)

func getEmbeddedWebAssetsFS() (fs.FS, error) {
	return webassets.EmbeddedFS()
}
