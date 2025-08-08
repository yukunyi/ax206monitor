package main

import (
	"image/color"

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
	val, ok := tryGetFloat64(value.Value)
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

	// Calculate header height using actual text metrics
	headerHeight := 0
	if item.GetShowHeader() {
		if item.FontSize > 0 {
			// Use font cache to get font and measure text height
			if font, err := fontCache.GetFont(item.FontSize); err == nil {
				dc.SetFontFace(font)
				_, textHeight := dc.MeasureString("Ag") // Use characters with ascenders and descenders
				headerHeight = int(textHeight) + 4      // 2px top + 2px bottom
			} else {
				headerHeight = item.FontSize + 4 // fallback
			}
		} else {
			headerHeight = 20 // fallback
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
			// Use dynamic color based on monitor value
			itemColor = getDynamicColorFromMonitor(item.Monitor, monitor, config)
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

// Removed duplicate functions - now using common utilities from render_common.go

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
			// Use actual text height for positioning
			_, textHeight := dc.MeasureString("Ag")
			labelY := float64(item.Y) + 2 + textHeight // 2px from top edge + actual text height
			dc.DrawString(label, float64(item.X+2), labelY)
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
			valueX := float64(item.X+item.Width) - textWidth - 2 // 2px from right edge
			// Use actual text height for positioning
			_, textHeight := dc.MeasureString("Ag")
			valueY := float64(item.Y) + 2 + textHeight // 2px from top edge + actual text height

			dc.DrawString(valueText, valueX, valueY)
		}
	}
}

func (p *ProgressRenderer) formatValue(value *MonitorValue, showUnit bool) string {
	return FormatMonitorValue(value, showUnit, "")
}
