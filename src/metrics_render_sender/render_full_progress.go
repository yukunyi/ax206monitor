package main

import (
	"math"
	"strings"

	"github.com/fogleman/gg"
)

type FullProgressRenderer struct {
	itemType string
	vertical bool
}

func NewFullProgressRenderer(itemType string, vertical bool) *FullProgressRenderer {
	return &FullProgressRenderer{
		itemType: itemType,
		vertical: vertical,
	}
}

func (r *FullProgressRenderer) GetType() string {
	return r.itemType
}

func (r *FullProgressRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
	monitor, value, ok := frame.AvailableItemValue(item)
	if !ok {
		return nil
	}
	numberValue, ok := tryGetFloat64(value.Value)
	if !ok {
		return nil
	}

	cardRadius := resolveItemCardRadius(item, config)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), cardRadius)

	labelText, valueText, unitText := fullResolveTextParts(item, monitor, value, config)
	displayValue := strings.TrimSpace(valueText + " " + unitText)
	lineColor := resolveMonitorColor(item, monitor, config)
	textColor := resolveItemStaticColor(item, config)
	valueColor := resolveMonitorValueColor(item, monitor.name, value, numberValue, config)
	unitColor := resolveMonitorUnitColor(item, monitor.name, value, numberValue, config)

	if r.vertical {
		r.drawVertical(dc, item, frame, fontCache, value, numberValue, labelText, valueText, unitText, lineColor, textColor, valueColor, unitColor, config)
		drawBaseItemBorder(dc, item, config, cardRadius)
		return nil
	}

	contentPaddingX, contentPaddingY := resolveContentPaddingXY(item, config, 1, 1, 0, 0)
	headerRect, bodyRect, labelFace, valueFace := fullBuildHeaderAndBody(item, config, fontCache, labelText, displayValue, contentPaddingX, contentPaddingY, 0)
	unitFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)
	drawFullHeader(dc, item, config, headerRect, labelFace, valueFace, labelText, "", textColor, valueColor)
	drawFullHeaderValueWithUnit(dc, headerRect, valueFace, unitFace, valueText, unitText, valueColor, unitColor)
	r.drawHorizontalBody(dc, item, frame, value, numberValue, lineColor, bodyRect, config)
	drawBaseItemBorder(dc, item, config, cardRadius)
	return nil
}

func (r *FullProgressRenderer) drawHorizontalBody(dc *gg.Context, item *ItemConfig, frame *RenderFrame, value *CollectValue, numberValue float64, lineColor string, body fullRect, config *MonitorConfig) {
	progress, style, barRadius, barHeight, trackColor, segments, segmentGap := resolveFullProgressLayout(item, frame, value, numberValue, config)
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
	drawRoundedRectFill(dc, body.x, barY, body.w, barHeight, barRadius, trackColor)

	fillWidth := body.w * progress
	if fillWidth <= 1 {
		return
	}
	drawFullProgressFillHorizontal(dc, style, body.x, barY, fillWidth, body.w, barHeight, barRadius, lineColor, segments, segmentGap)
}

