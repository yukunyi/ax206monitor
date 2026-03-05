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

	radius := float64(item.Radius)
	if radius < 0 {
		radius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)
	drawItemBorder(dc, item)
	return nil
}
