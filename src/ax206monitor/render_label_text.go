package main

import (
	"strings"

	"github.com/fogleman/gg"
)

type LabelTextRenderer struct {
	itemType string
}

func NewLabelTextRenderer(itemType string) *LabelTextRenderer {
	return &LabelTextRenderer{itemType: itemType}
}

func (r *LabelTextRenderer) GetType() string {
	return r.itemType
}

func (r *LabelTextRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}
	value := monitor.GetValue()
	if value == nil {
		return nil
	}

	labelText := strings.TrimSpace(getItemAttrString(item, "label", ""))
	if labelText == "" {
		labelText = strings.TrimSpace(item.Text)
	}
	if labelText == "" {
		labelText = strings.TrimSpace(monitor.GetLabel())
	}
	if labelText == "" {
		labelText = strings.TrimSpace(item.Monitor)
	}
	valueText, unitText := FormatCollectValueParts(value, resolveUnitOverride(item))

	radius := float64(item.Radius)
	if radius < 0 {
		radius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	switch r.itemType {
	case itemTypeLabelText2:
		r.renderLabelText2(dc, item, fontCache, config, monitor, labelText, valueText, unitText)
	default:
		r.renderLabelText1(dc, item, fontCache, config, monitor, labelText, valueText, unitText)
	}

	drawItemBorder(dc, item)
	return nil
}

func (r *LabelTextRenderer) renderLabelText1(
	dc *gg.Context,
	item *ItemConfig,
	fontCache *FontCache,
	config *MonitorConfig,
	monitor *CollectItem,
	labelText string,
	valueText string,
	unitText string,
) {
	padding := float64(getItemAttrInt(item, "content_padding", 6))
	if padding < 2 {
		padding = 2
	}

	valueSize := resolveItemFontSize(item, config, 20)
	valueSize = getItemAttrInt(item, "value_font_size", valueSize)
	if valueSize < 8 {
		valueSize = 8
	}
	labelSize := getItemAttrInt(item, "label_font_size", valueSize-2)
	if labelSize < 8 {
		labelSize = 8
	}
	unitSize := resolveUnitFontSize(item, config, labelSize)
	if unitSize < 8 {
		unitSize = 8
	}

	labelColor := resolveItemStaticColor(item, config)
	valueColor := resolveMonitorColor(item, monitor, config)
	unitColor := resolveUnitColor(item, valueColor)
	centerY := float64(item.Y) + float64(item.Height)/2

	labelFace := resolveFontFace(fontCache, labelSize)
	dc.SetFontFace(labelFace)
	dc.SetColor(parseColor(labelColor))
	dc.DrawStringAnchored(labelText, float64(item.X)+padding, centerY, 0, 0.5)

	rightX := float64(item.X+item.Width) - padding
	if strings.TrimSpace(unitText) == "" {
		valueFace := resolveFontFace(fontCache, valueSize)
		dc.SetFontFace(valueFace)
		dc.SetColor(parseColor(valueColor))
		dc.DrawStringAnchored(valueText, rightX, centerY, 1, 0.5)
		return
	}

	valueFace := resolveFontFace(fontCache, valueSize)
	unitFace := resolveFontFace(fontCache, unitSize)
	dc.SetFontFace(valueFace)
	valueWidth, _ := dc.MeasureString(valueText)
	dc.SetFontFace(unitFace)
	unitWidth, _ := dc.MeasureString(unitText)
	gap := 2.0
	startX := rightX - (valueWidth + gap + unitWidth)

	dc.SetFontFace(valueFace)
	dc.SetColor(parseColor(valueColor))
	dc.DrawStringAnchored(valueText, startX, centerY, 0, 0.5)
	dc.SetFontFace(unitFace)
	dc.SetColor(parseColor(unitColor))
	dc.DrawStringAnchored(unitText, startX+valueWidth+gap, centerY, 0, 0.5)
}

func (r *LabelTextRenderer) renderLabelText2(
	dc *gg.Context,
	item *ItemConfig,
	fontCache *FontCache,
	config *MonitorConfig,
	monitor *CollectItem,
	labelText string,
	valueText string,
	unitText string,
) {
	padding := float64(getItemAttrInt(item, "content_padding", 5))
	if padding < 2 {
		padding = 2
	}

	valueSize := resolveItemFontSize(item, config, 24)
	valueSize = getItemAttrInt(item, "value_font_size", valueSize)
	if valueSize < 10 {
		valueSize = 10
	}
	metaSize := getItemAttrInt(item, "meta_font_size", valueSize-2)
	if metaSize < 8 {
		metaSize = 8
	}
	labelColor := resolveItemStaticColor(item, config)
	valueColor := resolveMonitorColor(item, monitor, config)
	unitColor := resolveUnitColor(item, labelColor)

	metaFace := resolveFontFace(fontCache, metaSize)
	valueFace := resolveFontFace(fontCache, valueSize)
	headerY := float64(item.Y) + padding

	dc.SetFontFace(metaFace)
	dc.SetColor(parseColor(labelColor))
	dc.DrawStringAnchored(labelText, float64(item.X)+padding, headerY, 0, 0)

	if strings.TrimSpace(unitText) != "" {
		dc.SetColor(parseColor(unitColor))
		dc.DrawStringAnchored(unitText, float64(item.X+item.Width)-padding, headerY, 1, 0)
	}

	dc.SetFontFace(valueFace)
	dc.SetColor(parseColor(valueColor))
	dc.DrawStringAnchored(
		valueText,
		float64(item.X)+float64(item.Width)/2,
		float64(item.Y)+float64(item.Height)/2+1,
		0.5,
		0.5,
	)
}
