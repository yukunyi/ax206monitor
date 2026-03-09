package main

import (
	"fmt"
	"image"
	"strings"

	"github.com/fogleman/gg"
)

type RenderItem interface {
	Render(dc *gg.Context, item *ItemConfig, registry *CollectorManager, fontCache *FontCache, config *MonitorConfig) error
	GetType() string
}

type RenderManager struct {
	renderers map[string]RenderItem
	fontCache *FontCache
	registry  *CollectorManager
}

func NewRenderManager(fontCache *FontCache, registry *CollectorManager) *RenderManager {
	rm := &RenderManager{
		renderers: make(map[string]RenderItem),
		fontCache: fontCache,
		registry:  registry,
	}

	rm.RegisterRenderer(NewValueRenderer())
	rm.RegisterRenderer(NewProgressRenderer())
	rm.RegisterRenderer(NewLineChartRenderer())
	rm.RegisterRenderer(NewLabelRenderer())
	rm.RegisterRenderer(NewRectRenderer())
	rm.RegisterRenderer(NewCircleRenderer())
	rm.RegisterRenderer(NewLabelTextRenderer(itemTypeLabelText1))
	rm.RegisterRenderer(NewLabelTextRenderer(itemTypeLabelText2))

	fullHistory := newFullHistoryStore()
	for _, itemType := range fullItemTypes {
		rm.RegisterRenderer(NewFullWidgetRenderer(itemType, fullHistory))
	}

	return rm
}

func (rm *RenderManager) RegisterRenderer(renderer RenderItem) {
	rm.renderers[renderer.GetType()] = renderer
}

func (rm *RenderManager) Render(config *MonitorConfig) (image.Image, error) {
	dc := gg.NewContext(config.Width, config.Height)
	dc.SetColor(parseColor(config.GetDefaultBackgroundColor()))
	dc.Clear()

	for idx := range config.Items {
		item := &config.Items[idx]
		renderer, exists := rm.renderers[item.Type]
		if !exists {
			continue
		}
		if err := rm.renderItemSafely(renderer, dc, item, config); err != nil {
			logWarnModule("render", "skip item idx=%d type=%s monitor=%s: %v", idx, item.Type, strings.TrimSpace(item.Monitor), err)
		}
	}

	return dc.Image(), nil
}

func (rm *RenderManager) renderItemSafely(renderer RenderItem, dc *gg.Context, item *ItemConfig, config *MonitorConfig) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return renderer.Render(dc, item, rm.registry, rm.fontCache, config)
}
