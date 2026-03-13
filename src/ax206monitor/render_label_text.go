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

func (r *LabelTextRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
	monitor, value, ok := frame.AvailableItemValue(item)
	if !ok {
		return nil
	}

	labelText := resolveItemLabelText(item, config)
	if labelText == "" {
		labelText = resolveItemText(item)
	}
	if labelText == "" {
		labelText = strings.TrimSpace(monitor.label)
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
	monitor *RenderMonitorSnapshot,
	labelText string,
	valueText string,
	unitText string,
) {
	paddingX, paddingY := resolveContentPaddingXY(item, config, 3, 3, 2, 0)
	valueFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleValue, 18, 8)
	labelFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleLabel, 16, 8)
	unitFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)

	labelColor := resolveItemStaticColor(item, config)
	valueColor := resolveMonitorColor(item, monitor, config)
	unitColor := resolveUnitColor(item, config, valueColor)
	textTop := float64(item.Y) + paddingY
	textHeight := float64(item.Height) - paddingY*2
	if textHeight < 1 {
		textHeight = 1
		textTop = float64(item.Y)
	}
	centerY := textTop + textHeight/2

	dc.SetColor(parseColor(labelColor))
	drawMetricAnchoredText(dc, labelFace, labelText, float64(item.X)+paddingX, centerY, 0)

	rightX := float64(item.X+item.Width) - paddingX
	if strings.TrimSpace(unitText) == "" {
		dc.SetColor(parseColor(valueColor))
		drawMetricAnchoredText(dc, valueFace, valueText, rightX, centerY, 1)
		return
	}

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
