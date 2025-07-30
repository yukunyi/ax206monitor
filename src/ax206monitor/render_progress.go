package main

import (
	"image/color"
	"strconv"

	"github.com/fogleman/gg"
)

type ProgressRenderer struct{}

func NewProgressRenderer() *ProgressRenderer {
	return &ProgressRenderer{}
}

func (p *ProgressRenderer) GetType() string {
	return "progress"
}

func (p *ProgressRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}

	value := monitor.GetValue()
	val, ok := p.getFloat64(value.Value)
	if !ok {
		return nil
	}

	maxValue := item.Max
	if maxValue == 0 {
		maxValue = value.Max
		if maxValue == 0 {
			maxValue = 100
		}
	}

	percentage := val / maxValue * 100
	if percentage > 100 {
		percentage = 100
	}

	// Calculate header height
	headerHeight := 0
	if item.GetShowHeader() {
		headerHeight = 20 // Reserve space for header
		if item.FontSize > 0 {
			headerHeight = item.FontSize + 4
		}
	}

	// Draw background
	bgColor := "#404040"
	if color, exists := config.Colors["progress_background"]; exists {
		bgColor = color
	}
	dc.SetColor(parseColor(bgColor))
	dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height))
	dc.Fill()

	// Draw header background if enabled
	if item.GetShowHeader() {
		dc.SetColor(color.RGBA{20, 20, 20, 255})
		dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(headerHeight))
		dc.Fill()
	}

	// Calculate progress bar area (excluding header)
	progressY := item.Y + headerHeight
	progressHeight := item.Height - headerHeight

	if percentage > 0 {
		fillWidth := float64(item.Width) * percentage / 100
		if fillWidth < 1 {
			fillWidth = 1
		}

		itemColor := item.Color
		if itemColor == "" {
			// Use dynamic color based on percentage value
			itemColor = config.GetDynamicColor(item.Monitor, val)
		}
		dc.SetColor(parseColor(itemColor))
		dc.DrawRectangle(float64(item.X), float64(progressY), fillWidth, float64(progressHeight))
		dc.Fill()
	}

	// Draw border
	dc.SetColor(color.RGBA{80, 80, 80, 255})
	dc.SetLineWidth(1)
	dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height))
	dc.Stroke()

	// Draw header content if enabled
	if item.GetShowHeader() && monitor != nil {
		p.drawHeader(dc, item, monitor, fontCache, config, headerHeight)
	}

	return nil
}

func (p *ProgressRenderer) getFloat64(value interface{}) (float64, bool) {
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

func (p *ProgressRenderer) getColorFromConfig(monitorName string, config *MonitorConfig) string {
	// Check for specific monitor color first
	if color, exists := config.Colors[monitorName]; exists {
		return color
	}

	// Use default progress fill color
	if color, exists := config.Colors["progress_fill"]; exists {
		return color
	}

	return "#4ecdc4"
}

func (p *ProgressRenderer) drawHeader(dc *gg.Context, item *ItemConfig, monitor MonitorItem, fontCache *FontCache, config *MonitorConfig, headerHeight int) {
	fontSize := config.GetSmallFontSize()
	if item.FontSize > 0 {
		fontSize = item.FontSize
	}

	font, err := fontCache.GetFont(fontSize)
	if err != nil || font == nil {
		return
	}

	dc.SetFontFace(font)

	// Draw label on the left
	if item.GetShowLabel() {
		label := monitor.GetLabel()
		if label != "" {
			dc.SetColor(parseColor(config.Colors["default_text"]))
			labelY := float64(item.Y + headerHeight/2 + fontSize/2)
			dc.DrawString(label, float64(item.X+4), labelY)
		}
	}

	// Draw current value on the right
	value := monitor.GetValue()
	if value != nil {
		valueText := p.formatValue(value, item.GetShowUnit())
		if valueText != "" {
			dc.SetColor(parseColor(config.Colors["default_text"]))

			// Measure text to position it on the right
			textWidth, _ := dc.MeasureString(valueText)
			valueX := float64(item.X+item.Width) - textWidth - 4
			valueY := float64(item.Y + headerHeight/2 + fontSize/2)

			dc.DrawString(valueText, valueX, valueY)
		}
	}
}

func (p *ProgressRenderer) formatValue(value *MonitorValue, showUnit bool) string {
	return FormatMonitorValue(value, showUnit, "")
}