func (r *FullProgressRenderer) drawVertical(
	dc *gg.Context,
	item *ItemConfig,
	frame *RenderFrame,
	fontCache *FontCache,
	value *CollectValue,
	numberValue float64,
	labelText string,
	valueText string,
	unitText string,
	lineColor string,
	textColor string,
	valueColor string,
	unitColor string,
	config *MonitorConfig,
) {
	progress, style, barRadius, barWidth, trackColor, segments, segmentGap := resolveFullProgressLayout(item, frame, value, numberValue, config)
	contentPaddingX, contentPaddingY := resolveContentPaddingXY(item, config, 2, 2, 0, 0)

	valueFace, valueFontSize := resolveRoleFontFace(fontCache, item, config, TextRoleValue, 18, 8)
	_, unitFontSize := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)
	textFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleText, 16, 8)
	displayValue := strings.TrimSpace(valueText + " " + unitText)

	content := fullRect{
		x: float64(item.X) + contentPaddingX,
		y: float64(item.Y) + contentPaddingY,
		w: float64(item.Width) - contentPaddingX*2,
		h: float64(item.Height) - contentPaddingY*2,
	}
	if content.w < 1 {
		content.w = 1
	}
	if content.h < 1 {
		content.h = 1
	}

	valueMetrics := baseMeasureText(valueFace, displayValue)
	labelMetrics := baseMeasureText(textFace, labelText)
	valueHeight := math.Ceil(valueMetrics.ascent + valueMetrics.descent + 4)
	labelHeight := math.Ceil(labelMetrics.ascent + labelMetrics.descent + 4)
	if valueHeight < 12 {
		valueHeight = 12
	}
	if labelHeight < 10 {
		labelHeight = 10
	}

	sectionGap := 3.0
	barHeight := content.h - valueHeight - labelHeight - sectionGap*2
	if barHeight < 12 {
		barHeight = 12
		sectionGap = 1
	}
	if barHeight > content.h {
		barHeight = content.h
	}

	valueRect := fullRect{x: content.x, y: content.y, w: content.w, h: valueHeight}
	barRect := fullRect{x: content.x, y: valueRect.y + valueRect.h + sectionGap, w: content.w, h: barHeight}
	labelRect := fullRect{x: content.x, y: barRect.y + barRect.h + sectionGap, w: content.w, h: labelHeight}
	if labelRect.y+labelRect.h > content.y+content.h {
		labelRect.h = math.Max(1, content.y+content.h-labelRect.y)
	}

	drawCenteredValueWithUnit(
		dc,
		valueText,
		unitText,
		int(math.Round(valueRect.x)),
		int(math.Round(valueRect.y)),
		int(math.Round(valueRect.w)),
		int(math.Round(valueRect.h)),
		valueFontSize,
		valueColor,
		unitFontSize,
		unitColor,
		fontCache,
	)

	if barWidth <= 0 || barWidth > barRect.w {
		barWidth = math.Min(barRect.w, math.Max(10, math.Min(barRect.w*0.42, 24)))
	}
	if barWidth < 6 {
		barWidth = math.Min(barRect.w, 6)
	}
	if barRadius > barWidth/2 {
		barRadius = barWidth / 2
	}

	trackX := barRect.x + (barRect.w-barWidth)/2
	drawRoundedRectFill(dc, trackX, barRect.y, barWidth, barRect.h, barRadius, trackColor)

	fillHeight := barRect.h * progress
	if fillHeight > 1 {
		fillY := barRect.y + barRect.h - fillHeight
		drawFullProgressFillVertical(dc, style, trackX, fillY, barWidth, fillHeight, barRect.h, barRadius, lineColor, segments, segmentGap)
	}

	dc.SetColor(parseColor(textColor))
	drawBaseMetricAnchoredText(dc, textFace, labelText, labelRect.x+labelRect.w/2, labelRect.y+labelRect.h/2, 0.5)
}

func resolveFullProgressLayout(item *ItemConfig, frame *RenderFrame, value *CollectValue, numberValue float64, config *MonitorConfig) (float64, string, float64, float64, string, int, float64) {
	history := appendFrameRenderHistory(frame, item, numberValue)
	minValue, maxValue := resolveEffectiveMinMax(item, value, history, numberValue)
	progress := normalizeRatio(numberValue, minValue, maxValue)

	style := item.runtime.fullProgress.style
	barRadius := item.runtime.fullProgress.barRadius
	barHeight := item.runtime.fullProgress.barHeight
	trackColor := item.runtime.fullProgress.trackColor
	segments := item.runtime.fullProgress.segments
	segmentGap := item.runtime.fullProgress.segmentGap
	if !item.runtime.prepared {
		style = normalizeFullProgressStyle(getItemAttrStringCfg(item, config, "progress_style", "gradient"))
		barRadius = getItemAttrFloatCfg(item, config, "bar_radius", 0)
		if barRadius <= 0 {
			barRadius = resolveItemRadius(item, config, 0)
		}
		if barRadius < 0 {
			barRadius = 0
		}
		barHeight = getItemAttrFloatCfg(item, config, "bar_height", 0)
		trackColor = getItemAttrColorCfg(item, config, "track_color", "#1f2937")
		segments = clampRenderInt(getItemAttrIntCfg(item, config, "segments", 12), 4)
		segmentGap = getItemAttrFloatCfg(item, config, "segment_gap", 2)
	}
	return progress, style, barRadius, barHeight, trackColor, segments, segmentGap
}

