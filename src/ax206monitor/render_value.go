package main

import "github.com/fogleman/gg"

type ValueRenderer struct{}

func NewValueRenderer() *ValueRenderer {
	return &ValueRenderer{}
}

func (v *ValueRenderer) GetType() string {
	return itemTypeSimpleValue
}

func (v *ValueRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}

	radius := float64(item.Radius)
	if radius < 0 {
		radius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	value := monitor.GetValue()
	valueText, unitText := FormatCollectValueParts(value, resolveUnitOverride(item))
	fontSize := resolveItemFontSize(item, config, 16)
	unitFontSize := resolveUnitFontSize(item, config, fontSize)

	itemColor := resolveMonitorColor(item, monitor, config)
	unitColor := resolveUnitColor(item, itemColor)
	drawCenteredValueWithUnit(dc, valueText, unitText, item.X, item.Y, item.Width, item.Height, fontSize, itemColor, unitFontSize, unitColor, fontCache)
	drawItemBorder(dc, item)

	return nil
}
