package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type fullHistoryStore struct {
	mu      sync.Mutex
	history map[string][]float64
}

func newFullHistoryStore() *fullHistoryStore {
	return &fullHistoryStore{
		history: make(map[string][]float64),
	}
}

func (s *fullHistoryStore) append(key string, value float64, maxLen int) []float64 {
	if maxLen < 10 {
		maxLen = 10
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := append([]float64{}, s.history[key]...)
	if len(current) != maxLen {
		current = resizeChartHistory(current, maxLen)
	}
	if len(current) == 0 {
		current = make([]float64, maxLen)
		for idx := range current {
			current[idx] = math.NaN()
		}
	}
	copy(current, current[1:])
	current[len(current)-1] = value
	s.history[key] = current
	return append([]float64{}, current...)
}

type FullWidgetRenderer struct {
	itemType string
	history  *fullHistoryStore
}

func NewFullWidgetRenderer(itemType string, history *fullHistoryStore) *FullWidgetRenderer {
	return &FullWidgetRenderer{
		itemType: itemType,
		history:  history,
	}
}

func (r *FullWidgetRenderer) GetType() string {
	return r.itemType
}

func (r *FullWidgetRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
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

	cardRadius := float64(item.Radius)
	if cardRadius <= 0 {
		cardRadius = getItemAttrFloat(item, "card_radius", 2)
	}
	if cardRadius < 0 {
		cardRadius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), cardRadius)

	labelText := strings.TrimSpace(getItemAttrString(item, "title", ""))
	if labelText == "" {
		labelText = strings.TrimSpace(getItemAttrString(item, "label", monitor.GetLabel()))
	}
	if labelText == "" {
		labelText = strings.TrimSpace(item.Monitor)
	}
	valueText, unitText := FormatCollectValueParts(value, resolveUnitOverride(item))
	displayValue := strings.TrimSpace(valueText + " " + unitText)
	if displayValue == "" {
		displayValue = strings.TrimSpace(valueText)
	}

	contentPadding := getItemAttrFloat(item, "content_padding", 1)
	if contentPadding < 1 {
		contentPadding = 1
	}

	defaultFontSize := resolveFullDefaultFontSize(config, 10)
	labelSize := getItemAttrInt(item, "title_font_size", 0)
	if labelSize <= 0 {
		labelSize = defaultFontSize
	}
	if labelSize < 8 {
		labelSize = 8
	}
	valueSize := labelSize + 1
	labelFace := resolveFontFace(fontCache, labelSize)
	valueFace := resolveFontFace(fontCache, valueSize)

	labelMetrics := measureTextMetrics(labelFace, labelText)
	valueMetrics := measureTextMetrics(valueFace, displayValue)
	headerTopPadding := 2.0
	headerBottomPadding := 3.0
	headerAscent := math.Max(labelMetrics.ascent, valueMetrics.ascent)
	headerDescent := math.Max(labelMetrics.descent, valueMetrics.descent)
	headerHeight := int(math.Ceil(headerAscent + headerDescent + headerTopPadding + headerBottomPadding))
	if headerHeight < 10 {
		headerHeight = 10
	}
	maxHeader := item.Height / 2
	if maxHeader < 10 {
		maxHeader = 10
	}
	if headerHeight > maxHeader {
		headerHeight = maxHeader
	}

	headerRect := fullRect{
		x: float64(item.X) + contentPadding,
		y: float64(item.Y) + contentPadding,
		w: float64(item.Width) - contentPadding*2,
		h: float64(headerHeight),
	}
	if headerRect.w < 1 {
		headerRect.w = 1
	}
	if headerRect.h < 1 {
		headerRect.h = 1
	}
	headerBaselineY := headerRect.y + headerTopPadding + headerAscent

	bodyRect := fullRect{
		x: float64(item.X) + contentPadding,
		y: headerRect.y + headerRect.h + 4,
		w: float64(item.Width) - contentPadding*2,
		h: float64(item.Height) - contentPadding - (headerRect.h + 4),
	}
	if bodyRect.w < 1 {
		bodyRect.w = 1
	}
	if bodyRect.h < 1 {
		bodyRect.h = 1
	}

	lineColor := resolveMonitorColor(item, monitor, config)
	textColor := resolveItemStaticColor(item, config)
	if strings.TrimSpace(item.Color) == "" && config != nil {
		textColor = config.GetDefaultTextColor()
	}

	r.drawHeader(dc, headerRect, headerBaselineY, labelFace, valueFace, labelText, displayValue, textColor, lineColor, item)
	r.drawByType(dc, item, fontCache, value, numberValue, lineColor, textColor, bodyRect, config)

	drawItemBorder(dc, item)
	return nil
}

