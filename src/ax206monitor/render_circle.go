package main

import "github.com/fogleman/gg"

type CircleRenderer struct{}

func NewCircleRenderer() *CircleRenderer {
	return &CircleRenderer{}
}

func (r *CircleRenderer) GetType() string {
	return "circle"
}

func (r *CircleRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
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

	if item.BorderWidth > 0 {
		borderColor := item.BorderColor
		if borderColor == "" {
			borderColor = "#475569"
		}
		dc.SetColor(parseColor(borderColor))
		dc.SetLineWidth(item.BorderWidth)
		dc.DrawEllipse(cx, cy, rx, ry)
		dc.Stroke()
	}
	return nil
}
