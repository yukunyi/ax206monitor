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

	labelText := strings.TrimSpace(getItemAttrStringCfg(item, config, "label", ""))
	if labelText == "" {
		labelText = strings.TrimSpace(item.Text)
	}
	if labelText == "" {
		labelText = strings.TrimSpace(monitor.GetLabel())
	}
	if labelText == "" {
		labelText = strings.TrimSpace(item.Monitor)
	}
	valueText, unitText := resolveItemDisplayValueParts(item, monitor, value, config)

	radius := resolveItemRadius(item, config, 0)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	r.renderLabelText1(dc, item, fontCache, config, monitor, labelText, valueText, unitText)

	drawBaseItemBorder(dc, item, config, radius)
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
	padding := float64(getItemAttrIntCfg(item, config, "content_padding", 3))
	if padding < 2 {
		padding = 2
	}

	valueSize := resolveItemFontSize(item, config, 16)
	valueSize = getItemAttrIntCfg(item, config, "value_font_size", valueSize)
	if valueSize < 8 {
		valueSize = 8
	}
	labelSize := getItemAttrIntCfg(item, config, "label_font_size", valueSize-2)
	if labelSize <= 0 {
		labelSize = resolveLabelFontSize(item, config, valueSize-2)
	}
	if labelSize < 8 {
		labelSize = 8
	}
	unitSize := resolveUnitFontSize(item, config, labelSize)
	if unitSize < 8 {
		unitSize = 8
	}

	labelColor := resolveItemStaticColor(item, config)
	valueColor := resolveMonitorColor(item, monitor, config)
	unitColor := resolveUnitColor(item, config, valueColor)
	centerY := float64(item.Y) + float64(item.Height)/2

	labelFace := resolveFontFace(fontCache, labelSize)
	dc.SetColor(parseColor(labelColor))
	drawMetricAnchoredText(dc, labelFace, labelText, float64(item.X)+padding, centerY, 0)

	rightX := float64(item.X+item.Width) - padding
	if strings.TrimSpace(unitText) == "" {
		valueFace := resolveFontFace(fontCache, valueSize)
		dc.SetColor(parseColor(valueColor))
		drawMetricAnchoredText(dc, valueFace, valueText, rightX, centerY, 1)
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

	dc.SetColor(parseColor(valueColor))
	drawMetricAnchoredText(dc, valueFace, valueText, startX, centerY, 0)
	dc.SetColor(parseColor(unitColor))
	drawMetricAnchoredText(dc, unitFace, unitText, startX+valueWidth+gap, centerY, 0)
}
