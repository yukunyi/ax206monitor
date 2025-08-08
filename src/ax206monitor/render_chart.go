package main

import (
	"image/color"

	"github.com/fogleman/gg"
)

type ChartRenderer struct {
	history     map[string][]float64
	historySize int
}

func NewChartRenderer() *ChartRenderer {
	return &ChartRenderer{
		history:     make(map[string][]float64),
		historySize: 60, // Default size
	}
}

func (c *ChartRenderer) GetType() string {
	return "chart"
}

func (c *ChartRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
	if !item.History {
		return nil
	}

	// Update history size from config
	if config.HistorySize > 0 {
		c.historySize = config.HistorySize
	}

	monitor := registry.Get(item.Monitor)
	var val float64

	if monitor == nil || !monitor.IsAvailable() {
		// Use default value 0 for missing records
		val = 0.0
	} else {
		value := monitor.GetValue()
		var ok bool
		val, ok = tryGetFloat64(value.Value)
		if !ok {
			val = 0.0
		}
	}

	c.updateHistory(item.Monitor, val)
	history := c.history[item.Monitor]

	if len(history) < 2 {
		return nil
	}

	// Calculate header height using actual text metrics
	headerHeight := 0
	if item.GetShowHeader() {
		if item.FontSize > 0 {
			headerHeight = int(float32(item.FontSize) * 1.5)
		} else {
			headerHeight = 20
		}
	}

	// Draw background
	dc.SetColor(color.RGBA{30, 30, 30, 255})
	dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height))
	dc.Fill()

	// Draw header background if enabled
	if item.GetShowHeader() {
		dc.SetColor(color.RGBA{20, 20, 20, 255})
		dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(headerHeight))
		dc.Fill()

		// Draw separator line at bottom of header
		dc.SetColor(color.RGBA{80, 80, 80, 255}) // Light gray line
		dc.SetLineWidth(1)
		separatorY := float64(item.Y + headerHeight - 1) // At bottom of header
		dc.DrawLine(float64(item.X), separatorY, float64(item.X+item.Width), separatorY)
		dc.Stroke()
	}

	// Calculate chart area (excluding header)
	chartY := item.Y + headerHeight
	chartHeight := item.Height - headerHeight

	minVal, maxVal := c.getMinMax(history)

	// Use monitor's max value if available and reasonable
	if monitor != nil {
		monitorValue := monitor.GetValue()
		if monitorValue != nil && monitorValue.Max > 0 {
			if monitorValue.Max > maxVal {
				maxVal = monitorValue.Max
			}
		}
		if monitorValue != nil {
			minVal = monitorValue.Min
		}
	}

	// Override with config values if specified
	if item.MaxValue != nil {
		maxVal = *item.MaxValue
	}
	if item.MinValue != nil {
		minVal = *item.MinValue
	}

	if maxVal == minVal {
		maxVal = minVal + 1
	}

	// Draw chart line
	itemColor := item.Color
	if itemColor == "" {
		// Use dynamic color based on current value
		if len(history) > 0 {
			currentValue := history[len(history)-1]
			itemColor = config.GetDynamicColor(item.Monitor, currentValue)
		} else {
			itemColor = getColorFromConfig(item.Monitor, "chart_line", "#3b82f6", config)
		}
	}
	dc.SetColor(parseColor(itemColor))
	dc.SetLineWidth(1.5)

	// Add padding to avoid overlap with borders
	padding := 1.0
	chartAreaX := float64(item.X) + padding
	chartAreaY := float64(chartY) + padding
	chartAreaWidth := float64(item.Width) - 2*padding
	chartAreaHeight := float64(chartHeight) - 2*padding

	points := make([]float64, 0, len(history)*2)
	for i, histVal := range history {
		x := chartAreaX + float64(i)*chartAreaWidth/float64(len(history)-1)
		y := chartAreaY + chartAreaHeight - (histVal-minVal)/(maxVal-minVal)*chartAreaHeight
		points = append(points, x, y)
	}

	if len(points) >= 4 {
		dc.MoveTo(points[0], points[1])
		for i := 2; i < len(points); i += 2 {
			dc.LineTo(points[i], points[i+1])
		}
		dc.Stroke()
	}

	// Draw border
	dc.SetColor(color.RGBA{80, 80, 80, 255})
	dc.SetLineWidth(1)
	dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height))
	dc.Stroke()

	// Draw header content if enabled
	if item.GetShowHeader() && monitor != nil {
		c.drawHeader(dc, item, monitor, fontCache, config, headerHeight)
	}

	return nil
}

func (c *ChartRenderer) updateHistory(monitor string, value float64) {
	maxPoints := c.historySize
	if maxPoints <= 0 {
		maxPoints = 60 // Default fallback
	}

	if _, exists := c.history[monitor]; !exists {
		// Initialize with full size filled with zeros for complete history
		c.history[monitor] = make([]float64, maxPoints)
		for i := range c.history[monitor] {
			c.history[monitor][i] = 0.0
		}
	}

	history := c.history[monitor]

	// Shift all values left and add new value at the end
	if len(history) >= maxPoints {
		copy(history[0:], history[1:])
		history[maxPoints-1] = value
	} else {
		// This shouldn't happen with our initialization, but handle it anyway
		history = append(history, value)
		if len(history) > maxPoints {
			history = history[len(history)-maxPoints:]
		}
	}

	c.history[monitor] = history
}

func (c *ChartRenderer) getMinMax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 1
	}

	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return min, max
}

// Removed duplicate functions - now using common utilities from render_common.go

func (c *ChartRenderer) drawHeader(dc *gg.Context, item *ItemConfig, monitor MonitorItem, fontCache *FontCache, config *MonitorConfig, headerHeight int) {
	fontSize := 16
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
			// Use actual text height for positioning
			_, textHeight := dc.MeasureString("Ag")
			labelY := float64(item.Y) + 2 + textHeight      // 2px from top edge + actual text height
			dc.DrawString(label, float64(item.X+2), labelY) // 2px from left edge
		}
	}

	// Draw current value on the right
	value := monitor.GetValue()
	if value != nil {
		valueText := c.formatValue(value, item.GetShowUnit())
		if valueText != "" {
			// Use dynamic color for value text
			textColor := config.Colors["default_text"]
			textColor = getDynamicColorFromMonitor(item.Monitor, monitor, config)
			dc.SetColor(parseColor(textColor))

			// Measure text to position it on the right
			textWidth, _ := dc.MeasureString(valueText)
			valueX := float64(item.X+item.Width) - textWidth - 2 // 2px from right edge
			// Use actual text height for positioning
			_, textHeight := dc.MeasureString("Ag")
			valueY := float64(item.Y) + 2 + textHeight // 2px from top edge + actual text height

			dc.DrawString(valueText, valueX, valueY)
		}
	}
}

func (c *ChartRenderer) formatValue(value *MonitorValue, showUnit bool) string {
	return FormatMonitorValue(value, showUnit, "")
}
