package main

import (
	"fmt"

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
	return "line_chart"
}

func (c *LineChartRenderer) Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error {
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

	historySize := 60
	if config != nil && config.HistorySize > 0 {
		historySize = config.HistorySize
	}
	if item.PointSize > 0 {
		historySize = item.PointSize
	}
	if historySize < 10 {
		historySize = 10
	}

	historyKey := c.getHistoryKey(item)
	c.updateHistory(historyKey, val, historySize)
	history := c.history[historyKey]

	radius := float64(item.Radius)
	if radius < 0 {
		radius = 0
	}
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)

	if len(history) < 2 {
		drawItemBorder(dc, item)
		return nil
	}

	minVal, maxVal := c.getMinMax(history)
	if item.MinValue != nil {
		minVal = *item.MinValue
	} else if value.Min < minVal {
		minVal = value.Min
	}
	if item.MaxValue != nil {
		maxVal = *item.MaxValue
	} else if item.Max > 0 {
		maxVal = item.Max
	} else if value.Max > maxVal {
		maxVal = value.Max
	}
	if maxVal <= minVal {
		maxVal = minVal + 1
	}

	lineColor := resolveMonitorColor(item, monitor, config)
	dc.SetColor(parseColor(lineColor))
	dc.SetLineWidth(1.5)

	padding := 2.0
	chartX := float64(item.X) + padding
	chartY := float64(item.Y) + padding
	chartWidth := float64(item.Width) - 2*padding
	chartHeight := float64(item.Height) - 2*padding
	if chartWidth <= 1 || chartHeight <= 1 {
		drawItemBorder(dc, item)
		return nil
	}

	for idx, histValue := range history {
		x := chartX
		if len(history) > 1 {
			x = chartX + float64(idx)*chartWidth/float64(len(history)-1)
		}
		y := chartY + chartHeight - (histValue-minVal)/(maxVal-minVal)*chartHeight
		if idx == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.Stroke()

	drawItemBorder(dc, item)
	_ = fontCache
	return nil
}

func (c *LineChartRenderer) getHistoryKey(item *ItemConfig) string {
	return fmt.Sprintf("%s|%d|%d|%d|%d", item.Monitor, item.X, item.Y, item.Width, item.Height)
}

func (c *LineChartRenderer) updateHistory(key string, value float64, historySize int) {
	history := c.history[key]
	if len(history) == 0 {
		history = make([]float64, 0, historySize)
	}
	history = append(history, value)
	if len(history) > historySize {
		history = history[len(history)-historySize:]
	}
	c.history[key] = history
}

func (c *LineChartRenderer) getMinMax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 1
	}
	minValue := values[0]
	maxValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	return minValue, maxValue
}
