package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/fogleman/gg"
)

type LineChartRenderer struct {
	history map[string][]float64
}

func NewLineChartRenderer() *LineChartRenderer {
	return &LineChartRenderer{
		history: make(map[string][]float64),
	}
}

func (c *LineChartRenderer) GetType() string {
	return itemTypeSimpleChart
}

func (c *LineChartRenderer) Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error {
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

	historySize := resolveItemHistoryPoints(item, config, 60)

	historyKey := c.getHistoryKey(item)
	c.updateHistory(historyKey, val, historySize)
	history := c.history[historyKey]

	radius := resolveItemRadius(item, config, 0)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	minVal, maxVal, ok := c.getMinMax(history)
	if !ok {
		drawBaseItemBorder(dc, item, config, radius)
		return nil
	}
	minVal, maxVal = resolveEffectiveMinMax(item, value, minVal, maxVal)

	lineColor := resolveMonitorColor(item, monitor, config)
	lineWidth := getItemAttrFloatCfg(item, config, "line_width", 1.5)
	if lineWidth < 1 {
		lineWidth = 1
	}
	enableThresholdColors := getItemAttrBoolCfg(item, config, "enable_threshold_colors", false)
	thresholds := []float64{}
	colors := []string{}
	if enableThresholdColors {
		thresholds = effectiveThresholds(item, minVal, maxVal, config)
		colors = effectiveLevelColors(item, config)
	}

	padding := 2.0
	chartX := float64(item.X) + padding
	chartY := float64(item.Y) + padding
	chartWidth := float64(item.Width) - 2*padding
	chartHeight := float64(item.Height) - 2*padding
	if chartWidth <= 1 || chartHeight <= 1 {
		drawBaseItemBorder(dc, item, config, radius)
		return nil
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
		x := chartX
		if len(history) > 1 {
			x = chartX + float64(idx)*chartWidth/float64(len(history)-1)
		}
		y := chartY + chartHeight - (histValue-minVal)/(maxVal-minVal)*chartHeight
		pointsOnChart = append(pointsOnChart, chartPoint{x: x, y: y, v: histValue})
	}
	if len(pointsOnChart) < 2 {
		drawBaseItemBorder(dc, item, config, radius)
		return nil
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
		dc.SetColor(parseColor(lineColor))
		dc.SetLineWidth(lineWidth)
		dc.Stroke()
	}

	drawBaseItemBorder(dc, item, config, radius)
	_ = fontCache
	return nil
}

func (c *LineChartRenderer) getHistoryKey(item *ItemConfig) string {
	if item != nil && strings.TrimSpace(item.ID) != "" {
		return fmt.Sprintf("id:%s|%s", strings.TrimSpace(item.ID), strings.TrimSpace(item.Monitor))
	}
	return fmt.Sprintf("%s|%d|%d|%d|%d", item.Monitor, item.X, item.Y, item.Width, item.Height)
}

func (c *LineChartRenderer) updateHistory(key string, value float64, historySize int) {
	history := c.history[key]
	if len(history) != historySize {
		history = resizeChartHistory(history, historySize)
	}
	if len(history) == 0 {
		history = make([]float64, historySize)
		for idx := range history {
			history[idx] = math.NaN()
		}
	}
	copy(history, history[1:])
	history[len(history)-1] = value
	c.history[key] = history
}

func resizeChartHistory(old []float64, historySize int) []float64 {
	if historySize < 1 {
		return []float64{}
	}
	resized := make([]float64, historySize)
	for idx := range resized {
		resized[idx] = math.NaN()
	}
	if len(old) == 0 {
		return resized
	}
	copyLen := len(old)
	if copyLen > historySize {
		copyLen = historySize
	}
	copy(resized[historySize-copyLen:], old[len(old)-copyLen:])
	return resized
}

func isFiniteHistoryValue(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func (c *LineChartRenderer) getMinMax(values []float64) (float64, float64, bool) {
	if len(values) == 0 {
		return 0, 1, false
	}
	minValue := 0.0
	maxValue := 0.0
	valid := false
	for _, value := range values {
		if !isFiniteHistoryValue(value) {
			continue
		}
		if !valid {
			minValue = value
			maxValue = value
			valid = true
			continue
		}
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	return minValue, maxValue, valid
}
