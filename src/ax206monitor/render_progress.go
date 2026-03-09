package main

import (
	"github.com/fogleman/gg"
)

type ProgressRenderer struct{}

func NewProgressRenderer() *ProgressRenderer {
	return &ProgressRenderer{}
}

func (p *ProgressRenderer) GetType() string {
	return itemTypeSimpleProgress
}

func (p *ProgressRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}

	value := monitor.GetValue()
	if value == nil {
		return nil
	}
	val, ok := tryGetFloat64(value.Value)
	if !ok {
		return nil
	}

	minValue, maxValue := resolveEffectiveMinMax(item, value, 0, 100)
	if val < minValue {
		val = minValue
	}
	if val > maxValue {
		val = maxValue
	}

	radius := resolveItemRadius(item, config, 0)

	bgColor := resolveItemBackground(item, config)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, bgColor, radius)

	percentage := (val - minValue) / (maxValue - minValue)
	fillWidth := float64(item.Width) * percentage
	if fillWidth > 0 {
		itemColor := resolveMonitorColor(item, monitor, config)
		dc.SetColor(parseColor(itemColor))
		if radius > 0 {
			dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), fillWidth, float64(item.Height), radius)
		} else {
			dc.DrawRectangle(float64(item.X), float64(item.Y), fillWidth, float64(item.Height))
		}
		dc.Fill()
	}

	valueText, unitText := FormatCollectValueParts(value, resolveUnitOverride(item))
	fontSize := resolveItemFontSize(item, config, 14)
	unitFontSize := resolveUnitFontSize(item, config, fontSize)
	textColor := config.GetDefaultTextColor()
	unitColor := resolveUnitColor(item, config, textColor)
	drawCenteredValueWithUnit(dc, valueText, unitText, item.X, item.Y, item.Width, item.Height, fontSize, textColor, unitFontSize, unitColor, fontCache)

	drawBaseItemBorder(dc, item, config, radius)
	return nil
}
