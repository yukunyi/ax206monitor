package main

import (
	"image/color"
	"strconv"

	"github.com/fogleman/gg"
)

// Common rendering utilities to reduce code duplication

// parseColor converts hex color string to color.Color
func parseColor(hexColor string) color.Color {
	if hexColor == "" {
		return color.RGBA{255, 255, 255, 255}
	}

	if hexColor[0] == '#' {
		hexColor = hexColor[1:]
	}

	if len(hexColor) != 6 {
		return color.RGBA{255, 255, 255, 255}
	}

	r, _ := strconv.ParseUint(hexColor[0:2], 16, 8)
	g, _ := strconv.ParseUint(hexColor[2:4], 16, 8)
	b, _ := strconv.ParseUint(hexColor[4:6], 16, 8)

	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// tryGetFloat64 converts interface{} to float64 with success flag
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

// drawRoundedBackground draws a rounded rectangle background
func drawRoundedBackground(dc *gg.Context, x, y, width, height int, bgColor string) {
	if bgColor != "" {
		dc.SetColor(parseColor(bgColor))
		dc.DrawRoundedRectangle(float64(x), float64(y), float64(width), float64(height), 5)
		dc.Fill()
	}
}

// calculateOptimalFontSize calculates the best font size for given text and area
func calculateOptimalFontSize(dc *gg.Context, text string, maxWidth, maxHeight float64,
	fontCache *FontCache, minSize, maxSize int) int {

	for fontSize := maxSize; fontSize >= minSize; fontSize-- {
		if font, err := fontCache.GetFont(fontSize); err == nil {
			dc.SetFontFace(font)
			w, h := dc.MeasureString(text)
			if w <= maxWidth && h <= maxHeight {
				return fontSize
			}
		}
	}
	return minSize
}

// getColorFromConfig gets color from config with fallback
func getColorFromConfig(monitorName, defaultKey, fallback string, config *MonitorConfig) string {
	// Check for specific monitor color first
	if color, exists := config.Colors[monitorName]; exists {
		return color
	}

	// Use default color key
	if color, exists := config.Colors[defaultKey]; exists {
		return color
	}

	return fallback
}

// getDynamicColorFromValue returns color based on monitor value and thresholds
func getDynamicColorFromValue(monitorName string, value interface{}, config *MonitorConfig) string {
	// Check for specific monitor color first (static override)
	if color, exists := config.Colors[monitorName]; exists {
		return color
	}

	// Try to get numeric value for dynamic coloring
	if numValue, ok := tryGetFloat64(value); ok {
		return config.GetDynamicColor(monitorName, numValue)
	}

	// Fallback to default color
	return getColorFromConfig(monitorName, "default_text", "#ffffff", config)
}

// getDynamicColorFromMonitor returns color based on monitor item and thresholds
// This version can handle special cases like network monitors with display values
func getDynamicColorFromMonitor(monitorName string, monitor MonitorItem, config *MonitorConfig) string {
	// Check for specific monitor color first (static override)
	if color, exists := config.Colors[monitorName]; exists {
		return color
	}

	// Special handling for network monitors - use display value for color calculation
	if isNetworkMonitor(monitorName) {
		if netMonitor, ok := monitor.(*NetworkInterfaceMonitor); ok {
			displayValue := netMonitor.GetDisplayValue()
			return config.GetDynamicColorForNetworkSpeed(monitorName, displayValue, netMonitor.GetValue().Unit)
		}
	}

	// Special handling for disk speed monitors - use display value for color calculation
	if isDiskSpeedMonitor(monitorName) {
		value := monitor.GetValue()
		var displayValue float64

		// Try to get display value from disk speed monitors
		if diskReadMonitor, ok := monitor.(*DiskTotalReadSpeedMonitor); ok {
			displayValue = diskReadMonitor.GetDisplayValue()
		} else if diskWriteMonitor, ok := monitor.(*DiskTotalWriteSpeedMonitor); ok {
			displayValue = diskWriteMonitor.GetDisplayValue()
		} else {
			// Fallback to raw value
			if numValue, ok := tryGetFloat64(value.Value); ok {
				displayValue = numValue
			}
		}

		return config.GetDynamicColorForDiskSpeed(monitorName, displayValue, value.Unit)
	}

	// Default handling for other monitors
	value := monitor.GetValue()
	if numValue, ok := tryGetFloat64(value.Value); ok {
		return config.GetDynamicColor(monitorName, numValue)
	}

	// Fallback to default color
	return getColorFromConfig(monitorName, "default_text", "#ffffff", config)
}

// drawLabel draws a label at the specified position
func drawLabel(dc *gg.Context, x, y, fontSize int, label, color string, fontCache *FontCache) {
	if label == "" {
		return
	}

	font, err := fontCache.GetFont(fontSize)
	if err != nil {
		return
	}

	dc.SetFontFace(font)
	dc.SetColor(parseColor(color))
	dc.DrawString(label, float64(x), float64(y))
}

// drawCenteredText draws text centered in the given area
func drawCenteredText(dc *gg.Context, text string, x, y, width, height int,
	fontSize int, color string, fontCache *FontCache) {

	if text == "" {
		return
	}

	font, err := fontCache.GetFont(fontSize)
	if err != nil {
		font = fontCache.contentFont
	}
	dc.SetFontFace(font)
	dc.SetColor(parseColor(color))

	textWidth, textHeight := dc.MeasureString(text)
	centerX := float64(x) + (float64(width)-textWidth)/2
	centerY := float64(y) + (float64(height)+textHeight)/2

	dc.DrawString(text, centerX, centerY)
}