func drawFullProgressFillHorizontal(
	dc *gg.Context,
	style string,
	x float64,
	y float64,
	fillWidth float64,
	totalWidth float64,
	barHeight float64,
	barRadius float64,
	lineColor string,
	segments int,
	segmentGap float64,
) {
	switch style {
	case "solid":
		drawRoundedRectFill(dc, x, y, fillWidth, barHeight, barRadius, lineColor)
	case "segmented":
		segW := (totalWidth - float64(segments-1)*segmentGap) / float64(segments)
		filled := int(math.Round((fillWidth / totalWidth) * float64(segments)))
		for i := 0; i < segments; i++ {
			segX := x + float64(i)*(segW+segmentGap)
			colorValue := applyAlpha(lineColor, 0.22)
			if i < filled {
				colorValue = lineColor
			}
			segRadius := barRadius
			if segRadius > segW/2 {
				segRadius = segW / 2
			}
			drawRoundedRectFill(dc, segX, y, segW, barHeight, segRadius, colorValue)
		}
	case "stripes":
		drawRoundedRectFill(dc, x, y, fillWidth, barHeight, barRadius, lineColor)
		dc.DrawRoundedRectangle(x, y, fillWidth, barHeight, barRadius)
		dc.Clip()
		dc.SetColor(parseColor(applyAlpha("#ffffff", 0.2)))
		for lineX := x - barHeight; lineX < x+fillWidth+barHeight; lineX += 8 {
			dc.DrawLine(lineX, y+barHeight, lineX+barHeight, y)
			dc.SetLineWidth(2)
			dc.Stroke()
		}
		dc.ResetClip()
	default:
		drawRoundedRectFill(dc, x, y, fillWidth, barHeight, barRadius, applyAlpha(lineColor, 0.9))
		dc.DrawRoundedRectangle(x, y, fillWidth, barHeight, barRadius)
		dc.Clip()
		gradient := gg.NewLinearGradient(x, y, x, y+barHeight)
		gradient.AddColorStop(0, parseColor(applyAlpha("#ffffff", 0.35)))
		gradient.AddColorStop(0.45, parseColor(applyAlpha("#ffffff", 0.08)))
		gradient.AddColorStop(1, parseColor(applyAlpha("#000000", 0.2)))
		dc.SetFillStyle(gradient)
		dc.DrawRectangle(x, y, fillWidth, barHeight)
		dc.Fill()
		dc.ResetClip()
	}
}

func drawFullProgressFillVertical(
	dc *gg.Context,
	style string,
	x float64,
	y float64,
	barWidth float64,
	fillHeight float64,
	totalHeight float64,
	barRadius float64,
	lineColor string,
	segments int,
	segmentGap float64,
) {
	switch style {
	case "solid":
		drawRoundedRectFill(dc, x, y, barWidth, fillHeight, barRadius, lineColor)
	case "segmented":
		segH := (totalHeight - float64(segments-1)*segmentGap) / float64(segments)
		filled := int(math.Round((fillHeight / totalHeight) * float64(segments)))
		trackTop := y + fillHeight - totalHeight
		for i := 0; i < segments; i++ {
			segY := trackTop + totalHeight - segH - float64(i)*(segH+segmentGap)
			colorValue := applyAlpha(lineColor, 0.22)
			if i < filled {
				colorValue = lineColor
			}
			segRadius := barRadius
			if segRadius > segH/2 {
				segRadius = segH / 2
			}
			drawRoundedRectFill(dc, x, segY, barWidth, segH, segRadius, colorValue)
		}
	case "stripes":
		drawRoundedRectFill(dc, x, y, barWidth, fillHeight, barRadius, lineColor)
		dc.DrawRoundedRectangle(x, y, barWidth, fillHeight, barRadius)
		dc.Clip()
		dc.SetColor(parseColor(applyAlpha("#ffffff", 0.2)))
		for lineY := y + fillHeight + barWidth; lineY > y-barWidth; lineY -= 8 {
			dc.DrawLine(x, lineY, x+barWidth, lineY-barWidth)
			dc.SetLineWidth(2)
			dc.Stroke()
		}
		dc.ResetClip()
	default:
		drawRoundedRectFill(dc, x, y, barWidth, fillHeight, barRadius, applyAlpha(lineColor, 0.9))
		dc.DrawRoundedRectangle(x, y, barWidth, fillHeight, barRadius)
		dc.Clip()
		gradient := gg.NewLinearGradient(x, y, x+barWidth, y)
		gradient.AddColorStop(0, parseColor(applyAlpha("#ffffff", 0.22)))
		gradient.AddColorStop(0.5, parseColor(applyAlpha("#ffffff", 0.05)))
		gradient.AddColorStop(1, parseColor(applyAlpha("#000000", 0.18)))
		dc.SetFillStyle(gradient)
		dc.DrawRectangle(x, y, barWidth, fillHeight)
		dc.Fill()
		dc.ResetClip()
	}
}
