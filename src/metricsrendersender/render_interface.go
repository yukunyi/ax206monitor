package main

import (
	"fmt"
	"image"
	"strings"
	"sync"

	"github.com/fogleman/gg"
)

type RenderItem interface {
	Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error
	GetType() string
}

type monitorBoundRenderer interface {
	RequiresMonitor() bool
}

type RenderManager struct {
	renderers map[string]RenderItem
	fontCache *FontCache
	registry  *CollectorManager
}

type renderFullCardRuntime struct {
	bodyGap             float64
	headerHeight        int
	headerDivider       bool
	headerDividerColor  string
	headerDividerWidth  float64
	headerDividerOffset float64
}

type renderSimpleChartRuntime struct {
	lineWidth             float64
	enableThresholdColors bool
	thresholdPercents     []float64
	levelColors           []string
}

type renderFullChartRuntime struct {
	lineColor             string
	chartAreaBg           string
	chartAreaBorder       string
	showSegmentLines      bool
	gridLines             int
	lineWidth             float64
	enableThresholdColors bool
	showAvgLine           bool
	thresholdPercents     []float64
	levelColors           []string
}

type renderFullTableRuntime struct {
	rows            []fullTableRowConfig
	colCount        int
	rowCount        int
	rowGap          float64
	rowRadius       float64
	rowBg           string
	rowAltBg        string
	columnGap       float64
	labelWidthRatio float64
	showUnits       bool
}

type renderFullProgressRuntime struct {
	style      string
	barRadius  float64
	barHeight  float64
	trackColor string
	segments   int
	segmentGap float64
}

type renderFullGaugeRuntime struct {
	thickness  float64
	gapDegrees float64
	trackColor string
	textGap    float64
}

type renderSimpleLineRuntime struct {
	orientation string
	lineWidth   float64
}

type renderSpecialFormatRuntime struct {
	monitorKey      string
	kind            string
	timeLayout      string
	displayTemplate string
}

type renderItemRuntime struct {
	prepared            bool
	historyKey          string
	historyPoints       int
	background          string
	staticColor         string
	explicitStaticColor string
	explicitUnitColor   string
	borderWidth         float64
	borderColor         string
	radius              float64
	hasCardRadius       bool
	cardRadius          float64
	hasPaddingX         bool
	hasPaddingY         bool
	paddingX            float64
	paddingY            float64
	valueFontSize       int
	textFontSize        int
	unitFontSize        int
	titleText           string
	labelText           string
	text                string
	fullCard            renderFullCardRuntime
	simpleChart         renderSimpleChartRuntime
	fullChart           renderFullChartRuntime
	fullTable           renderFullTableRuntime
	fullProgress        renderFullProgressRuntime
	fullGauge           renderFullGaugeRuntime
	simpleLine          renderSimpleLineRuntime
	specialFormat       renderSpecialFormatRuntime
}

type RenderMonitorSnapshot struct {
	name      string
	label     string
	available bool
	value     *CollectValue
}

type renderItemState struct {
	monitor *RenderMonitorSnapshot
}

type RenderFrame struct {
	registry *CollectorManager
	monitors map[string]*RenderMonitorSnapshot
	items    map[*ItemConfig]renderItemState
}

func newRenderFrame(registry *CollectorManager, renderers map[string]RenderItem, config *MonitorConfig) *RenderFrame {
	frame := &RenderFrame{
		registry: registry,
		monitors: make(map[string]*RenderMonitorSnapshot),
	}
	if registry == nil || config == nil || len(config.Items) == 0 {
		frame.items = make(map[*ItemConfig]renderItemState)
		return frame
	}
	frame.items = make(map[*ItemConfig]renderItemState, len(config.Items))

	for idx := range config.Items {
		item := &config.Items[idx]
		renderer := renderers[item.Type]
		state := renderItemState{}
		if rendererRequiresMonitor(renderer) {
			state.monitor = resolveRenderMonitorSnapshot(frame.monitors, registry, item.Monitor)
		}
		frame.items[item] = state
	}
	return frame
}

