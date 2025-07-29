package main

import (
	"image"

	"github.com/fogleman/gg"
)

type RenderItem interface {
	Render(dc *gg.Context, item *ItemConfig, registry *MonitorRegistry, fontCache *FontCache, config *MonitorConfig) error
	GetType() string
}

type RenderManager struct {
	renderers map[string]RenderItem
	fontCache *FontCache
	registry  *MonitorRegistry
}

func NewRenderManager(fontCache *FontCache, registry *MonitorRegistry) *RenderManager {
	rm := &RenderManager{
		renderers: make(map[string]RenderItem),
		fontCache: fontCache,
		registry:  registry,
	}

	rm.RegisterRenderer(NewValueRenderer())
	rm.RegisterRenderer(NewBigValueRenderer())
	rm.RegisterRenderer(NewProgressRenderer())
	rm.RegisterRenderer(NewChartRenderer())
	rm.RegisterRenderer(NewTextRenderer())
	rm.RegisterRenderer(NewRectRenderer())

	return rm
}

func (rm *RenderManager) RegisterRenderer(renderer RenderItem) {
	rm.renderers[renderer.GetType()] = renderer
}

func (rm *RenderManager) Render(config *MonitorConfig) (image.Image, error) {
	dc := gg.NewContext(config.Width, config.Height)

	dc.SetRGBA(0.1, 0.1, 0.1, 1.0)
	dc.Clear()

	for _, item := range config.Items {
		if renderer, exists := rm.renderers[item.Type]; exists {
			if err := renderer.Render(dc, &item, rm.registry, rm.fontCache, config); err != nil {
				continue
			}
		}
	}

	return dc.Image(), nil
}
