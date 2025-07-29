package main

import (
	"image/color"

	"github.com/fogleman/gg"
)

type RectRenderer struct{}

func NewRectRenderer() *RectRenderer {
	return &RectRenderer{}
}

func (r *RectRenderer) GetType() string {
	return "rect"
}

func (r *RectRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
	// Draw background
	if item.Background != "" {
		dc.SetColor(parseColor(item.Background))
		dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height), 5)
		dc.Fill()
	}

	// Draw border
	dc.SetColor(color.RGBA{80, 80, 80, 255})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height), 5)
	dc.Stroke()

	return nil
}
