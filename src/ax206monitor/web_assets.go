package main

import (
	"ax206monitor/webassets"
	"io/fs"
)

func getEmbeddedWebAssetsFS() (fs.FS, error) {
	return webassets.EmbeddedFS()
}
