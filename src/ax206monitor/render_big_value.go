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

	// Draw background
	if item.Background != "" {
		dc.SetColor(parseColor(item.Background))
		dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height), 5)
		dc.Fill()
	}

	// Draw label in top-left corner if enabled
	if item.GetShowLabel() {
		b.drawLabel(dc, item, monitor, fontCache, config)
	}

	// Draw value in center
	value := monitor.GetValue()
	text := FormatMonitorValue(value, item.GetShowUnit(), item.UnitText)

	fontSize := b.calculateFontSize(dc, item, text, fontCache, config)
	font, err := fontCache.GetFont(fontSize)
	if err != nil {
		font = fontCache.contentFont
	}
	dc.SetFontFace(font)

	itemColor := item.Color
	if itemColor == "" {
		itemColor = config.Colors["default_text"]
		if itemColor == "" {
			itemColor = "#ffffff"
		}
	}
	dc.SetColor(parseColor(itemColor))

	textWidth, textHeight := dc.MeasureString(text)
	x := float64(item.X) + (float64(item.Width)-textWidth)/2
	y := float64(item.Y) + (float64(item.Height)+textHeight)/2

	dc.DrawString(text, x, y)
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

	// Draw label at top-left corner with small padding
	dc.DrawString(label, float64(item.X+4), float64(item.Y+fontSize+4))
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

	for fontSize := 48; fontSize >= 12; fontSize -= 2 {
		font, err := fontCache.GetFont(fontSize)
		if err != nil {
			continue
		}
		dc.SetFontFace(font)
		textWidth, textHeight := dc.MeasureString(text)
		if textWidth <= maxWidth && textHeight <= maxHeight {
			return fontSize
		}
	}
	return 12
}
