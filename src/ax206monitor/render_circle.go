package main

import "github.com/fogleman/gg"

type CircleRenderer struct{}

func NewCircleRenderer() *CircleRenderer {
	return &CircleRenderer{}
}

func (r *CircleRenderer) GetType() string {
	return itemTypeSimpleCircle
}

func (r *CircleRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	_ = registry
	_ = fontCache

	cx := float64(item.X) + float64(item.Width)/2
	cy := float64(item.Y) + float64(item.Height)/2
	rx := float64(item.Width) / 2
	ry := float64(item.Height) / 2

	bgColor := resolveItemBackground(item, config)
	if bgColor != "" {
		dc.SetColor(parseColor(bgColor))
		dc.DrawEllipse(cx, cy, rx, ry)
		dc.Fill()
	}

	borderWidth := resolveItemBorderWidth(item, config)
	if borderWidth > 0 {
		dc.SetColor(parseColor(resolveItemBorderColor(item, config)))
		dc.SetLineWidth(borderWidth)
		dc.DrawEllipse(cx, cy, rx, ry)
		dc.Stroke()
	}
	return nil
}
