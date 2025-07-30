package main

import (
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

	// Draw background using common utility
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, item.Background)

	if item.GetShowLabel() {
		v.drawLabel(dc, item, monitor, fontCache, config)
	}

	if item.GetShowValue() {
		value := monitor.GetValue()
		text := FormatMonitorValue(value, item.GetShowUnit(), item.UnitText)

		fontSize := v.calculateFontSize(dc, item, text, fontCache, config)

		itemColor := item.Color
		if itemColor == "" {
			itemColor = getDynamicColorFromValue(item.Monitor, value.Value, config)
		}

		// Draw centered text using common utility
		drawCenteredText(dc, text, item.X, item.Y, item.Width, item.Height, fontSize, itemColor, fontCache)
	}

	return nil
}

// Removed duplicate functions - now using common utilities from render_common.go

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

	return calculateOptimalFontSize(dc, text, maxWidth, maxHeight, fontCache, 8, 20)
}
