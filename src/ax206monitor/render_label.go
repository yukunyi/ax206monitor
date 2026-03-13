package main

import "github.com/fogleman/gg"

type LabelRenderer struct{}

func NewLabelRenderer() *LabelRenderer {
	return &LabelRenderer{}
}

func (r *LabelRenderer) GetType() string {
	return itemTypeSimpleLabel
}

func (r *LabelRenderer) RequiresMonitor() bool {
	return false
}

func (r *LabelRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
	_ = frame
	if item == nil || item.Text == "" {
		return nil
	}
	drawBaseItemFrame(dc, item, config)
	drawTextInItemRect(dc, fontCache, item, config, item.Text, item.X, item.Y, item.Width, item.Height, BaseTextDrawOptions{
		Role:     TextRoleLabel,
		AlignH:   AlignLeft,
		AlignV:   AlignMiddle,
		PaddingX: 4,
	})
	return nil
}
