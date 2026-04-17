package main

import (
	"strings"

	"github.com/fogleman/gg"
)

type FullChartRenderer struct {
}

func NewFullChartRenderer() *FullChartRenderer {
	return &FullChartRenderer{}
}

func (r *FullChartRenderer) GetType() string {
	return itemTypeFullChart
}

func (r *FullChartRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
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
	lineColor := resolveFullChartLineColor(item, config)
	textColor := resolveItemStaticColor(item, config)
	valueColor := resolveMonitorValueColor(item, monitor.name, value, numberValue, config)
	unitColor := resolveMonitorUnitColor(item, monitor.name, value, numberValue, config)

	contentPaddingX, contentPaddingY := resolveContentPaddingXY(item, config, 1, 1, 0, 0)
	headerRect, bodyRect, labelFace, valueFace := fullBuildHeaderAndBody(item, config, fontCache, labelText, displayValue, contentPaddingX, contentPaddingY, 4)
	unitFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)
	drawFullHeader(dc, item, config, headerRect, labelFace, valueFace, labelText, "", textColor, valueColor)
	drawFullHeaderValueWithUnit(dc, headerRect, valueFace, unitFace, valueText, unitText, valueColor, unitColor)

	r.drawBody(dc, item, frame, value, numberValue, lineColor, bodyRect, config)
	drawBaseItemBorder(dc, item, config, cardRadius)
	return nil
}

func (r *FullChartRenderer) drawBody(dc *gg.Context, item *ItemConfig, frame *RenderFrame, value *CollectValue, numberValue float64, lineColor string, body fullRect, config *MonitorConfig) {
	history := appendFrameRenderHistory(frame, item, numberValue)
	if len(history) == 0 {
		return
	}

	minValue, maxValue := resolveEffectiveMinMax(item, value, history, numberValue)

	chartAreaBg := item.runtime.fullChart.chartAreaBg
	chartAreaBorder := item.runtime.fullChart.chartAreaBorder
	chartFillColor := item.runtime.fullChart.fillColor
	showSegmentLines := item.runtime.fullChart.showSegmentLines
	gridLines := item.runtime.fullChart.gridLines
	lineWidth := item.runtime.fullChart.lineWidth
	enableThresholdColors := item.runtime.fullChart.enableThresholdColors
	showAvgLine := item.runtime.fullChart.showAvgLine
	if !item.runtime.prepared {
		chartFillColor = getItemAttrColorCfg(item, config, "chart_fill_color", "rgba(0,0,0,0)")
		chartAreaBg = getItemAttrColorCfg(item, config, "chart_area_bg", "")
		chartAreaBorder = getItemAttrColorCfg(item, config, "chart_area_border_color", "")
		showSegmentLines = getItemAttrBoolCfg(
			item,
			config,
			"show_segment_lines",
			getItemAttrBoolCfg(item, config, "show_grid_lines", true),
		)
		gridLines = clampRenderInt(getItemAttrIntCfg(item, config, "grid_lines", 4), 2)
		lineWidth = clampRenderFloat(getItemAttrFloatCfg(item, config, "line_width", 2), 1)
		enableThresholdColors = getItemAttrBoolCfg(item, config, "enable_threshold_colors", false)
		showAvgLine = getItemAttrBoolCfg(item, config, "show_avg_line", true)
	}
	if chartAreaBg != "" {
		drawRoundedRectFill(dc, body.x, body.y, body.w, body.h, 4, chartAreaBg)
	}
	if chartAreaBorder != "" {
		dc.SetLineWidth(1)
		dc.SetColor(parseColor(chartAreaBorder))
		dc.DrawRoundedRectangle(body.x, body.y, body.w, body.h, 4)
		dc.Stroke()
	}

	if showSegmentLines {
		dc.SetLineWidth(1)
		dc.SetColor(parseColor("#4755693c"))
		for i := 0; i < gridLines; i++ {
			y := body.y + float64(i)*(body.h/float64(gridLines-1))
			dc.DrawLine(body.x, y, body.x+body.w, y)
			dc.Stroke()
		}
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

	if strings.TrimSpace(chartFillColor) != "" {
		bottomY := body.y + body.h
		dc.MoveTo(pointsOnChart[0].x, bottomY)
		dc.LineTo(pointsOnChart[0].x, pointsOnChart[0].y)
		for idx := 1; idx < len(pointsOnChart); idx++ {
			p := pointsOnChart[idx]
			dc.LineTo(p.x, p.y)
		}
		lastPoint := pointsOnChart[len(pointsOnChart)-1]
		dc.LineTo(lastPoint.x, bottomY)
		dc.ClosePath()
		dc.SetColor(parseColor(chartFillColor))
		dc.Fill()
	}

	if enableThresholdColors {
		dc.SetLineWidth(lineWidth)
		for idx := 1; idx < len(pointsOnChart); idx++ {
			p0 := pointsOnChart[idx-1]
			p1 := pointsOnChart[idx]
			segmentColor := resolveMonitorValueColor(item, item.Monitor, value, (p0.v+p1.v)/2, config)
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

	if showAvgLine {
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
