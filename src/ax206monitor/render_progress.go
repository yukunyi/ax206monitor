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

	maxValue := item.Max
	if maxValue <= 0 {
		if item.MaxValue != nil {
			maxValue = *item.MaxValue
		} else if value.Max > 0 {
			maxValue = value.Max
		} else {
			maxValue = 100
		}
	}
	if maxValue <= 0 {
		maxValue = 100
	}
	if val < 0 {
		val = 0
	}
	if val > maxValue {
		val = maxValue
	}

	radius := float64(item.Radius)
	if radius < 0 {
		radius = 0
	}

	bgColor := resolveItemBackground(item, config)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, bgColor, radius)

	percentage := val / maxValue
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
	unitColor := resolveUnitColor(item, textColor)
	drawCenteredValueWithUnit(dc, valueText, unitText, item.X, item.Y, item.Width, item.Height, fontSize, textColor, unitFontSize, unitColor, fontCache)

	drawItemBorder(dc, item)
	return nil
}
