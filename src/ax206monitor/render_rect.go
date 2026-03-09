package main

import "github.com/fogleman/gg"

type RectRenderer struct{}

func NewRectRenderer() *RectRenderer {
	return &RectRenderer{}
}

func (r *RectRenderer) GetType() string {
	return itemTypeSimpleRect
}

func (r *RectRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	_ = registry
	_ = fontCache
	drawBaseItemFrame(dc, item, config)
	return nil
}
