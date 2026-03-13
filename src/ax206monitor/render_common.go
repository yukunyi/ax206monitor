package main

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

func parseColor(hexColor string) color.Color {
	raw := strings.TrimSpace(hexColor)
	if raw == "" {
		return color.RGBA{255, 255, 255, 255}
	}

	if strings.HasPrefix(strings.ToLower(raw), "rgba(") && strings.HasSuffix(raw, ")") {
		content := strings.TrimSpace(raw[5 : len(raw)-1])
		parts := strings.Split(content, ",")
		if len(parts) == 4 {
			r, errR := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			g, errG := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			b, errB := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			a, errA := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
			if errR == nil && errG == nil && errB == nil && errA == nil {
				if a < 0 {
					a = 0
				}
				if a > 1 {
					a = 1
				}
				return color.RGBA{
					R: uint8(clampFloat64(r, 0, 255)),
					G: uint8(clampFloat64(g, 0, 255)),
					B: uint8(clampFloat64(b, 0, 255)),
					A: uint8(a * 255),
				}
			}
		}
	}

	if raw[0] == '#' {
		raw = raw[1:]
	}

	switch len(raw) {
	case 3:
		raw = strings.Repeat(string(raw[0]), 2) + strings.Repeat(string(raw[1]), 2) + strings.Repeat(string(raw[2]), 2)
	case 4:
		raw = strings.Repeat(string(raw[0]), 2) + strings.Repeat(string(raw[1]), 2) + strings.Repeat(string(raw[2]), 2) + strings.Repeat(string(raw[3]), 2)
	}

	if len(raw) != 6 && len(raw) != 8 {
		return color.RGBA{255, 255, 255, 255}
	}

	r, errR := strconv.ParseUint(raw[0:2], 16, 8)
	g, errG := strconv.ParseUint(raw[2:4], 16, 8)
	b, errB := strconv.ParseUint(raw[4:6], 16, 8)
	if errR != nil || errG != nil || errB != nil {
		return color.RGBA{255, 255, 255, 255}
	}
	a := uint8(255)
	if len(raw) == 8 {
		alpha, errA := strconv.ParseUint(raw[6:8], 16, 8)
		if errA != nil {
			return color.RGBA{255, 255, 255, 255}
		}
		a = uint8(alpha)
	}

	return color.RGBA{uint8(r), uint8(g), uint8(b), a}
}

