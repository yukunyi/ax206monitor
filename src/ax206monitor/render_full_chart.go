package main

import "github.com/fogleman/gg"

type FullChartRenderer struct {
	history *renderHistoryStore
}

func NewFullChartRenderer(history *renderHistoryStore) *FullChartRenderer {
	return &FullChartRenderer{history: history}
}

func (r *FullChartRenderer) GetType() string {
	return itemTypeFullChart
}

func (r *FullChartRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
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
	lineColor := resolveFullChartLineColor(item, config)
	textColor := resolveItemStaticColor(item, config)

	contentPadding := getItemAttrFloatCfg(item, config, "content_padding", 1)
	if contentPadding < 0 {
		contentPadding = 0
	}
	headerRect, bodyRect, labelFace, valueFace := fullBuildHeaderAndBody(item, config, fontCache, labelText, displayValue, contentPadding, 4)
	drawFullHeader(dc, item, config, headerRect, labelFace, valueFace, labelText, displayValue, textColor, lineColor, 1)

	r.drawBody(dc, item, value, numberValue, lineColor, bodyRect, config)
	drawBaseItemBorder(dc, item, config, cardRadius)
	return nil
}

func (r *FullChartRenderer) drawBody(dc *gg.Context, item *ItemConfig, value *CollectValue, numberValue float64, lineColor string, body fullRect, config *MonitorConfig) {
	history := appendRenderHistory(r.history, item, itemTypeFullChart, numberValue, 90, config)
	if len(history) == 0 {
		return
	}

	minValue, maxValue := resolveEffectiveMinMax(item, value, historyMin(history), historyMax(history))
	if maxValue <= minValue {
		maxValue = minValue + 1
	}

	chartAreaBg := getItemAttrColorCfg(item, config, "chart_area_bg", "")
	if chartAreaBg != "" {
		drawRoundedRectFill(dc, body.x, body.y, body.w, body.h, 4, chartAreaBg)
	}
	chartAreaBorder := getItemAttrColorCfg(item, config, "chart_area_border_color", "")
	if chartAreaBorder != "" {
		dc.SetLineWidth(1)
		dc.SetColor(parseColor(chartAreaBorder))
		dc.DrawRoundedRectangle(body.x, body.y, body.w, body.h, 4)
		dc.Stroke()
	}

	showSegmentLines := getItemAttrBoolCfg(
		item,
		config,
		"show_segment_lines",
		getItemAttrBoolCfg(item, config, "show_grid_lines", true),
	)
	if showSegmentLines {
		gridLines := getItemAttrIntCfg(item, config, "grid_lines", 4)
		if gridLines < 2 {
			gridLines = 2
		}
		dc.SetLineWidth(1)
		dc.SetColor(parseColor("#4755693c"))
		for i := 0; i < gridLines; i++ {
			y := body.y + float64(i)*(body.h/float64(gridLines-1))
			dc.DrawLine(body.x, y, body.x+body.w, y)
			dc.Stroke()
		}
	}

	lineWidth := getItemAttrFloatCfg(item, config, "line_width", 2)
	if lineWidth < 1 {
		lineWidth = 1
	}
	enableThresholdColors := getItemAttrBoolCfg(item, config, "enable_threshold_colors", false)
	thresholds := []float64{}
	colors := []string{}
	if enableThresholdColors {
		thresholds = effectiveThresholds(item, minValue, maxValue, config)
		colors = effectiveLevelColors(item, config)
	}

	type chartPoint struct {
		x float64
		y float64
		v float64
	}
	pointsOnChart := make([]chartPoint, 0, len(history))
	for idx, histValue := range history {
		if !isFiniteHistoryValue(histValue) {
			continue
		}
		x := body.x
		if len(history) > 1 {
			x = body.x + body.w*float64(idx)/float64(len(history)-1)
		}
		y := body.y + body.h - ((histValue-minValue)/(maxValue-minValue))*body.h
		y = clampFloat64(y, body.y, body.y+body.h)
		pointsOnChart = append(pointsOnChart, chartPoint{x: x, y: y, v: histValue})
	}
	if len(pointsOnChart) < 2 {
		return
	}

	if enableThresholdColors && len(thresholds) > 0 && len(colors) > 0 {
		dc.SetLineWidth(lineWidth)
		for idx := 1; idx < len(pointsOnChart); idx++ {
			p0 := pointsOnChart[idx-1]
			p1 := pointsOnChart[idx]
			segmentColor := resolveChartThresholdColor((p0.v+p1.v)/2, thresholds, colors, lineColor)
			dc.SetColor(parseColor(segmentColor))
			dc.DrawLine(p0.x, p0.y, p1.x, p1.y)
			dc.Stroke()
		}
	} else {
		dc.MoveTo(pointsOnChart[0].x, pointsOnChart[0].y)
		for idx := 1; idx < len(pointsOnChart); idx++ {
			p := pointsOnChart[idx]
			dc.LineTo(p.x, p.y)
		}
		dc.SetLineWidth(lineWidth)
		dc.SetColor(parseColor(lineColor))
		dc.Stroke()
	}

	if getItemAttrBoolCfg(item, config, "show_avg_line", true) {
		avg := historyAverage(history)
		y := body.y + body.h - ((avg-minValue)/(maxValue-minValue))*body.h
		y = clampFloat64(y, body.y, body.y+body.h)
		dc.SetColor(parseColor(applyAlpha(lineColor, 0.7)))
		dc.SetDash(4, 4)
		dc.SetLineWidth(1)
		dc.DrawLine(body.x, y, body.x+body.w, y)
		dc.Stroke()
		dc.SetDash()
	}
}