func (r *FullWidgetRenderer) drawByType(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect, config *MonitorConfig) {
	switch r.itemType {
	case itemTypeFullChart:
		r.drawFullChart(dc, item, fontCache, value, numberValue, lineColor, body)
	case itemTypeFullProgress:
		r.drawFullProgress(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	case itemTypeFullGauge:
		r.drawFullGauge(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	case itemTypeFullRing:
		r.drawFullRing(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	case itemTypeFullMinMax:
		r.drawFullMinMax(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	case itemTypeFullDelta:
		r.drawFullDelta(dc, item, fontCache, value, numberValue, lineColor, textColor, body, config)
	case itemTypeFullStatus:
		r.drawFullStatus(dc, item, fontCache, value, numberValue, lineColor, textColor, body, config)
	case itemTypeFullMeterH:
		r.drawFullMeterH(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	case itemTypeFullMeterV:
		r.drawFullMeterV(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	case itemTypeFullHeatStrip:
		r.drawFullHeatStrip(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	default:
		r.drawFullProgress(dc, item, fontCache, value, numberValue, lineColor, textColor, body)
	}
}

func (r *FullWidgetRenderer) drawHeader(dc *gg.Context, rect fullRect, baselineY float64, labelFace, valueFace font.Face, labelText string, valueText string, labelColor string, valueColor string, item *ItemConfig) {
	dc.SetFontFace(labelFace)
	dc.SetColor(parseColor(labelColor))
	dc.DrawStringAnchored(labelText, rect.x, baselineY, 0, 0)

	dc.SetFontFace(valueFace)
	dc.SetColor(parseColor(valueColor))
	dc.DrawStringAnchored(valueText, rect.x+rect.w, baselineY, 1, 0)

	divider := getItemAttrBool(item, "header_divider", true)
	if divider {
		dc.SetColor(parseColor(getItemAttrColor(item, "header_divider_color", "#94a3b840")))
		dc.SetLineWidth(1)
		dividerOffset := getItemAttrFloat(item, "header_divider_offset", 3)
		if dividerOffset < 0 {
			dividerOffset = 0
		}
		y := rect.y + rect.h + dividerOffset
		dc.DrawLine(rect.x, y, rect.x+rect.w, y)
		dc.Stroke()
	}
}

func measureFontHeight(face font.Face) float64 {
	metrics := face.Metrics()
	height := float64((metrics.Ascent + metrics.Descent).Ceil())
	if height <= 0 {
		return 8
	}
	return height
}

func measureFontAscent(face font.Face) float64 {
	ascent := float64(face.Metrics().Ascent.Ceil())
	if ascent <= 0 {
		return 6
	}
	return ascent
}

type textMetrics struct {
	ascent  float64
	descent float64
}

func measureTextMetrics(face font.Face, text string) textMetrics {
	metrics := textMetrics{
		ascent:  measureFontAscent(face),
		descent: math.Max(0, measureFontHeight(face)-measureFontAscent(face)),
	}
	bounds, _ := font.BoundString(face, strings.TrimSpace(text))
	minY := float64(bounds.Min.Y) / 64.0
	maxY := float64(bounds.Max.Y) / 64.0
	if minY < 0 {
		metrics.ascent = -minY
	}
	if maxY > 0 {
		metrics.descent = maxY
	}
	return metrics
}

func resolveFullDefaultFontSize(config *MonitorConfig, fallback int) int {
	if config != nil && config.GetDefaultFontSize() > 0 {
		return config.GetDefaultFontSize()
	}
	if fallback > 0 {
		return fallback
	}
	return 14
}

func (r *FullWidgetRenderer) drawFullChart(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, body fullRect) {
	points := getItemAttrInt(item, "history_points", 90)
	history := r.appendHistory(item, numberValue, points)
	if len(history) == 0 {
		return
	}

	minValue, maxValue := resolveRangeValue(item, value, historyMin(history), historyMax(history))
	if maxValue <= minValue {
		maxValue = minValue + 1
	}

	gridLines := getItemAttrInt(item, "grid_lines", 4)
	if gridLines < 2 {
		gridLines = 2
	}
	dc.SetLineWidth(1)
	dc.SetColor(color.RGBA{71, 85, 105, 60})
	for i := 0; i < gridLines; i++ {
		y := body.y + float64(i)*(body.h/float64(gridLines-1))
		dc.DrawLine(body.x, y, body.x+body.w, y)
		dc.Stroke()
	}

	fillArea := getItemAttrBool(item, "fill_area", true)
	lineWidth := getItemAttrFloat(item, "line_width", 2)
	if lineWidth < 1 {
		lineWidth = 1
	}

	firstX := 0.0
	lastX := 0.0
	validPoints := 0
	for idx, histValue := range history {
		if !isFiniteHistoryValue(histValue) {
			continue
		}
		x := body.x
		if len(history) > 1 {
			x = body.x + body.w*float64(idx)/float64(len(history)-1)
		}
		y := body.y + body.h - ((histValue-minValue)/(maxValue-minValue))*body.h
		y = clampFloat64(y, body.y, body.y+body.h)
		if validPoints == 0 {
			firstX = x
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
		lastX = x
		validPoints++
	}
	if validPoints < 2 {
		return
	}

	if fillArea {
		dc.LineTo(lastX, body.y+body.h)
		dc.LineTo(firstX, body.y+body.h)
		dc.ClosePath()
		fill := gg.NewLinearGradient(body.x, body.y, body.x, body.y+body.h)
		fill.AddColorStop(0, parseColor(applyAlpha(lineColor, 0.34)))
		fill.AddColorStop(1, parseColor(applyAlpha(lineColor, 0.02)))
		dc.SetFillStyle(fill)
		dc.FillPreserve()
	}

	dc.SetLineWidth(lineWidth)
	dc.SetColor(parseColor(lineColor))
	dc.Stroke()

	if getItemAttrBool(item, "show_avg_line", true) {
		avg := historyAverage(history)
		y := body.y + body.h - ((avg-minValue)/(maxValue-minValue))*body.h
		dc.SetColor(parseColor(applyAlpha(lineColor, 0.7)))
		dc.SetDash(4, 4)
		dc.SetLineWidth(1)
		dc.DrawLine(body.x, y, body.x+body.w, y)
		dc.Stroke()
		dc.SetDash()

		avgFace := resolveFontFace(fontCache, 10)
		dc.SetFontFace(avgFace)
		dc.DrawStringAnchored("AVG", body.x+body.w-2, y-2, 1, 1)
	}
}

func (r *FullWidgetRenderer) drawFullProgress(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor, textColor string, body fullRect) {
	minValue, maxValue := resolveRangeValue(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	style := strings.ToLower(getItemAttrString(item, "progress_style", "gradient"))
	barRadius := getItemAttrFloat(item, "bar_radius", 8)
	if barRadius < 0 {
		barRadius = 0
	}

	barHeight := getItemAttrFloat(item, "bar_height", math.Min(20, body.h*0.42))
	if barHeight < 8 {
		barHeight = 8
	}
	barY := body.y + (body.h-barHeight)/2
	trackColor := getItemAttrColor(item, "track_color", "#1f2937")
	drawRoundedRectFill(dc, body.x, barY, body.w, barHeight, barRadius, trackColor)

	fillWidth := body.w * progress
	if fillWidth > 1 {
		switch style {
		case "solid":
			drawRoundedRectFill(dc, body.x, barY, fillWidth, barHeight, barRadius, lineColor)
		case "segmented":
			segments := getItemAttrInt(item, "segments", 12)
			if segments < 4 {
				segments = 4
			}
			gap := getItemAttrFloat(item, "segment_gap", 2)
			segW := (body.w - float64(segments-1)*gap) / float64(segments)
			filled := int(math.Round(progress * float64(segments)))
			for i := 0; i < segments; i++ {
				segX := body.x + float64(i)*(segW+gap)
				colorValue := applyAlpha(lineColor, 0.22)
				if i < filled {
					colorValue = lineColor
				}
				drawRoundedRectFill(dc, segX, barY, segW, barHeight, math.Min(barRadius, 3), colorValue)
			}
		case "stripes":
			drawRoundedRectFill(dc, body.x, barY, fillWidth, barHeight, barRadius, lineColor)
			dc.DrawRoundedRectangle(body.x, barY, fillWidth, barHeight, barRadius)
			dc.Clip()
			dc.SetColor(parseColor(applyAlpha("#ffffff", 0.2)))
			for x := body.x - barHeight; x < body.x+fillWidth+barHeight; x += 8 {
				dc.DrawLine(x, barY+barHeight, x+barHeight, barY)
				dc.SetLineWidth(2)
				dc.Stroke()
			}
			dc.ResetClip()
		case "glow":
			drawRoundedRectFill(dc, body.x, barY, fillWidth, barHeight, barRadius, lineColor)
			dc.SetColor(parseColor(applyAlpha(lineColor, 0.4)))
			dc.SetLineWidth(6)
			dc.DrawRoundedRectangle(body.x, barY, fillWidth, barHeight, barRadius)
			dc.Stroke()
		default:
			gradient := gg.NewLinearGradient(body.x, barY, body.x+fillWidth, barY)
			gradient.AddColorStop(0, parseColor(applyAlpha(lineColor, 0.7)))
			gradient.AddColorStop(1, parseColor(lineColor))
			dc.SetFillStyle(gradient)
			dc.DrawRoundedRectangle(body.x, barY, fillWidth, barHeight, barRadius)
			dc.Fill()
		}
	}

	percentText := fmt.Sprintf("%.0f%%", progress*100)
	dc.SetFontFace(resolveFontFace(fontCache, 10))
	dc.SetColor(parseColor(textColor))
	dc.DrawStringAnchored(percentText, body.x+body.w, barY+barHeight+12, 1, 0.5)
}

func (r *FullWidgetRenderer) drawFullGauge(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect) {
	minValue, maxValue := resolveRangeValue(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	cx := body.x + body.w/2
	cy := body.y + body.h*0.95
	radius := math.Min(body.w*0.42, body.h*0.86)
	if radius < 16 {
		radius = 16
	}
	thickness := getItemAttrFloat(item, "gauge_thickness", 9)
	if thickness < 4 {
		thickness = 4
	}

	start := math.Pi
	end := 2 * math.Pi
	dc.SetLineWidth(thickness)
	dc.SetColor(parseColor("#1f2937"))
	dc.DrawArc(cx, cy, radius, start, end)
	dc.Stroke()

	dc.SetColor(parseColor(lineColor))
	dc.DrawArc(cx, cy, radius, start, start+(end-start)*progress)
	dc.Stroke()

	for i := 0; i <= 6; i++ {
		ratio := float64(i) / 6
		angle := start + (end-start)*ratio
		x1 := cx + math.Cos(angle)*(radius-thickness-2)
		y1 := cy + math.Sin(angle)*(radius-thickness-2)
		x2 := cx + math.Cos(angle)*(radius+2)
		y2 := cy + math.Sin(angle)*(radius+2)
		dc.SetLineWidth(1)
		dc.SetColor(parseColor(applyAlpha("#94a3b8", 0.55)))
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}

	label := FormatCollectValue(value, true, resolveUnitOverride(item))
	dc.SetFontFace(resolveFontFace(fontCache, 12))
	dc.SetColor(parseColor(textColor))
	dc.DrawStringAnchored(label, cx, body.y+body.h*0.72, 0.5, 0.5)
}

func (r *FullWidgetRenderer) drawFullRing(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect) {
	minValue, maxValue := resolveRangeValue(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	cx := body.x + body.w/2
	cy := body.y + body.h/2
	radius := math.Min(body.w, body.h) * 0.34
	if radius < 12 {
		radius = 12
	}
	thickness := getItemAttrFloat(item, "ring_thickness", 8)
	if thickness < 4 {
		thickness = 4
	}
	start := -math.Pi / 2
	end := 3 * math.Pi / 2

	dc.SetLineWidth(thickness)
	dc.SetColor(parseColor("#1f2937"))
	dc.DrawArc(cx, cy, radius, start, end)
	dc.Stroke()

	dc.SetColor(parseColor(lineColor))
	dc.DrawArc(cx, cy, radius, start, start+(end-start)*progress)
	dc.Stroke()

	dc.SetColor(parseColor(textColor))
	dc.SetFontFace(resolveFontFace(fontCache, 12))
	dc.DrawStringAnchored(fmt.Sprintf("%.0f%%", progress*100), cx, cy, 0.5, 0.5)
}

func (r *FullWidgetRenderer) drawFullMinMax(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect) {
	history := r.appendHistory(item, numberValue, getItemAttrInt(item, "history_points", 120))
	if len(history) == 0 {
		return
	}

	minVal := historyMin(history)
	maxVal := historyMax(history)
	avgVal := historyAverage(history)
	if maxVal <= minVal {
		maxVal = minVal + 1
	}

	statsY := body.y + 12
	dc.SetFontFace(resolveFontFace(fontCache, 10))
	dc.SetColor(parseColor(applyAlpha(textColor, 0.78)))
	dc.DrawStringAnchored(fmt.Sprintf("Min %.1f", minVal), body.x, statsY, 0, 0.5)
	dc.DrawStringAnchored(fmt.Sprintf("Avg %.1f", avgVal), body.x+body.w/2, statsY, 0.5, 0.5)
	dc.DrawStringAnchored(fmt.Sprintf("Max %.1f", maxVal), body.x+body.w, statsY, 1, 0.5)

	chart := fullRect{x: body.x, y: body.y + 20, w: body.w, h: body.h - 24}
	dc.SetColor(parseColor(applyAlpha("#334155", 0.5)))
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(chart.x, chart.y, chart.w, chart.h, 4)
	dc.Stroke()

	dc.SetColor(parseColor(lineColor))
	dc.SetLineWidth(1.6)
	validPoints := 0
	for idx, histValue := range history {
		if !isFiniteHistoryValue(histValue) {
			continue
		}
		x := chart.x
		if len(history) > 1 {
			x = chart.x + chart.w*float64(idx)/float64(len(history)-1)
		}
		y := chart.y + chart.h - ((histValue-minVal)/(maxVal-minVal))*chart.h
		y = clampFloat64(y, chart.y, chart.y+chart.h)
		if validPoints == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
		validPoints++
	}
	if validPoints < 2 {
		return
	}
	dc.Stroke()
}

func (r *FullWidgetRenderer) drawFullDelta(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect, config *MonitorConfig) {
	history := r.appendHistory(item, numberValue, getItemAttrInt(item, "history_points", 60))
	previous := findPreviousValidHistoryValue(history, numberValue)
	delta := numberValue - previous

	minValue, maxValue := resolveRangeValue(item, value, previous-1, previous+1)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	mainSize := getItemAttrInt(item, "main_font_size", 0)
	if mainSize <= 0 {
		mainSize = resolveFullDefaultFontSize(config, 16)
	}
	if mainSize < 12 {
		mainSize = 12
	}
	deltaColor := lineColor
	deltaSign := "↗"
	if delta < 0 {
		deltaColor = "#ef4444"
		deltaSign = "↘"
	} else if delta == 0 {
		deltaColor = "#94a3b8"
		deltaSign = "→"
	}

	dc.SetColor(parseColor(textColor))
	dc.SetFontFace(resolveFontFace(fontCache, mainSize))
	dc.DrawStringAnchored(FormatCollectValue(value, true, resolveUnitOverride(item)), body.x+2, body.y+body.h*0.35, 0, 0.5)

	dc.SetColor(parseColor(deltaColor))
	dc.SetFontFace(resolveFontFace(fontCache, 12))
	dc.DrawStringAnchored(fmt.Sprintf("%s %.2f", deltaSign, delta), body.x+body.w, body.y+body.h*0.35, 1, 0.5)

	barY := body.y + body.h*0.58
	barH := math.Min(14, body.h*0.35)
	drawRoundedRectFill(dc, body.x, barY, body.w, barH, 6, "#1f2937")
	drawRoundedRectFill(dc, body.x, barY, body.w*progress, barH, 6, lineColor)
}

func (r *FullWidgetRenderer) drawFullStatus(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect, config *MonitorConfig) {
	minValue, maxValue := resolveRangeValue(item, value, 0, 100)
	thresholds := effectiveThresholds(item, minValue, maxValue, config)
	statusText := "NORMAL"
	statusColor := "#22c55e"
	switch findThresholdIndex(numberValue, thresholds) {
	case 0, 1:
		statusText = "NORMAL"
		statusColor = "#22c55e"
	case 2:
		statusText = "WARM"
		statusColor = "#f59e0b"
	case 3:
		statusText = "HOT"
		statusColor = "#f97316"
	default:
		statusText = "CRITICAL"
		statusColor = "#ef4444"
	}

	pillW := math.Min(110, body.w*0.56)
	pillH := 24.0
	pillX := body.x
	pillY := body.y + 4
	drawRoundedRectFill(dc, pillX, pillY, pillW, pillH, 12, applyAlpha(statusColor, 0.2))
	dc.SetColor(parseColor(statusColor))
	dc.SetFontFace(resolveFontFace(fontCache, 11))
	dc.DrawStringAnchored(statusText, pillX+pillW/2, pillY+pillH/2, 0.5, 0.5)

	dc.SetColor(parseColor(textColor))
	dc.SetFontFace(resolveFontFace(fontCache, 12))
	dc.DrawStringAnchored(FormatCollectValue(value, true, resolveUnitOverride(item)), body.x+body.w, pillY+pillH/2, 1, 0.5)

	progress := normalizeRatio(numberValue, minValue, maxValue)
	barY := body.y + body.h - 16
	drawRoundedRectFill(dc, body.x, barY, body.w, 10, 5, "#1f2937")
	drawRoundedRectFill(dc, body.x, barY, body.w*progress, 10, 5, lineColor)
}

func (r *FullWidgetRenderer) drawFullMeterH(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect) {
	minValue, maxValue := resolveRangeValue(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	trackY := body.y + body.h*0.58
	trackH := 10.0
	drawRoundedRectFill(dc, body.x, trackY, body.w, trackH, 5, "#1f2937")

	ticks := getItemAttrInt(item, "ticks", 8)
	if ticks < 3 {
		ticks = 3
	}
	dc.SetLineWidth(1)
	dc.SetColor(parseColor(applyAlpha("#94a3b8", 0.58)))
	for i := 0; i <= ticks; i++ {
		x := body.x + body.w*float64(i)/float64(ticks)
		dc.DrawLine(x, trackY-4, x, trackY+trackH+4)
		dc.Stroke()
	}

	markerX := body.x + body.w*progress
	dc.SetColor(parseColor(lineColor))
	dc.MoveTo(markerX, trackY-7)
	dc.LineTo(markerX-5, trackY-1)
	dc.LineTo(markerX+5, trackY-1)
	dc.ClosePath()
	dc.Fill()

	dc.SetColor(parseColor(textColor))
	dc.SetFontFace(resolveFontFace(fontCache, 11))
	dc.DrawStringAnchored(FormatCollectValue(value, true, resolveUnitOverride(item)), body.x+body.w, body.y+12, 1, 0.5)
}

func (r *FullWidgetRenderer) drawFullMeterV(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect) {
	minValue, maxValue := resolveRangeValue(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	trackW := math.Min(24, body.w*0.28)
	trackX := body.x + body.w/2 - trackW/2
	trackY := body.y + 6
	trackH := body.h - 12
	drawRoundedRectFill(dc, trackX, trackY, trackW, trackH, 6, "#1f2937")

	segments := getItemAttrInt(item, "segments", 12)
	if segments < 4 {
		segments = 4
	}
	gap := 2.0
	segH := (trackH - gap*float64(segments-1)) / float64(segments)
	filled := int(math.Round(progress * float64(segments)))
	for i := 0; i < segments; i++ {
		y := trackY + trackH - float64(i+1)*segH - float64(i)*gap
		colorValue := applyAlpha(lineColor, 0.2)
		if i < filled {
			colorValue = lineColor
		}
		drawRoundedRectFill(dc, trackX+2, y, trackW-4, segH, 2, colorValue)
	}

	dc.SetColor(parseColor(textColor))
	dc.SetFontFace(resolveFontFace(fontCache, 11))
	dc.DrawStringAnchored(FormatCollectValue(value, true, resolveUnitOverride(item)), body.x+body.w/2, body.y+body.h-2, 0.5, 1)
}

func (r *FullWidgetRenderer) drawFullHeatStrip(dc *gg.Context, item *ItemConfig, fontCache *FontCache, value *CollectValue, numberValue float64, lineColor string, textColor string, body fullRect) {
	minValue, maxValue := resolveRangeValue(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	cells := getItemAttrInt(item, "cells", 12)
	if cells < 4 {
		cells = 4
	}
	gap := getItemAttrFloat(item, "cell_gap", 2)
	if gap < 0 {
		gap = 0
	}
	cellW := (body.w - float64(cells-1)*gap) / float64(cells)
	active := int(math.Round(progress * float64(cells)))

	for i := 0; i < cells; i++ {
		x := body.x + float64(i)*(cellW+gap)
		ratio := 0.0
		if cells > 1 {
			ratio = float64(i) / float64(cells-1)
		}
		heatColor := heatGradientColor(ratio)
		if i >= active {
			heatColor = applyAlpha(heatColor, 0.2)
		}
		drawRoundedRectFill(dc, x, body.y+4, cellW, body.h-12, 3, heatColor)
	}

	dc.SetColor(parseColor(textColor))
	dc.SetFontFace(resolveFontFace(fontCache, 11))
	dc.DrawStringAnchored(fmt.Sprintf("%.0f%%", progress*100), body.x+body.w, body.y+body.h-1, 1, 1)
	_ = lineColor
	_ = value
}

func (r *FullWidgetRenderer) appendHistory(item *ItemConfig, value float64, defaultPoints int) []float64 {
	if r.history == nil {
		return []float64{value}
	}
	points := getItemAttrInt(item, "history_points", defaultPoints)
	key := fmt.Sprintf("%s|%s|%d|%d|%d|%d", item.Type, item.Monitor, item.X, item.Y, item.Width, item.Height)
	return r.history.append(key, value, points)
}

type fullRect struct {
	x float64
	y float64
	w float64
	h float64
}

func resolveRangeValue(item *ItemConfig, monitorValue *CollectValue, fallbackMin float64, fallbackMax float64) (float64, float64) {
	minValue := fallbackMin
	maxValue := fallbackMax
	if monitorValue != nil {
		minValue = monitorValue.Min
		maxValue = monitorValue.Max
	}
	if item.MinValue != nil {
		minValue = *item.MinValue
	}
	if item.MaxValue != nil {
		maxValue = *item.MaxValue
	}
	if item.Max > 0 {
		maxValue = item.Max
	}
	if maxValue <= minValue {
		maxValue = minValue + 1
	}
	return minValue, maxValue
}

func normalizeRatio(value float64, minValue float64, maxValue float64) float64 {
	if maxValue <= minValue {
		return 0
	}
	progress := (value - minValue) / (maxValue - minValue)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func historyMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	result := 0.0
	valid := false
	for _, value := range values {
		if !isFiniteHistoryValue(value) {
			continue
		}
		if !valid {
			result = value
			valid = true
			continue
		}
		if value < result {
			result = value
		}
	}
	if !valid {
		return 0
	}
	return result
}

func historyMax(values []float64) float64 {
	if len(values) == 0 {
		return 1
	}
	result := 0.0
	valid := false
	for _, value := range values {
		if !isFiniteHistoryValue(value) {
			continue
		}
		if !valid {
			result = value
			valid = true
			continue
		}
		if value > result {
			result = value
		}
	}
	if !valid {
		return 1
	}
	return result
}

func historyAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	count := 0
	for _, value := range values {
		if !isFiniteHistoryValue(value) {
			continue
		}
		sum += value
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func findPreviousValidHistoryValue(values []float64, fallback float64) float64 {
	for idx := len(values) - 2; idx >= 0; idx-- {
		if isFiniteHistoryValue(values[idx]) {
			return values[idx]
		}
	}
	return fallback
}

func applyAlpha(colorText string, alpha float64) string {
	parsed := color.RGBAModel.Convert(parseColor(colorText)).(color.RGBA)
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", parsed.R, parsed.G, parsed.B, uint8(alpha*255))
}

func drawRoundedRectFill(dc *gg.Context, x, y, width, height, radius float64, colorText string) {
	if width <= 0 || height <= 0 {
		return
	}
	if radius < 0 {
		radius = 0
	}
	if radius > height/2 {
		radius = height / 2
	}
	dc.SetColor(parseColor(colorText))
	if radius > 0 {
		dc.DrawRoundedRectangle(x, y, width, height, radius)
	} else {
		dc.DrawRectangle(x, y, width, height)
	}
	dc.Fill()
}

func findThresholdIndex(value float64, thresholds []float64) int {
	if len(thresholds) == 0 {
		return 0
	}
	for idx, threshold := range thresholds {
		if value <= threshold {
			return idx
		}
	}
	return len(thresholds)
}

func heatGradientColor(ratio float64) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	// green -> yellow -> red
	if ratio < 0.5 {
		t := ratio / 0.5
		r := uint8(34 + (245-34)*t)
		g := uint8(197 + (158-197)*t)
		b := uint8(94 - 94*t)
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	t := (ratio - 0.5) / 0.5
	r := uint8(245 + (239-245)*t)
	g := uint8(158 - 90*t)
	b := uint8(0 + 68*t)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}
