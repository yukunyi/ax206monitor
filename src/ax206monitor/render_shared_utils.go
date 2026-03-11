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

type renderHistoryStore struct {
	mu      sync.Mutex
	history map[string][]float64
}

func newRenderHistoryStore() *renderHistoryStore {
	return &renderHistoryStore{
		history: make(map[string][]float64),
	}
}

func (s *renderHistoryStore) append(key string, value float64, maxLen int) []float64 {
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

type fullRect struct {
	x float64
	y float64
	w float64
	h float64
}

func fullHistoryKey(item *ItemConfig, itemType string) string {
	if item != nil && strings.TrimSpace(item.ID) != "" {
		return fmt.Sprintf("%s|id:%s|%s", itemType, strings.TrimSpace(item.ID), strings.TrimSpace(item.Monitor))
	}
	return fmt.Sprintf("%s|%s|%d|%d|%d|%d", itemType, item.Monitor, item.X, item.Y, item.Width, item.Height)
}

func appendRenderHistory(store *renderHistoryStore, item *ItemConfig, itemType string, value float64, defaultPoints int, config *MonitorConfig) []float64 {
	if store == nil {
		return []float64{value}
	}
	points := resolveItemHistoryPoints(item, config, defaultPoints)
	return store.append(fullHistoryKey(item, itemType), value, points)
}

func fullResolveTexts(item *ItemConfig, monitor *CollectItem, value *CollectValue, config *MonitorConfig) (string, string) {
	labelText := strings.TrimSpace(getItemAttrStringCfg(item, config, "title", ""))
	if labelText == "" {
		labelText = strings.TrimSpace(getItemAttrStringCfg(item, config, "label", monitor.GetLabel()))
	}
	if labelText == "" {
		labelText = strings.TrimSpace(item.Monitor)
	}
	valueText, unitText := resolveItemDisplayValueParts(item, monitor, value, config)
	displayValue := strings.TrimSpace(valueText + " " + unitText)
	if displayValue == "" {
		displayValue = strings.TrimSpace(valueText)
	}
	return labelText, displayValue
}

func resolveFullDefaultLabelFontSize(config *MonitorConfig, fallback int) int {
	size := resolveStyleInt(nil, config, "medium_font_size", 0)
	if size > 0 {
		return size
	}
	if fallback > 0 {
		return fallback
	}
	return 14
}

func resolveFullDefaultValueFontSize(config *MonitorConfig, fallback int) int {
	size := resolveStyleInt(nil, config, "large_font_size", 0)
	if size > 0 {
		return size
	}
	if fallback > 0 {
		return fallback
	}
	return 16
}

func resolveFullChartLineColor(item *ItemConfig, config *MonitorConfig) string {
	attrColor := strings.TrimSpace(getItemAttrColorCfg(item, config, "chart_color", ""))
	if attrColor != "" {
		return attrColor
	}
	if item != nil {
		if color := strings.TrimSpace(resolveStyleColor(item, config, "color", "")); color != "" {
			return color
		}
	}
	return "#38bdf8"
}

func resolveChartThresholdColor(value float64, thresholds []float64, colors []string, fallback string) string {
	if len(thresholds) == 0 || len(colors) == 0 {
		return fallback
	}
	for idx, threshold := range thresholds {
		if value <= threshold {
			if idx < len(colors) {
				return colors[idx]
			}
			return colors[len(colors)-1]
		}
	}
	return colors[len(colors)-1]
}

func fullBuildHeaderAndBody(
	item *ItemConfig,
	config *MonitorConfig,
	fontCache *FontCache,
	labelText string,
	displayValue string,
	contentPadding float64,
	defaultBodyGap float64,
) (fullRect, fullRect, font.Face, font.Face) {
	defaultLabelSize := resolveFullDefaultLabelFontSize(config, 10)
	defaultValueSize := resolveFullDefaultValueFontSize(config, 11)
	labelSize := getItemAttrIntCfg(item, config, "title_font_size", 0)
	if labelSize <= 0 {
		labelSize = defaultLabelSize
	}
	if labelSize < 8 {
		labelSize = 8
	}
	valueSize := getItemAttrIntCfg(item, config, "value_font_size", 0)
	if valueSize <= 0 {
		valueSize = defaultValueSize
	}
	if valueSize < 8 {
		valueSize = 8
	}
	labelFace := resolveFontFace(fontCache, labelSize)
	valueFace := resolveFontFace(fontCache, valueSize)

	labelMetrics := baseMeasureText(labelFace, labelText)
	valueMetrics := baseMeasureText(valueFace, displayValue)
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

	bodyGap := getItemAttrFloatCfg(item, config, "body_gap", defaultBodyGap)
	if bodyGap < 0 {
		bodyGap = 0
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

	bodyRect := fullRect{
		x: float64(item.X) + contentPadding,
		y: headerRect.y + headerRect.h + bodyGap,
		w: float64(item.Width) - contentPadding*2,
		h: float64(item.Height) - contentPadding - (headerRect.h + bodyGap),
	}
	if bodyRect.w < 1 {
		bodyRect.w = 1
	}
	if bodyRect.h < 1 {
		bodyRect.h = 1
	}

	return headerRect, bodyRect, labelFace, valueFace
}

func drawFullHeader(
	dc *gg.Context,
	item *ItemConfig,
	config *MonitorConfig,
	rect fullRect,
	labelFace font.Face,
	valueFace font.Face,
	labelText string,
	valueText string,
	labelColor string,
	valueColor string,
	centerOffsetY float64,
) {
	headerCenterY := rect.y + rect.h/2 + centerOffsetY
	const headerHorizontalPadding = 2.0

	dc.SetColor(parseColor(labelColor))
	drawBaseMetricAnchoredText(dc, labelFace, labelText, rect.x+headerHorizontalPadding, headerCenterY, 0)

	dc.SetColor(parseColor(valueColor))
	drawBaseMetricAnchoredText(dc, valueFace, valueText, rect.x+rect.w-headerHorizontalPadding, headerCenterY, 1)

	divider := getItemAttrBoolCfg(item, config, "header_divider", true)
	if divider {
		dc.SetColor(parseColor(getItemAttrColorCfg(item, config, "header_divider_color", "#94a3b840")))
		dividerWidth := getItemAttrFloatCfg(item, config, "header_divider_width", 1)
		if dividerWidth <= 0 {
			dividerWidth = 1
		}
		dc.SetLineWidth(dividerWidth)
		dividerOffset := getItemAttrFloatCfg(item, config, "header_divider_offset", 3)
		if dividerOffset < 0 {
			dividerOffset = 0
		}
		y := rect.y + rect.h + dividerOffset
		dc.DrawLine(rect.x, y, rect.x+rect.w, y)
		dc.Stroke()
	}
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

func baselineForCenteredText(face font.Face, text string, centerY float64) float64 {
	metrics := baseMeasureText(face, text)
	return centerY + (metrics.ascent-metrics.descent)/2
}

func drawMetricAnchoredText(dc *gg.Context, face font.Face, text string, x, centerY, anchorX float64) {
	drawBaseMetricAnchoredText(dc, face, text, x, centerY, anchorX)
}
