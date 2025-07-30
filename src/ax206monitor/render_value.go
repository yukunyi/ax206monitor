package main

import (
	"image/color"
	"strconv"

	"github.com/fogleman/gg"
)

type ValueRenderer struct{}

func NewValueRenderer() *ValueRenderer {
	return &ValueRenderer{}
}

func (v *ValueRenderer) GetType() string {
	return "value"
}

func (v *ValueRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}

	if item.Background != "" {
		dc.SetColor(parseColor(item.Background))
		dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height), 5)
		dc.Fill()
	}

	if item.GetShowLabel() {
		v.drawLabel(dc, item, monitor, fontCache, config)
	}

	if item.GetShowValue() {
		value := monitor.GetValue()
		text := FormatMonitorValue(value, item.GetShowUnit(), item.UnitText)

		fontSize := v.calculateFontSize(dc, item, text, fontCache, config)
		font, err := fontCache.GetFont(fontSize)
		if err != nil {
			font = fontCache.contentFont
		}
		dc.SetFontFace(font)

		itemColor := item.Color
		if itemColor == "" {
			itemColor = v.getDynamicColorFromValue(item.Monitor, value.Value, config)
		}
		dc.SetColor(parseColor(itemColor))

		textWidth, textHeight := dc.MeasureString(text)
		x := float64(item.X) + (float64(item.Width)-textWidth)/2
		y := float64(item.Y) + (float64(item.Height)+textHeight)/2

		dc.DrawString(text, x, y)
	}

	return nil
}

func (v *ValueRenderer) getFloat64(value interface{}) (float64, bool) {
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

func (v *ValueRenderer) getColorFromConfig(monitorName string, config *MonitorConfig) string {
	// Check for specific monitor color first
	if color, exists := config.Colors[monitorName]; exists {
		return color
	}

	// Use default text color
	if color, exists := config.Colors["default_text"]; exists {
		return color
	}

	return "#ffffff"
}

// getDynamicColorFromValue returns color based on monitor value and thresholds
func (v *ValueRenderer) getDynamicColorFromValue(monitorName string, value interface{}, config *MonitorConfig) string {
	// Check for specific monitor color first (static override)
	if color, exists := config.Colors[monitorName]; exists {
		return color
	}

	// Try to get numeric value for dynamic coloring
	if numValue, ok := v.getFloat64(value); ok {
		return config.GetDynamicColor(monitorName, numValue)
	}

	// Fallback to default color
	return v.getColorFromConfig(monitorName, config)
}

func (v *ValueRenderer) drawLabel(dc *gg.Context, item *ItemConfig, monitor MonitorItem, fontCache *FontCache, config *MonitorConfig) {
	label := config.GetLabelText(monitor.GetName(), monitor.GetLabel())
	if label == "" {
		return
	}

	fontSize := config.GetSmallFontSize()
	if item.FontSize > 0 {
		fontSize = item.FontSize
	}

	font, err := fontCache.GetFont(fontSize)
	if err != nil {
		return
	}

	dc.SetFontFace(font)
	dc.SetColor(parseColor(config.Colors["default_text"]))

	dc.DrawString(label, float64(item.X+4), float64(item.Y+fontSize+4))
}

func (v *ValueRenderer) calculateFontSize(dc *gg.Context, item *ItemConfig, text string, fontCache *FontCache, config *MonitorConfig) int {
	if item.ValueFontSize > 0 {
		return item.ValueFontSize
	}
	if item.FontSize > 0 {
		return item.FontSize
	}

	maxWidth := float64(item.Width) * 0.9
	maxHeight := float64(item.Height) * 0.6

	for fontSize := 20; fontSize >= 8; fontSize-- {
		if font, err := fontCache.GetFont(fontSize); err == nil {
			dc.SetFontFace(font)
			w, h := dc.MeasureString(text)
			if w <= maxWidth && h <= maxHeight {
				return fontSize
			}
		}
	}
	return 8
}

func parseColor(colorStr string) color.Color {
	if colorStr == "" {
		return color.RGBA{255, 255, 255, 255}
	}

	if colorStr[0] == '#' && len(colorStr) == 7 {
		r, _ := strconv.ParseUint(colorStr[1:3], 16, 8)
		g, _ := strconv.ParseUint(colorStr[3:5], 16, 8)
		b, _ := strconv.ParseUint(colorStr[5:7], 16, 8)
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}

	return color.RGBA{255, 255, 255, 255}
}
