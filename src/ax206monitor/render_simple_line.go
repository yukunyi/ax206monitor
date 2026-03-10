package main

import (
	"strings"

	"github.com/fogleman/gg"
)

type SimpleLineRenderer struct{}

func NewSimpleLineRenderer() *SimpleLineRenderer {
	return &SimpleLineRenderer{}
}

func (r *SimpleLineRenderer) GetType() string {
	return itemTypeSimpleLine
}

func (r *SimpleLineRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	_ = registry
	_ = fontCache
	if dc == nil || item == nil {
		return nil
	}

	orientation := strings.ToLower(strings.TrimSpace(getItemAttrStringCfg(item, config, "line_orientation", "horizontal")))
	lineWidth := getItemAttrFloatCfg(item, config, "line_width", 1)
	if lineWidth < 1 {
		lineWidth = 1
	}

	dc.SetColor(parseColor(resolveItemStaticColor(item, config)))
	dc.SetLineWidth(lineWidth)

	centerX := float64(item.X) + float64(item.Width)/2
	centerY := float64(item.Y) + float64(item.Height)/2
	if orientation == "vertical" {
		dc.DrawLine(centerX, float64(item.Y), centerX, float64(item.Y+item.Height))
		dc.Stroke()
		return nil
	}

	dc.DrawLine(float64(item.X), centerY, float64(item.X+item.Width), centerY)
	dc.Stroke()
	return nil
}