func resolveRenderMonitorSnapshot(cache map[string]*RenderMonitorSnapshot, registry *CollectorManager, name string) *RenderMonitorSnapshot {
	name = strings.TrimSpace(name)
	if name == "" || registry == nil {
		return nil
	}
	if monitor, exists := cache[name]; exists {
		return monitor
	}
	collectItem := registry.Get(name)
	if collectItem == nil {
		cache[name] = nil
		return nil
	}
	_, available, value := collectItem.SnapshotState()
	monitor := &RenderMonitorSnapshot{
		name:      collectItem.GetName(),
		label:     collectItem.GetLabel(),
		available: available,
		value:     value,
	}
	cache[name] = monitor
	return monitor
}

func rendererRequiresMonitor(renderer RenderItem) bool {
	if renderer == nil {
		return false
	}
	required, ok := renderer.(monitorBoundRenderer)
	if !ok {
		return true
	}
	return required.RequiresMonitor()
}

func (f *RenderFrame) AvailableItemValue(item *ItemConfig) (*RenderMonitorSnapshot, *CollectValue, bool) {
	if f == nil || item == nil {
		return nil, nil, false
	}
	state, exists := f.items[item]
	if !exists || state.monitor == nil || !state.monitor.available || state.monitor.value == nil {
		return nil, nil, false
	}
	return state.monitor, state.monitor.value, true
}

func (f *RenderFrame) ResolveMonitor(name string) *RenderMonitorSnapshot {
	if f == nil {
		return nil
	}
	return resolveRenderMonitorSnapshot(f.monitors, f.registry, name)
}

type RenderResult struct {
	Image image.Image

	mu          sync.Mutex
	outputFrame *OutputFrame
}

func NewRenderResult(img image.Image) *RenderResult {
	if img == nil {
		return nil
	}
	return &RenderResult{Image: img}
}

func (r *RenderResult) OutputFrame() *OutputFrame {
	if r == nil || r.Image == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.outputFrame == nil {
		r.outputFrame = NewOutputFrame(r.Image)
	}
	return r.outputFrame
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
	rm.RegisterRenderer(NewSimpleLineRenderer())
	rm.RegisterRenderer(NewLabelRenderer())
	rm.RegisterRenderer(NewRectRenderer())
	rm.RegisterRenderer(NewCircleRenderer())
	rm.RegisterRenderer(NewLabelTextRenderer(itemTypeLabelText))

	fullHistory := newRenderHistoryStore()
	rm.RegisterRenderer(NewFullChartRenderer(fullHistory))
	rm.RegisterRenderer(NewFullTableRenderer())
	rm.RegisterRenderer(NewFullProgressRenderer(itemTypeFullProgressH, false))
	rm.RegisterRenderer(NewFullProgressRenderer(itemTypeFullProgressV, true))
	rm.RegisterRenderer(NewFullGaugeRenderer())

	return rm
}

func (rm *RenderManager) RegisterRenderer(renderer RenderItem) {
	rm.renderers[renderer.GetType()] = renderer
}

func (rm *RenderManager) Render(config *MonitorConfig) (*RenderResult, error) {
	dc := gg.NewContext(config.Width, config.Height)
	dc.SetColor(parseColor(config.GetDefaultBackgroundColor()))
	dc.Clear()
	frame := newRenderFrame(rm.registry, rm.renderers, config)

	for idx := range config.Items {
		item := &config.Items[idx]
		renderer, exists := rm.renderers[item.Type]
		if !exists {
			continue
		}
		if err := rm.renderItemSafely(renderer, dc, item, frame, config); err != nil {
			logWarnModule("render", "skip item idx=%d type=%s monitor=%s: %v", idx, item.Type, strings.TrimSpace(item.Monitor), err)
		}
	}

	return NewRenderResult(dc.Image()), nil
}

func (rm *RenderManager) renderItemSafely(renderer RenderItem, dc *gg.Context, item *ItemConfig, frame *RenderFrame, config *MonitorConfig) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return renderer.Render(dc, item, frame, rm.fontCache, config)
}
