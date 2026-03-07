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

func drawItemBorder(dc *gg.Context, item *ItemConfig) {
	if item.BorderWidth <= 0 {
		return
	}
	borderColor := item.BorderColor
	if borderColor == "" {
		borderColor = "#475569"
	}
	radius := float64(item.Radius)
	if radius < 0 {
		radius = 0
	}
	dc.SetColor(parseColor(borderColor))
	dc.SetLineWidth(item.BorderWidth)
	if radius > 0 {
		dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height), radius)
	} else {
		dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height))
	}
	dc.Stroke()
}

func drawCenteredText(dc *gg.Context, text string, x, y, width, height int, fontSize int, textColor string, fontCache *FontCache) {
	if text == "" {
		return
	}

	font := resolveFontFace(fontCache, fontSize)
	dc.SetFontFace(font)
	dc.SetColor(parseColor(textColor))

	textWidth, textHeight := dc.MeasureString(text)
	centerX := float64(x) + (float64(width)-textWidth)/2
	centerY := float64(y) + (float64(height)+textHeight)/2
	dc.DrawString(text, centerX, centerY)
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
		drawCenteredText(dc, valueText, x, y, width, height, valueFontSize, valueColor, fontCache)
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
		dc.SetFontFace(valueFace)
		dc.SetColor(parseColor(valueColor))
		dc.DrawStringAnchored(valueText, startX, centerY, 0, 0.5)
		startX += valueWidth + gap
	}

	dc.SetFontFace(unitFace)
	dc.SetColor(parseColor(unitColor))
	dc.DrawStringAnchored(unitText, startX, centerY, 0, 0.5)
}

func resolveItemFontSize(item *ItemConfig, config *MonitorConfig, fallback int) int {
	if item.FontSize > 0 {
		return item.FontSize
	}
	if config != nil && config.GetDefaultFontSize() > 0 {
		return config.GetDefaultFontSize()
	}
	if fallback > 0 {
		return fallback
	}
	return 14
}

func resolveUnitFontSize(item *ItemConfig, config *MonitorConfig, fallback int) int {
	if item.UnitFontSize > 0 {
		return item.UnitFontSize
	}
	return resolveItemFontSize(item, config, fallback)
}

func resolveItemBackground(item *ItemConfig, config *MonitorConfig) string {
	if strings.TrimSpace(item.Background) != "" {
		return strings.TrimSpace(item.Background)
	}
	if item != nil && isShapeItemType(item.Type) {
		return "#33415566"
	}
	if item != nil && isFullItemType(item.Type) {
		return "#111827c8"
	}
	_ = config
	return ""
}

func resolveItemStaticColor(item *ItemConfig, config *MonitorConfig) string {
	if strings.TrimSpace(item.Color) != "" {
		return strings.TrimSpace(item.Color)
	}
	if config != nil {
		return config.GetDefaultTextColor()
	}
	return "#f8fafc"
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

func resolveMonitorColor(item *ItemConfig, monitor *CollectItem, config *MonitorConfig) string {
	if strings.TrimSpace(item.Color) != "" {
		return strings.TrimSpace(item.Color)
	}
	if monitor == nil {
		return resolveItemStaticColor(item, config)
	}
	value := monitor.GetValue()
	if value == nil {
		return resolveItemStaticColor(item, config)
	}
	numberValue, ok := tryGetFloat64(value.Value)
	if !ok {
		return resolveItemStaticColor(item, config)
	}

	minValue := value.Min
	maxValue := value.Max
	if item.MinValue != nil {
		minValue = *item.MinValue
	}
	if item.MaxValue != nil {
		maxValue = *item.MaxValue
	}
	if item.Max > 0 {
		maxValue = item.Max
	}

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

func resolveUnitColor(item *ItemConfig, fallback string) string {
	if strings.TrimSpace(item.UnitColor) != "" {
		return strings.TrimSpace(item.UnitColor)
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "#f8fafc"
}

func effectiveLevelColors(item *ItemConfig, config *MonitorConfig) []string {
	candidate := make([]string, 0, 4)
	for _, color := range item.LevelColors {
		trimmed := strings.TrimSpace(color)
		if trimmed != "" {
			candidate = append(candidate, trimmed)
		}
	}
	if len(candidate) == 0 && config != nil {
		candidate = append(candidate, config.GetLevelColors()...)
	}
	if len(candidate) == 0 {
		candidate = append(candidate, defaultLevelColors()...)
	}
	for len(candidate) < 4 {
		candidate = append(candidate, candidate[len(candidate)-1])
	}
	if len(candidate) > 4 {
		candidate = candidate[:4]
	}
	return candidate
}

func effectiveThresholds(item *ItemConfig, minValue, maxValue float64, config *MonitorConfig) []float64 {
	thresholds := normalizeThresholds(item.Thresholds, minValue, maxValue)
	if len(thresholds) == 4 {
		return thresholds
	}
	if config != nil {
		thresholds = normalizeThresholds(config.DefaultThresholds, minValue, maxValue)
		if len(thresholds) == 4 {
			return thresholds
		}
	}
	return buildAverageThresholds(minValue, maxValue)
}
