package main

import (
	"math"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type FullGaugeRenderer struct{}

func NewFullGaugeRenderer() *FullGaugeRenderer {
	return &FullGaugeRenderer{}
}

func (r *FullGaugeRenderer) GetType() string {
	return itemTypeFullGauge
}

func (r *FullGaugeRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}
	value := monitor.GetValue()
	if value == nil {
		return nil
	}
	numberValue, ok := tryGetFloat64(value.Value)
	if !ok {
		return nil
	}

	cardRadius := getItemAttrFloatCfg(item, config, "card_radius", -1)
	if cardRadius < 0 {
		cardRadius = resolveItemRadius(item, config, 0)
	}
	if cardRadius < 0 {
		cardRadius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), cardRadius)

	contentPadding := getItemAttrFloatCfg(item, config, "content_padding", 1)
	if contentPadding < 0 {
		contentPadding = 0
	}
	body := fullRect{
		x: float64(item.X) + contentPadding,
		y: float64(item.Y) + contentPadding,
		w: float64(item.Width) - contentPadding*2,
		h: float64(item.Height) - contentPadding*2,
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
	monitor *CollectItem,
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

	if body.w <= 4 || body.h <= 4 {
		return
	}

	cx := body.x + body.w/2
	cy := body.y + body.h/2
	shortSide := math.Min(body.w, body.h)

	thickness := getItemAttrFloatCfg(item, config, "gauge_thickness", 10)
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

	gapDegrees := getItemAttrFloatCfg(item, config, "gauge_gap_degrees", 76)
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

	trackColor := getItemAttrColorCfg(item, config, "track_color", "#1f2937")
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
	label := strings.TrimSpace(getItemAttrStringCfg(item, config, "title", ""))
	if label == "" {
		label = strings.TrimSpace(getItemAttrStringCfg(item, config, "label", ""))
	}
	if label == "" && monitor != nil {
		label = strings.TrimSpace(monitor.GetLabel())
	}
	if label == "" {
		label = strings.TrimSpace(item.Monitor)
	}

	valueSize := getItemAttrIntCfg(item, config, "value_font_size", 0)
	if valueSize <= 0 {
		valueSize = resolveItemFontSize(item, config, 16)
	}
	if valueSize < 8 {
		valueSize = 8
	}
	labelSize := getItemAttrIntCfg(item, config, "label_font_size", 0)
	if labelSize <= 0 {
		labelSize = resolveLabelFontSize(item, config, 12)
	}
	if labelSize < 8 {
		labelSize = 8
	}
	unitSize := resolveUnitFontSize(item, config, valueSize-3)
	if unitSize < 8 {
		unitSize = 8
	}

	textGap := getItemAttrFloatCfg(item, config, "gauge_text_gap", 4)
	if textGap < 0 {
		textGap = 0
	}
	valueFace := resolveFontFace(fontCache, valueSize)
	labelFace := resolveFontFace(fontCache, labelSize)
	unitFace := resolveFontFace(fontCache, unitSize)
	valueHeight := measureFontHeight(valueFace)
	labelHeight := measureFontHeight(labelFace)
	topCenterY := cy - (labelHeight+textGap)/2
	bottomCenterY := cy + (valueHeight+textGap)/2

	topCenterY = clampFloat64(topCenterY, body.y+valueHeight/2, body.y+body.h-labelHeight/2)
	bottomCenterY = clampFloat64(bottomCenterY, body.y+valueHeight/2, body.y+body.h-labelHeight/2)

	valueColor := lineColor
	unitColor := resolveUnitColor(item, config, valueColor)
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
	drawBaseMetricAnchoredText(dc, labelFace, label, cx, bottomCenterY, 0.5)
}

func measureFontHeight(face font.Face) float64 {
	metrics := face.Metrics()
	height := float64((metrics.Ascent + metrics.Descent).Ceil())
	if height <= 0 {
		return 8
	}
	return height
}
