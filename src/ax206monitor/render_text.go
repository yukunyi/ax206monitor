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

	// Draw background using common utility
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, item.Background)

	fontSize := item.FontSize
	if fontSize == 0 {
		fontSize = config.GetDefaultFontSize()
	}

	itemColor := item.Color
	if itemColor == "" {
		itemColor = getColorFromConfig("", "default_text", "#ffffff", config)
	}

	// Draw centered text using common utility
	drawCenteredText(dc, item.Text, item.X, item.Y, item.Width, item.Height, fontSize, itemColor, fontCache)

	return nil
}
