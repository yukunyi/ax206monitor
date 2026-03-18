package main

import (
	"metricsrendersender/webassets"
	"io/fs"
)

func getEmbeddedWebAssetsFS() (fs.FS, error) {
	return webassets.EmbeddedFS()
}
