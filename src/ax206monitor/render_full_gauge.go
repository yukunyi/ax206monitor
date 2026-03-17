package main

import (
	"math"
	"strings"

	"github.com/fogleman/gg"
)

type FullGaugeRenderer struct{}

func NewFullGaugeRenderer() *FullGaugeRenderer {
	return &FullGaugeRenderer{}
}

func (r *FullGaugeRenderer) GetType() string {
	return itemTypeFullGauge
}

func (r *FullGaugeRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
	monitor, value, ok := frame.AvailableItemValue(item)
	if !ok {
		return nil
	}
	numberValue, ok := tryGetFloat64(value.Value)
	if !ok {
		return nil
	}

	cardRadius := resolveItemCardRadius(item, config)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), cardRadius)

	contentPaddingX, contentPaddingY := resolveContentPaddingXY(item, config, 1, 1, 0, 0)
	body := fullRect{
		x: float64(item.X) + contentPaddingX,
		y: float64(item.Y) + contentPaddingY,
		w: float64(item.Width) - contentPaddingX*2,
		h: float64(item.Height) - contentPaddingY*2,
	}
	if body.w < 1 {
		body.w = 1
	}
	if body.h < 1 {
		body.h = 1
	}

	r.drawBody(dc, item, monitor, fontCache, value, numberValue, body, config)
	drawBaseItemBorder(dc, item, config, cardRadius)
	return nil
}

func (r *FullGaugeRenderer) drawBody(
	dc *gg.Context,
	item *ItemConfig,
	monitor *RenderMonitorSnapshot,
	fontCache *FontCache,
	value *CollectValue,
	numberValue float64,
	body fullRect,
	config *MonitorConfig,
) {
	minValue, maxValue := resolveEffectiveMinMax(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)
	lineColor := resolveMonitorColor(item, monitor, config)
	textColor := resolveItemStaticColor(item, config)
	thickness := item.runtime.fullGauge.thickness
	gapDegrees := item.runtime.fullGauge.gapDegrees
	trackColor := item.runtime.fullGauge.trackColor
	textGap := item.runtime.fullGauge.textGap
	if !item.runtime.prepared {
		thickness = getItemAttrFloatCfg(item, config, "gauge_thickness", 10)
		gapDegrees = getItemAttrFloatCfg(item, config, "gauge_gap_degrees", 76)
		trackColor = getItemAttrColorCfg(item, config, "track_color", "#1f2937")
		textGap = getItemAttrFloatCfg(item, config, "gauge_text_gap", 1)
	}

	if body.w <= 4 || body.h <= 4 {
		return
	}

	cx := body.x + body.w/2
	cy := body.y + body.h/2
	shortSide := math.Min(body.w, body.h)

	if thickness < 2 {
		thickness = 2
	}
	if thickness > shortSide*0.45 {
		thickness = shortSide * 0.45
	}

	radius := shortSide/2 - thickness/2 - 1
	if radius < 8 {
		return
	}

	if gapDegrees < 20 {
		gapDegrees = 20
	}
	if gapDegrees > 260 {
		gapDegrees = 260
	}
	gapRad := gapDegrees * math.Pi / 180
	sweep := 2*math.Pi - gapRad
	if sweep < 0.2 {
		sweep = 0.2
	}
	start := math.Pi/2 + gapRad/2
	end := start + sweep

	dc.SetLineWidth(thickness)
	dc.SetColor(parseColor(trackColor))
	dc.DrawArc(cx, cy, radius, start, end)
	dc.Stroke()

	dc.SetColor(parseColor(lineColor))
	dc.DrawArc(cx, cy, radius, start, start+sweep*progress)
	dc.Stroke()

	if progress > 0 {
		endAngle := start + sweep*progress
		endX := cx + math.Cos(endAngle)*radius
		endY := cy + math.Sin(endAngle)*radius
		dc.SetColor(parseColor(lineColor))
		dc.DrawCircle(endX, endY, math.Max(1.5, thickness*0.16))
		dc.Fill()
	}

	valueText, unitText := resolveItemDisplayValueParts(item, monitor, value, config)
	label := resolveItemTitleText(item, config)
	if label == "" {
		label = resolveItemLabelText(item, config)
	}
	if label == "" && monitor != nil {
		label = strings.TrimSpace(monitor.label)
	}
	if label == "" {
		label = strings.TrimSpace(item.Monitor)
	}

	valueFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleValue, 18, 8)
	textFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleText, 16, 8)
	unitFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)

	if textGap < 0 {
		textGap = 0
	}
	innerInset := thickness + 2
	textTop := body.y + innerInset
	textBottom := body.y + body.h - innerInset
	if textBottom-textTop < 2 {
		textTop = body.y
		textBottom = body.y + body.h
	}
	availableHeight := textBottom - textTop
	if availableHeight < 2 {
		availableHeight = 2
	}
	if textGap > availableHeight-1 {
		textGap = availableHeight - 1
	}
	if textGap < 0 {
		textGap = 0
	}
	valueMetrics := baseMeasureText(valueFace, valueText)
	unitMetrics := baseMeasureText(unitFace, unitText)
	textMetrics := baseMeasureText(textFace, label)
	valueHeight := math.Max(valueMetrics.ascent+valueMetrics.descent, unitMetrics.ascent+unitMetrics.descent)
	if valueHeight < 1 {
		valueHeight = 1
	}
	textHeight := textMetrics.ascent + textMetrics.descent
	if textHeight < 1 {
		textHeight = 1
	}
	topCenterY := (textTop + textBottom) / 2
	bottomCenterY := topCenterY + valueHeight/2 + textGap + textHeight/2
	if bottomCenterY+textHeight/2 > textBottom {
		shiftUp := bottomCenterY + textHeight/2 - textBottom
		topCenterY -= shiftUp
		bottomCenterY -= shiftUp
	}
	if topCenterY-valueHeight/2 < textTop {
		shiftDown := textTop - (topCenterY - valueHeight/2)
		topCenterY += shiftDown
		bottomCenterY += shiftDown
	}

	valueColor := lineColor
	unitColor := resolveMonitorUnitColor(item, monitor.name, value, numberValue, config)
	dc.SetColor(parseColor(textColor))
	if strings.TrimSpace(unitText) == "" {
		drawBaseMetricAnchoredText(dc, valueFace, valueText, cx, topCenterY, 0.5)
	} else {
		dc.SetFontFace(valueFace)
		valueWidth, _ := dc.MeasureString(valueText)
		dc.SetFontFace(unitFace)
		unitWidth, _ := dc.MeasureString(unitText)
		gap := 2.0
		total := valueWidth + unitWidth
		if strings.TrimSpace(valueText) != "" {
			total += gap
		}
		startX := cx - total/2
		dc.SetColor(parseColor(valueColor))
		drawBaseMetricAnchoredText(dc, valueFace, valueText, startX, topCenterY, 0)
		dc.SetColor(parseColor(unitColor))
		drawBaseMetricAnchoredText(dc, unitFace, unitText, startX+valueWidth+gap, topCenterY, 0)
	}

	dc.SetColor(parseColor(textColor))
	drawBaseMetricAnchoredText(dc, textFace, label, cx, bottomCenterY, 0.5)
}
