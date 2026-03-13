package main

import "github.com/fogleman/gg"

type ValueRenderer struct{}

func NewValueRenderer() *ValueRenderer {
	return &ValueRenderer{}
}

func (v *ValueRenderer) GetType() string {
	return itemTypeSimpleValue
}

func (v *ValueRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
	monitor, value, ok := frame.AvailableItemValue(item)
	if !ok {
		return nil
	}

	radius := resolveItemRadius(item, config, 0)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	valueText, unitText := resolveItemDisplayValueParts(item, monitor, value, config)
	_, fontSize := resolveRoleFontFace(fontCache, item, config, TextRoleValue, 18, 8)
	_, unitFontSize := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)

	itemColor := resolveMonitorColor(item, monitor, config)
	unitColor := resolveUnitColor(item, config, itemColor)
	drawCenteredValueWithUnit(dc, valueText, unitText, item.X, item.Y, item.Width, item.Height, fontSize, itemColor, unitFontSize, unitColor, fontCache)
	drawBaseItemBorder(dc, item, config, radius)

	return nil
}
