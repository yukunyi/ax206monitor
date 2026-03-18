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

	textText := resolveItemLabelText(item, config)
	if textText == "" {
		textText = resolveItemText(item)
	}
	if textText == "" {
		textText = strings.TrimSpace(monitor.label)
	}
	if textText == "" {
		textText = strings.TrimSpace(item.Monitor)
	}
	valueText, unitText := resolveItemDisplayValueParts(item, monitor, value, config)

	radius := resolveItemRadius(item, config, 0)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	r.renderLabelText1(dc, item, fontCache, config, monitor, textText, valueText, unitText)

	drawBaseItemBorder(dc, item, config, radius)
	return nil
}

func (r *LabelTextRenderer) renderLabelText1(
	dc *gg.Context,
	item *ItemConfig,
	fontCache *FontCache,
	config *MonitorConfig,
	monitor *RenderMonitorSnapshot,
	textText string,
	valueText string,
	unitText string,
) {
	paddingX, paddingY := resolveContentPaddingXY(item, config, 3, 3, 2, 0)
	valueFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleValue, 18, 8)
	textFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleText, 16, 8)
	unitFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)

	textColor := resolveItemStaticColor(item, config)
	valueColor := resolveMonitorColor(item, monitor, config)
	numberValue := 0.0
	if monitor != nil && monitor.value != nil {
		numberValue, _ = tryGetFloat64(monitor.value.Value)
	}
	unitColor := resolveMonitorUnitColor(item, monitor.name, monitor.value, numberValue, config)
	textTop := float64(item.Y) + paddingY
	textHeight := float64(item.Height) - paddingY*2
	if textHeight < 1 {
		textHeight = 1
		textTop = float64(item.Y)
	}
	centerY := textTop + textHeight/2

	dc.SetColor(parseColor(textColor))
	drawMetricAnchoredText(dc, textFace, textText, float64(item.X)+paddingX, centerY, 0)

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
