package main

import (
	"math"
	"strings"

	"github.com/fogleman/gg"
)

type FullProgressRenderer struct{}

func NewFullProgressRenderer() *FullProgressRenderer {
	return &FullProgressRenderer{}
}

func (r *FullProgressRenderer) GetType() string {
	return itemTypeFullProgress
}

func (r *FullProgressRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
	monitor := registry.Get(item.Monitor)
	if monitor == nil || !monitor.IsAvailable() {
		return nil
	}
	value := monitor.GetValue()
	if value == nil {
		return nil
	}
	numberValue, ok := tryGetFloat64(value.Value)
	if !ok {
		return nil
	}

	cardRadius := getItemAttrFloatCfg(item, config, "card_radius", -1)
	if cardRadius < 0 {
		cardRadius = resolveItemRadius(item, config, 0)
	}
	if cardRadius < 0 {
		cardRadius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), cardRadius)

	labelText, displayValue := fullResolveTexts(item, monitor, value, config)
	lineColor := resolveMonitorColor(item, monitor, config)
	textColor := resolveItemStaticColor(item, config)

	contentPadding := getItemAttrFloatCfg(item, config, "content_padding", 1)
	if contentPadding < 0 {
		contentPadding = 0
	}
	headerRect, bodyRect, labelFace, valueFace := fullBuildHeaderAndBody(item, config, fontCache, labelText, displayValue, contentPadding, 0)
	drawFullHeader(dc, item, config, headerRect, labelFace, valueFace, labelText, displayValue, textColor, lineColor, 0)

	r.drawBody(dc, item, value, numberValue, lineColor, bodyRect, config)
	drawBaseItemBorder(dc, item, config, cardRadius)
	return nil
}

func (r *FullProgressRenderer) drawBody(dc *gg.Context, item *ItemConfig, value *CollectValue, numberValue float64, lineColor string, body fullRect, config *MonitorConfig) {
	minValue, maxValue := resolveEffectiveMinMax(item, value, 0, 100)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	style := strings.ToLower(getItemAttrStringCfg(item, config, "progress_style", "gradient"))
	if style == "glow" {
		style = "gradient"
	}
	barRadius := getItemAttrFloatCfg(item, config, "bar_radius", 0)
	if barRadius <= 0 {
		barRadius = resolveItemRadius(item, config, 0)
	}
	if barRadius < 0 {
		barRadius = 0
	}

	barHeight := getItemAttrFloatCfg(item, config, "bar_height", 0)
	if barHeight <= 0 || barHeight > body.h {
		barHeight = body.h
	}
	if barHeight < 4 {
		barHeight = 4
	}
	if barRadius > barHeight/2 {
		barRadius = barHeight / 2
	}
	barY := body.y + (body.h-barHeight)/2
	trackColor := getItemAttrColorCfg(item, config, "track_color", "#1f2937")
	drawRoundedRectFill(dc, body.x, barY, body.w, barHeight, barRadius, trackColor)

	fillWidth := body.w * progress
	if fillWidth <= 1 {
		return
	}

	switch style {
	case "solid":
		drawRoundedRectFill(dc, body.x, barY, fillWidth, barHeight, barRadius, lineColor)
	case "segmented":
		segments := getItemAttrIntCfg(item, config, "segments", 12)
		if segments < 4 {
			segments = 4
		}
		gap := getItemAttrFloatCfg(item, config, "segment_gap", 2)
		segW := (body.w - float64(segments-1)*gap) / float64(segments)
		filled := int(math.Round(progress * float64(segments)))
		for i := 0; i < segments; i++ {
			segX := body.x + float64(i)*(segW+gap)
			colorValue := applyAlpha(lineColor, 0.22)
			if i < filled {
				colorValue = lineColor
			}
			segRadius := barRadius
			if segRadius > segW/2 {
				segRadius = segW / 2
			}
			drawRoundedRectFill(dc, segX, barY, segW, barHeight, segRadius, colorValue)
		}
	case "stripes":
		drawRoundedRectFill(dc, body.x, barY, fillWidth, barHeight, barRadius, lineColor)
		dc.DrawRoundedRectangle(body.x, barY, fillWidth, barHeight, barRadius)
		dc.Clip()
		dc.SetColor(parseColor(applyAlpha("#ffffff", 0.2)))
		for x := body.x - barHeight; x < body.x+fillWidth+barHeight; x += 8 {
			dc.DrawLine(x, barY+barHeight, x+barHeight, barY)
			dc.SetLineWidth(2)
			dc.Stroke()
		}
		dc.ResetClip()
	default:
		drawRoundedRectFill(dc, body.x, barY, fillWidth, barHeight, barRadius, applyAlpha(lineColor, 0.9))
		dc.DrawRoundedRectangle(body.x, barY, fillWidth, barHeight, barRadius)
		dc.Clip()
		gradient := gg.NewLinearGradient(body.x, barY, body.x, barY+barHeight)
		gradient.AddColorStop(0, parseColor(applyAlpha("#ffffff", 0.35)))
		gradient.AddColorStop(0.45, parseColor(applyAlpha("#ffffff", 0.08)))
		gradient.AddColorStop(1, parseColor(applyAlpha("#000000", 0.2)))
		dc.SetFillStyle(gradient)
		dc.DrawRectangle(body.x, barY, fillWidth, barHeight)
		dc.Fill()
		dc.ResetClip()
	}
}