func clampFloat64(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func tryGetFloat64(value interface{}) (float64, bool) {
	switch val := value.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func drawRoundedBackground(dc *gg.Context, x, y, width, height int, bgColor string, radius float64) {
	if bgColor == "" {
		return
	}
	if radius < 0 {
		radius = 0
	}
	dc.SetColor(parseColor(bgColor))
	if radius > 0 {
		dc.DrawRoundedRectangle(float64(x), float64(y), float64(width), float64(height), radius)
	} else {
		dc.DrawRectangle(float64(x), float64(y), float64(width), float64(height))
	}
	dc.Fill()
}

func drawCenteredText(dc *gg.Context, text string, x, y, width, height int, fontSize int, textColor string, fontCache *FontCache) {
	if text == "" {
		return
	}

	face := resolveFontFace(fontCache, fontSize)
	centerX := float64(x) + float64(width)/2
	centerY := float64(y) + float64(height)/2
	dc.SetColor(parseColor(textColor))
	drawMetricAnchoredText(dc, face, text, centerX, centerY, 0.5)
}

func resolveFontFace(fontCache *FontCache, fontSize int) font.Face {
	if fontCache == nil {
		return basicfont.Face7x13
	}
	font, err := fontCache.GetFont(fontSize)
	if err != nil {
		if !isNilFontFace(font) {
			return font
		}
		if !isNilFontFace(fontCache.contentFont) {
			return fontCache.contentFont
		}
		return basicfont.Face7x13
	}
	if isNilFontFace(font) {
		if !isNilFontFace(fontCache.contentFont) {
			return fontCache.contentFont
		}
		return basicfont.Face7x13
	}
	return font
}

func drawCenteredValueWithUnit(dc *gg.Context, valueText, unitText string, x, y, width, height int, valueFontSize int, valueColor string, unitFontSize int, unitColor string, fontCache *FontCache) {
	if strings.TrimSpace(valueText) == "" && strings.TrimSpace(unitText) == "" {
		return
	}
	if strings.TrimSpace(unitText) == "" {
		valueFace := resolveFontFace(fontCache, valueFontSize)
		dc.SetColor(parseColor(valueColor))
		drawMetricAnchoredText(dc, valueFace, valueText, float64(x)+float64(width)/2, float64(y)+float64(height)/2, 0.5)
		return
	}

	valueFace := resolveFontFace(fontCache, valueFontSize)
	unitFace := resolveFontFace(fontCache, unitFontSize)

	dc.SetFontFace(valueFace)
	valueWidth, _ := dc.MeasureString(valueText)

	dc.SetFontFace(unitFace)
	unitWidth, _ := dc.MeasureString(unitText)

	gap := 0.0
	if strings.TrimSpace(valueText) != "" {
		gap = 2.0
	}

	totalWidth := valueWidth + gap + unitWidth
	startX := float64(x) + (float64(width)-totalWidth)/2
	centerY := float64(y) + float64(height)/2

	if strings.TrimSpace(valueText) != "" {
		dc.SetColor(parseColor(valueColor))
		drawMetricAnchoredText(dc, valueFace, valueText, startX, centerY, 0)
		startX += valueWidth + gap
	}

	dc.SetColor(parseColor(unitColor))
	drawMetricAnchoredText(dc, unitFace, unitText, startX, centerY, 0)
}

func canUseItemCustomStyle(item *ItemConfig, config *MonitorConfig) bool {
	if item == nil || config == nil {
		return false
	}
	if !config.AllowCustomStyle {
		return false
	}
	return item.CustomStyle
}

func resolveValueFontSize(item *ItemConfig, config *MonitorConfig, fallback int) int {
	if item != nil && item.runtime.prepared && item.runtime.valueFontSize > 0 {
		return item.runtime.valueFontSize
	}
	size := resolveStyleInt(item, config, "value_font_size", 0)
	if size > 0 {
		return size
	}
	if fallback > 0 {
		return fallback
	}
	return 18
}

func resolveTextFontSize(item *ItemConfig, config *MonitorConfig, fallback int) int {
	if item != nil && item.runtime.prepared && item.runtime.textFontSize > 0 {
		return item.runtime.textFontSize
	}
	size := resolveStyleInt(item, config, "text_font_size", 0)
	if size > 0 {
		return size
	}
	if fallback > 0 {
		return fallback
	}
	return 16
}

func resolveUnitFontSize(item *ItemConfig, config *MonitorConfig, fallback int) int {
	if item != nil && item.runtime.prepared && item.runtime.unitFontSize > 0 {
		return item.runtime.unitFontSize
	}
	size := resolveStyleInt(item, config, "unit_font_size", 0)
	if size > 0 {
		return size
	}
	if fallback > 0 {
		return fallback
	}
	return 14
}

func resolveItemBackground(item *ItemConfig, config *MonitorConfig) string {
	if item == nil {
		return ""
	}
	if item.runtime.prepared {
		return item.runtime.background
	}
	bg := strings.TrimSpace(resolveStyleColor(item, config, "bg", "rgba(0,0,0,0)"))
	if bg == "rgba(0,0,0,0)" {
		return ""
	}
	return bg
}

func resolveItemStaticColor(item *ItemConfig, config *MonitorConfig) string {
	if item == nil {
		return "#f8fafc"
	}
	if item.runtime.prepared && strings.TrimSpace(item.runtime.staticColor) != "" {
		return item.runtime.staticColor
	}
	return resolveStyleColor(item, config, "color", "#f8fafc")
}

func resolveUnitOverride(item *ItemConfig) string {
	if item == nil {
		return ""
	}
	unit := strings.TrimSpace(item.Unit)
	if unit == "" || strings.EqualFold(unit, "auto") {
		return ""
	}
	return unit
}

func resolveMonitorColor(item *ItemConfig, monitor *RenderMonitorSnapshot, config *MonitorConfig) string {
	if monitor == nil {
		return resolveItemStaticColor(item, config)
	}
	value := monitor.value
	if value == nil {
		return resolveItemStaticColor(item, config)
	}
	numberValue, ok := tryGetFloat64(value.Value)
	if !ok {
		return resolveItemStaticColor(item, config)
	}

	minValue, maxValue := resolveEffectiveMinMax(item, value, value.Min, value.Max)

	thresholds := effectiveThresholds(item, minValue, maxValue, config)
	colors := effectiveLevelColors(item, config)
	if len(thresholds) == 0 || len(colors) == 0 {
		return resolveItemStaticColor(item, config)
	}

	for idx, threshold := range thresholds {
		if numberValue <= threshold {
			if idx < len(colors) {
				return colors[idx]
			}
			return colors[len(colors)-1]
		}
	}
	return colors[len(colors)-1]
}

func resolveEffectiveMinMax(item *ItemConfig, monitorValue *CollectValue, fallbackMin float64, fallbackMax float64) (float64, float64) {
	minValue := fallbackMin
	maxValue := fallbackMax
	if monitorValue != nil {
		minValue = monitorValue.Min
		maxValue = monitorValue.Max
	}

	if !hasExplicitItemRange(item) && isTemperatureMetric(item, monitorValue) {
		minValue = 0
		maxValue = 100
	}

	if item != nil {
		if item.MinValue != nil {
			minValue = *item.MinValue
		}
		if item.MaxValue != nil {
			maxValue = *item.MaxValue
		}
		if item.Max > 0 {
			maxValue = item.Max
		}
	}

	if maxValue <= minValue {
		maxValue = minValue + 1
	}
	return minValue, maxValue
}

func hasExplicitItemRange(item *ItemConfig) bool {
	if item == nil {
		return false
	}
	if item.MinValue != nil || item.MaxValue != nil {
		return true
	}
	return item.Max > 0
}

func isTemperatureMetric(item *ItemConfig, monitorValue *CollectValue) bool {
	unit := ""
	if item != nil {
		unitOverride := strings.TrimSpace(item.Unit)
		if unitOverride != "" && !strings.EqualFold(unitOverride, "auto") {
			unit = unitOverride
		}
	}
	if unit == "" && monitorValue != nil {
		unit = monitorValue.Unit
	}
	normalizedUnit := strings.ToLower(strings.TrimSpace(unit))
	if strings.Contains(normalizedUnit, "°c") || strings.Contains(normalizedUnit, "℃") {
		return true
	}

	if item == nil {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(item.Monitor))
	return strings.Contains(name, "temp") || strings.Contains(name, "temperature")
}

func resolveUnitColor(item *ItemConfig, config *MonitorConfig, fallback string) string {
	if item != nil {
		if item.runtime.prepared && item.runtime.explicitUnitColor != "" {
			return item.runtime.explicitUnitColor
		}
		if color := strings.TrimSpace(resolveStyleColor(item, config, "unit_color", "")); color != "" {
			return color
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "#f8fafc"
}

func resolveItemBorderWidth(item *ItemConfig, config *MonitorConfig) float64 {
	if item == nil {
		return 0
	}
	if item.runtime.prepared {
		return item.runtime.borderWidth
	}
	width := resolveStyleFloat(item, config, "border_width", 0)
	if width < 0 {
		width = 0
	}
	return width
}

func resolveItemBorderColor(item *ItemConfig, config *MonitorConfig) string {
	if item == nil {
		return "#475569"
	}
	if item.runtime.prepared && strings.TrimSpace(item.runtime.borderColor) != "" {
		return item.runtime.borderColor
	}
	return resolveStyleColor(item, config, "border_color", "#475569")
}

func resolveItemRadius(item *ItemConfig, config *MonitorConfig, fallback int) float64 {
	if item == nil {
		if fallback < 0 {
			return 0
		}
		return float64(fallback)
	}
	if item.runtime.prepared {
		if item.runtime.radius > 0 {
			return item.runtime.radius
		}
		if fallback < 0 {
			return 0
		}
		return float64(fallback)
	}
	radius := resolveStyleInt(item, config, "radius", 0)
	if radius > 0 {
		return float64(radius)
	}
	if fallback < 0 {
		return 0
	}
	return float64(fallback)
}

func resolveItemCardRadius(item *ItemConfig, config *MonitorConfig) float64 {
	if item == nil {
		return 0
	}
	if item.runtime.prepared {
		if item.runtime.hasCardRadius {
			return item.runtime.cardRadius
		}
		return item.runtime.radius
	}
	cardRadius := getItemAttrFloatCfg(item, config, "card_radius", -1)
	if cardRadius < 0 {
		cardRadius = resolveItemRadius(item, config, 0)
	}
	if cardRadius < 0 {
		cardRadius = 0
	}
	return cardRadius
}

func resolveItemHistoryPoints(item *ItemConfig, config *MonitorConfig, fallback int) int {
	points := resolveStyleInt(item, config, "history_points", 0)
	if points > 0 {
		if points < 10 {
			return 10
		}
		return points
	}
	if fallback < 10 {
		return 10
	}
	return fallback
}

func effectiveLevelColors(item *ItemConfig, config *MonitorConfig) []string {
	if item != nil && item.runtime.prepared {
		switch item.Type {
		case itemTypeSimpleChart:
			if len(item.runtime.simpleChart.levelColors) > 0 {
				return item.runtime.simpleChart.levelColors
			}
		case itemTypeFullChart:
			if len(item.runtime.fullChart.levelColors) > 0 {
				return item.runtime.fullChart.levelColors
			}
		}
	}
	colors := resolveStyleLevelColors(item, config)
	return normalizeRenderLevelColors(colors)
}

func effectiveThresholds(item *ItemConfig, minValue, maxValue float64, config *MonitorConfig) []float64 {
	percentages := []float64(nil)
	if item != nil && item.runtime.prepared {
		switch item.Type {
		case itemTypeSimpleChart:
			percentages = item.runtime.simpleChart.thresholdPercents
		case itemTypeFullChart:
			percentages = item.runtime.fullChart.thresholdPercents
		}
	}
	if len(percentages) == 0 {
		percentages = resolveStyleThresholdsPercent(item, config)
	}
	thresholds := normalizeThresholds(percentages, minValue, maxValue)
	if len(thresholds) == 4 {
		return thresholds
	}
	return buildAverageThresholds(minValue, maxValue)
}
