package main

import "github.com/fogleman/gg"

type RectRenderer struct{}

func NewRectRenderer() *RectRenderer {
	return &RectRenderer{}
}

func (r *RectRenderer) GetType() string {
	return itemTypeSimpleRect
}

func (r *RectRenderer) RequiresMonitor() bool {
	return false
}

func (r *RectRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
	_ = frame
	_ = fontCache
	drawBaseItemFrame(dc, item, config)
	return nil
}
