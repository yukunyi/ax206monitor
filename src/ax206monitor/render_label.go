package main

import "github.com/fogleman/gg"

type LabelRenderer struct{}

func NewLabelRenderer() *LabelRenderer {
	return &LabelRenderer{}
}

func (r *LabelRenderer) GetType() string {
	return itemTypeSimpleLabel
}

func (r *LabelRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	_ = registry
	if item.Text == "" {
		return nil
	}

	radius := float64(item.Radius)
	if radius < 0 {
		radius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	fontSize := resolveItemFontSize(item, config, 16)
	font := resolveFontFace(fontCache, fontSize)
	dc.SetFontFace(font)
	dc.SetColor(parseColor(resolveItemStaticColor(item, config)))

	_, textHeight := dc.MeasureString("Ag")
	dc.DrawString(item.Text, float64(item.X+4), float64(item.Y)+4+textHeight)
	drawItemBorder(dc, item)
	return nil
}
