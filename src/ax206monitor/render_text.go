package main

import (
	"github.com/fogleman/gg"
)

type TextRenderer struct{}

func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

func (t *TextRenderer) GetType() string {
	return "text"
}

func (t *TextRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
	if item.Text == "" {
		return nil
	}

	if item.Background != "" {
		dc.SetColor(parseColor(item.Background))
		dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height), 5)
		dc.Fill()
	}

	fontSize := item.FontSize
	if fontSize == 0 {
		fontSize = config.GetDefaultFontSize()
	}

	font, err := fontCache.GetFont(fontSize)
	if err != nil {
		font = fontCache.contentFont
	}
	dc.SetFontFace(font)

	itemColor := item.Color
	if itemColor == "" {
		if color, exists := config.Colors["default_text"]; exists {
			itemColor = color
		} else {
			itemColor = "#ffffff"
		}
	}
	dc.SetColor(parseColor(itemColor))

	textWidth, textHeight := dc.MeasureString(item.Text)
	x := float64(item.X) + (float64(item.Width)-textWidth)/2
	y := float64(item.Y) + (float64(item.Height)+textHeight)/2

	dc.DrawString(item.Text, x, y)
	return nil
}
