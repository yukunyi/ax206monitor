package main

import (
	"github.com/fogleman/gg"
)

type BigValueRenderer struct{}

func NewBigValueRenderer() *BigValueRenderer {
	return &BigValueRenderer{}
}

func (b *BigValueRenderer) GetType() string {
	return "big_value"
}

func (b *BigValueRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}

	// Draw background using common utility
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, item.Background)

	// Draw label in top-left corner if enabled
	if item.GetShowLabel() {
		b.drawLabel(dc, item, monitor, fontCache, config)
	}

	// Draw value in center
	value := monitor.GetValue()
	text := FormatMonitorValue(value, item.GetShowUnit(), item.UnitText)

	fontSize := b.calculateFontSize(dc, item, text, fontCache, config)

	itemColor := item.Color
	if itemColor == "" {
		itemColor = getDynamicColorFromMonitor(item.Monitor, monitor, config)
	}

	// Draw centered text using common utility
	drawCenteredText(dc, text, item.X, item.Y, item.Width, item.Height, fontSize, itemColor, fontCache)

	return nil
}

func (b *BigValueRenderer) drawLabel(dc *gg.Context, item *ItemConfig, monitor MonitorItem, fontCache *FontCache, config *MonitorConfig) {
	label := item.LabelText
	if label == "" {
		label = config.GetLabelText(monitor.GetName(), monitor.GetLabel())
	}
	if label == "" {
		return
	}

	fontSize := config.GetMinLabelFontSize()
	if item.LabelFontSize > 0 {
		fontSize = item.LabelFontSize
	}

	font, err := fontCache.GetFont(fontSize)
	if err != nil {
		return
	}

	dc.SetFontFace(font)
	dc.SetColor(parseColor(config.Colors["default_text"]))

	// Draw label at top-left corner with fixed 2px padding using actual text height
	_, textHeight := dc.MeasureString("Ag")
	dc.DrawString(label, float64(item.X+2), float64(item.Y)+2+textHeight)
}

func (b *BigValueRenderer) calculateFontSize(dc *gg.Context, item *ItemConfig, text string, fontCache *FontCache, config *MonitorConfig) int {
	if item.ValueFontSize > 0 {
		return item.ValueFontSize
	}
	if item.FontSize > 0 {
		return item.FontSize
	}

	maxWidth := float64(item.Width) * 0.8
	maxHeight := float64(item.Height) * 0.6

	return calculateOptimalFontSize(dc, text, maxWidth, maxHeight, fontCache, 12, 48)
}
