package main

import (
	"ax206monitor/rtsssource"
	"fmt"
	"image"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fogleman/gg"
)

const (
	webTickerInterval = 250 * time.Millisecond
)

type WebMonitorSnapshotItem struct {
	Available bool   `json:"available"`
	Label     string `json:"label,omitempty"`
	Text      string `json:"text"`
	Unit      string `json:"unit,omitempty"`
}

type WebSnapshotResponse struct {
	Mode           string                            `json:"mode"`
	UpdatedAt      string                            `json:"updated_at"`
	Monitors       []string                          `json:"monitors"`
	Values         map[string]WebMonitorSnapshotItem `json:"values"`
	MonitorRuntime *CollectorManagerStats            `json:"monitor_runtime,omitempty"`
}

type WebAPI struct {
	mu            sync.RWMutex
	applyMu       sync.Mutex
	renderMu      sync.Mutex
	fontCache     *FontCache
	config        *MonitorConfig
	required      []string
	registry      *CollectorManager
	renderManager *RenderManager
	outputManager *OutputManager
	outputTypes   []string
	outputHasMem  bool
	idleProvider  func() (*MonitorConfig, error)
	previewOutput *MemImgOutputHandler
	lastProbeCC   string
	lastProbeLHM  string
	lastProbeRTSS bool
	lockMonitor   LockScreenMonitor

	activityMu   sync.RWMutex
	lastActivity time.Time
	modeFull     bool
	updatedAt    time.Time
	realtimeConn int32
	lastEpoch    int64

	lockMu           sync.RWMutex
	lockScreenActive bool
	lockPauseEnabled bool
	lockFrameReady   bool

	outputChan chan webOutputFrame
	outputWg   sync.WaitGroup

	stopOnce sync.Once
	stopCh   chan struct{}
	stopped  chan struct{}
}

type webOutputFrame struct {
	img        image.Image
	enqueuedAt time.Time
	modeFull   bool
}

func NewWebAPI(cfg *MonitorConfig) (*WebAPI, error) {
	fontCache, err := loadFontCache()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize web runtime fonts: %w", err)
	}

	runtime := &WebAPI{
		fontCache:     fontCache,
		previewOutput: NewMemImgOutputHandler(),
		outputChan:    make(chan webOutputFrame, 1),
		stopCh:        make(chan struct{}),
		stopped:       make(chan struct{}),
	}

	if err := runtime.applyConfigInternal(cfg, false); err != nil {
		return nil, err
	}

	runtime.outputWg.Add(1)
	go runtime.outputLoop()
	go runtime.loop()
	return runtime, nil
}

func (r *WebAPI) Touch() {
	r.activityMu.Lock()
	r.lastActivity = time.Now()
	r.activityMu.Unlock()
}

func (r *WebAPI) ApplyConfig(cfg *MonitorConfig) error {
	if err := r.applyConfigInternal(cfg, false); err != nil {
		return err
	}
	_, _ = r.renderOnce(true)
	return nil
}

func (r *WebAPI) ApplyPreviewConfig(cfg *MonitorConfig) error {
	// Preview edit should not rebuild USB outputs on every minor config change.
	if err := r.applyConfigInternal(cfg, true); err != nil {
		return err
	}
	_, err := r.renderOnce(true)
	return err
}

func (r *WebAPI) SetRealtimeConnectionCount(count int) {
	if count < 0 {
		count = 0
	}
	atomic.StoreInt32(&r.realtimeConn, int32(count))
}

func (r *WebAPI) SetIdleConfigProvider(provider func() (*MonitorConfig, error)) {
	r.mu.Lock()
	r.idleProvider = provider
	r.mu.Unlock()
}

func (r *WebAPI) outputLoop() {
	defer r.outputWg.Done()
	for frame := range r.outputChan {
		if frame.img == nil {
			continue
		}
		outputStart := time.Now()
		// Always refresh preview buffer for WebSocket clients, independent of output types.
		if err := r.previewOutput.Output(frame.img); err != nil {
			logDebugModule("web", "preview output failed: %v", err)
		}

		_, _, _, _, outputManager, _ := r.getRuntimeRefs()
		if outputManager == nil {
			continue
		}
		if err := outputManager.Output(frame.img); err != nil {
			logDebugModule("web", "runtime output failed: %v", err)
			continue
		}
		queueDelay := outputStart.Sub(frame.enqueuedAt)
		outputDuration := time.Since(outputStart)
		logDebugModule("web", "output=%v queue=%v", outputDuration, queueDelay)
	}
}

func (r *WebAPI) renderOnce(forceFull bool) (bool, error) {
	r.renderMu.Lock()
	defer r.renderMu.Unlock()

	modeFull := forceFull || r.isFullMode()
	r.setMode(modeFull)
	noteRenderAccess()

	cfg, _, registry, renderManager, _, _ := r.getRuntimeRefs()
	if cfg == nil || registry == nil || renderManager == nil {
		return false, nil
	}
	if r.shouldRenderLockScreen() {
		img := r.renderLockScreenImage(cfg)
		replaced, ok := enqueueLatestWebFrame(r.outputChan, webOutputFrame{
			img:        img,
			enqueuedAt: time.Now(),
			modeFull:   modeFull,
		})
		if !ok {
			logDebugModule("web", "output queue busy, skip locked frame")
			return false, nil
		}
		if replaced {
			logDebugModule("web", "output queue replaced stale frame")
		}
		r.setLockFrameReady(true)
		r.setUpdatedAt(time.Now())
		return true, nil
	}
	r.setLockFrameReady(false)

	waitMax := cfg.GetRenderWaitMaxDuration()
	currentEpoch := registry.CurrentEpoch()
	if currentEpoch > r.lastEpoch {
		waitComplete, waitDuration := registry.WaitForEpoch(currentEpoch, waitMax)
		logDebugModule("web", "epoch=%d wait=%v complete=%v", currentEpoch, waitDuration, waitComplete)
		r.lastEpoch = currentEpoch
	} else if !forceFull {
		return false, nil
	}

	renderStartedAt := time.Now()
	img, err := renderManager.Render(cfg)
	if err != nil {
		return false, err
	}
	recordRenderDuration(time.Since(renderStartedAt))

	replaced, ok := enqueueLatestWebFrame(r.outputChan, webOutputFrame{
		img:        img,
		enqueuedAt: time.Now(),
		modeFull:   modeFull,
	})
	if !ok {
		logDebugModule("web", "output queue busy, skip frame")
	} else if replaced {
		logDebugModule("web", "output queue replaced stale frame")
	}

	r.setUpdatedAt(time.Now())
	return true, nil
}

func (r *WebAPI) applyConfigInternal(cfg *MonitorConfig, forceMemImg bool) error {
	r.applyMu.Lock()
	defer r.applyMu.Unlock()

	// Prevent output manager swap/close from racing with ongoing render output.
	r.renderMu.Lock()
	defer r.renderMu.Unlock()

	configCopy := cloneMonitorConfig(cfg)
	normalizeMonitorConfig(configCopy)

	SetGlobalCollectorConfig(configCopy)
	initializeCache()

	required := getRequiredMonitors(configCopy)
	registry := GetCollectorManagerWithConfig(required, configCopy.GetNetworkInterface())
	registry.SetPreviewMode(forceMemImg)
	renderManager := NewRenderManager(r.fontCache, registry)

	outputTypes := registry.ResolveOutputTypes(configCopy.OutputTypes)
	outputHasMem := false
	for _, typeName := range outputTypes {
		if typeName == outputTypeMemImg {
			outputHasMem = true
			break
		}
	}

	var outputManager *OutputManager
	r.mu.RLock()
	existingOutputManager := r.outputManager
	existingOutputTypes := append([]string(nil), r.outputTypes...)
	r.mu.RUnlock()
	if existingOutputManager != nil && equalStringSlices(existingOutputTypes, outputTypes) {
		outputManager = existingOutputManager
	} else {
		outputManager, _ = buildOutputManager(&MonitorConfig{OutputTypes: outputTypes}, false)
	}

	r.mu.Lock()
	oldOutputManager := r.outputManager
	r.config = configCopy
	r.required = required
	r.registry = registry
	r.renderManager = renderManager
	r.outputManager = outputManager
	r.outputTypes = append([]string(nil), outputTypes...)
	r.outputHasMem = outputHasMem
	r.lastEpoch = 0
	r.mu.Unlock()
	r.setLockPauseEnabled(configCopy.IsPauseCollectOnLockEnabled())
	r.updateLockMonitorState()
	r.applyLockPolicy()

	if oldOutputManager != nil && oldOutputManager != outputManager {
		oldOutputManager.Close()
	}
	r.maybeProbeDataSources(configCopy)
	return nil
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func enqueueLatestWebFrame(ch chan webOutputFrame, frame webOutputFrame) (bool, bool) {
	select {
	case ch <- frame:
		return false, true
	default:
	}

	replaced := false
	select {
	case <-ch:
		replaced = true
	default:
	}

	select {
	case ch <- frame:
		return replaced, true
	default:
		return replaced, false
	}
}

func (r *WebAPI) maybeProbeDataSources(cfg *MonitorConfig) {
	ccURL := cfg.GetCoolerControlURL()
	lhmURL := cfg.GetLibreHardwareMonitorURL()
	ccEnabled := cfg.IsCollectorEnabled(collectorCoolerControl, false)
	lhmEnabled := cfg.IsCollectorEnabled(collectorLibreHardwareMonitor, false)
	rtssEnabled := cfg.IsCollectorEnabled(collectorRTSS, false)

	shouldProbeCC := false
	shouldProbeLHM := false
	shouldProbeRTSS := false

	r.mu.Lock()
	if !ccEnabled {
		r.lastProbeCC = ""
	} else if ccURL != "" && ccURL != r.lastProbeCC {
		r.lastProbeCC = ccURL
		shouldProbeCC = true
	}
	if !lhmEnabled {
		r.lastProbeLHM = ""
	} else if lhmURL != "" && lhmURL != r.lastProbeLHM {
		r.lastProbeLHM = lhmURL
		shouldProbeLHM = true
	}
	if !rtssEnabled {
		r.lastProbeRTSS = false
	} else if !r.lastProbeRTSS {
		r.lastProbeRTSS = true
		shouldProbeRTSS = true
	}
	r.mu.Unlock()

	if shouldProbeCC {
		go func(url string, cfg *MonitorConfig) {
			options, err := listConfiguredCoolerControlOptions(cfg)
			if err != nil {
				logWarnModule("web", "coolercontrol probe failed (url=%s): %v", url, err)
				return
			}
			logInfoModule(
				"web",
				"coolercontrol probe success (url=%s, monitors=%d)",
				url,
				len(options),
			)
		}(ccURL, cfg)
	}

	if shouldProbeLHM {
		go func(url string, cfg *MonitorConfig) {
			items, err := listConfiguredLibreHardwareMonitorOptions(cfg)
			if err != nil {
				logWarnModule("web", "librehardwaremonitor probe failed (url=%s): %v", url, err)
				return
			}
			logInfoModule("web", "librehardwaremonitor probe success (url=%s, monitors=%d)", url, len(items))
		}(lhmURL, cfg)
	}

	if shouldProbeRTSS {
		go func() {
			client := rtsssource.GetRTSSClient()
			connected := client.RefreshMetrics(250 * time.Millisecond)
			options := client.ListMonitorOptions()
			if !connected {
				logWarnModule("web", "rtss probe failed: shared memory unavailable or no active app")
				return
			}
			logInfoModule("web", "rtss probe success (monitors=%d)", len(options))
		}()
	}
}

func (r *WebAPI) loop() {
	defer close(r.stopped)
	ticker := time.NewTicker(webTickerInterval)
	defer ticker.Stop()

	lastModeFull := false

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
		}

		modeFull := r.isFullMode()
		if modeFull != lastModeFull {
			if modeFull {
				logInfoModule("web", "Monitor update mode switched to full scan")
			} else {
				logInfoModule("web", "Monitor update mode switched to required-only")
				r.restoreIdleConfig()
			}
			lastModeFull = modeFull
		}
		cfg, _, _, _, _, _ := r.getRuntimeRefs()
		if cfg == nil {
			continue
		}
		if r.shouldRenderLockScreen() && r.isLockFrameReady() {
			continue
		}

		_, err := r.renderOnce(false)
		if err != nil {
			logDebugModule("web", "render runtime image failed: %v", err)
			continue
		}
	}
}

func (r *WebAPI) Snapshot() WebSnapshotResponse {
	modeFull := r.isFullMode()
	r.setMode(modeFull)

	cfg, required, registry, _, _, _ := r.getRuntimeRefs()
	values := make(map[string]WebMonitorSnapshotItem)
	if registry == nil {
		return WebSnapshotResponse{
			Mode:      "required",
			UpdatedAt: time.Now().Format(time.RFC3339Nano),
			Values:    values,
		}
	}
	monitorStats := registry.Stats()

	names := make([]string, 0)
	if modeFull {
		all := registry.GetAll()
		for name := range all {
			names = append(names, name)
		}
	} else {
		names = append(names, required...)
	}
	sort.Strings(names)

	for _, name := range names {
		monitor := registry.Get(name)
		if monitor == nil {
			continue
		}
		item := WebMonitorSnapshotItem{
			Available: monitor.IsAvailable(),
			Label:     strings.TrimSpace(monitor.GetLabel()),
			Text:      "-",
		}
		if monitor.IsAvailable() {
			if value := monitor.GetValue(); value != nil {
				item.Text = FormatCollectValue(value, true, "")
				item.Unit = value.Unit
			}
		}
		values[name] = item
	}
	optimizeWebSnapshotLabels(values)

	mode := "required"
	if modeFull {
		mode = "full"
	}

	updatedAt := r.getUpdatedAt()
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	_ = cfg
	return WebSnapshotResponse{
		Mode:           mode,
		UpdatedAt:      updatedAt.Format(time.RFC3339Nano),
		Monitors:       names,
		Values:         values,
		MonitorRuntime: &monitorStats,
	}
}

func (r *WebAPI) MonitorStats() *CollectorManagerStats {
	_, _, registry, _, _, _ := r.getRuntimeRefs()
	if registry == nil {
		return nil
	}
	stats := registry.Stats()
	return &stats
}

func (r *WebAPI) AllMonitorNames() []string {
	_, _, registry, _, _, _ := r.getRuntimeRefs()
	if registry == nil {
		return []string{}
	}
	all := registry.GetAll()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *WebAPI) CollectorNames() []string {
	_, _, registry, _, _, _ := r.getRuntimeRefs()
	if registry == nil {
		return []string{}
	}
	return registry.CollectorNames()
}

func (r *WebAPI) CollectorStates() map[string]bool {
	_, _, registry, _, _, _ := r.getRuntimeRefs()
	if registry == nil {
		return map[string]bool{}
	}
	return registry.CollectorStates()
}

func (r *WebAPI) SetCollectorEnabled(name string, enabled bool) bool {
	_, _, registry, _, _, _ := r.getRuntimeRefs()
	if registry == nil {
		return false
	}
	return registry.SetCollectorEnabled(name, enabled)
}

func (r *WebAPI) CurrentConfig() *MonitorConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.config == nil {
		return nil
	}
	return cloneMonitorConfig(r.config)
}

func (r *WebAPI) Close() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
		<-r.stopped
	})

	r.mu.Lock()
	oldOutputManager := r.outputManager
	outputChan := r.outputChan
	r.outputChan = nil
	r.outputManager = nil
	r.outputTypes = nil
	r.registry = nil
	r.renderManager = nil
	r.config = nil
	r.required = nil
	r.outputHasMem = false
	r.mu.Unlock()

	if outputChan != nil {
		close(outputChan)
	}
	r.outputWg.Wait()
	if oldOutputManager != nil {
		oldOutputManager.Close()
	}
	r.lockMu.Lock()
	monitor := r.lockMonitor
	r.lockMonitor = nil
	r.lockMu.Unlock()
	if monitor != nil {
		monitor.Close()
	}
}

func (r *WebAPI) SetSystemLocked(locked bool) {
	r.lockMu.Lock()
	changed := r.lockScreenActive != locked
	r.lockScreenActive = locked
	if locked {
		r.lockFrameReady = false
	}
	enabled := r.lockPauseEnabled
	r.lockMu.Unlock()

	r.applyLockPolicy()
	if !changed {
		return
	}
	if enabled && locked {
		logInfoModule("main", "lock detected, pause collection and render lock screen")
	} else if enabled {
		logInfoModule("main", "unlock detected, resume collection")
	}
	go func() {
		_, _ = r.renderOnce(true)
	}()
}

func (r *WebAPI) setLockPauseEnabled(enabled bool) {
	r.lockMu.Lock()
	r.lockPauseEnabled = enabled
	if !enabled {
		r.lockFrameReady = false
		r.lockScreenActive = false
	}
	r.lockMu.Unlock()
}

func (r *WebAPI) isLockPauseEnabled() bool {
	r.lockMu.RLock()
	defer r.lockMu.RUnlock()
	return r.lockPauseEnabled
}

func (r *WebAPI) updateLockMonitorState() {
	enabled := r.isLockPauseEnabled()
	r.lockMu.RLock()
	current := r.lockMonitor
	r.lockMu.RUnlock()

	if !enabled {
		if current != nil {
			current.Close()
			r.lockMu.Lock()
			if r.lockMonitor == current {
				r.lockMonitor = nil
			}
			r.lockMu.Unlock()
		}
		return
	}
	if current != nil {
		return
	}
	monitor, err := StartLockScreenMonitor(func(locked bool) {
		r.SetSystemLocked(locked)
	})
	if err != nil {
		logWarnModule("main", "lock screen monitor unavailable: %v", err)
		return
	}
	if monitor == nil {
		return
	}
	r.lockMu.Lock()
	if r.lockPauseEnabled && r.lockMonitor == nil {
		r.lockMonitor = monitor
		monitor = nil
	}
	r.lockMu.Unlock()
	if monitor != nil {
		monitor.Close()
	}
}

func (r *WebAPI) shouldRenderLockScreen() bool {
	r.lockMu.RLock()
	defer r.lockMu.RUnlock()
	return r.lockPauseEnabled && r.lockScreenActive
}

func (r *WebAPI) setLockFrameReady(ready bool) {
	r.lockMu.Lock()
	r.lockFrameReady = ready
	r.lockMu.Unlock()
}

func (r *WebAPI) isLockFrameReady() bool {
	r.lockMu.RLock()
	defer r.lockMu.RUnlock()
	return r.lockFrameReady
}

func (r *WebAPI) applyLockPolicy() {
	r.mu.RLock()
	registry := r.registry
	r.mu.RUnlock()
	if registry == nil {
		return
	}
	registry.SetPaused(r.shouldRenderLockScreen())
}

func (r *WebAPI) renderLockScreenImage(cfg *MonitorConfig) image.Image {
	width := 480
	height := 320
	if cfg != nil {
		if cfg.Width > 0 {
			width = cfg.Width
		}
		if cfg.Height > 0 {
			height = cfg.Height
		}
	}
	dc := gg.NewContext(width, height)
	bg := "#0b1220"
	color := "#f8fafc"
	if cfg != nil {
		bg = cfg.GetDefaultBackgroundColor()
		color = cfg.GetDefaultTextColor()
	}
	dc.SetColor(parseColor(bg))
	dc.Clear()

	fontSize := int(math.Round(math.Min(float64(width), float64(height)) * 0.16))
	if fontSize < 24 {
		fontSize = 24
	}
	if fontSize > 64 {
		fontSize = 64
	}
	face := resolveFontFace(r.fontCache, fontSize)
	dc.SetColor(parseColor(color))
	drawMetricAnchoredText(dc, face, "已锁屏", float64(width)/2, float64(height)/2, 0.5)
	return dc.Image()
}

func (r *WebAPI) getRuntimeRefs() (*MonitorConfig, []string, *CollectorManager, *RenderManager, *OutputManager, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	requiredCopy := append([]string(nil), r.required...)
	return r.config, requiredCopy, r.registry, r.renderManager, r.outputManager, r.outputHasMem
}

func (r *WebAPI) restoreIdleConfig() {
	r.mu.RLock()
	provider := r.idleProvider
	r.mu.RUnlock()
	if provider == nil {
		return
	}

	cfg, err := provider()
	if err != nil {
		logWarnModule("web", "load active profile for idle mode failed: %v", err)
		return
	}
	if cfg == nil {
		return
	}
	if err := r.applyConfigInternal(cfg, false); err != nil {
		logWarnModule("web", "apply active profile for idle mode failed: %v", err)
		return
	}
	_, _ = r.renderOnce(false)
}

func (r *WebAPI) isFullMode() bool {
	_ = atomic.LoadInt32(&r.realtimeConn)
	return true
}

func (r *WebAPI) setMode(modeFull bool) {
	r.activityMu.Lock()
	r.modeFull = modeFull
	r.activityMu.Unlock()
}

func (r *WebAPI) setUpdatedAt(updatedAt time.Time) {
	r.activityMu.Lock()
	r.updatedAt = updatedAt
	r.activityMu.Unlock()
}

func (r *WebAPI) getUpdatedAt() time.Time {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()
	return r.updatedAt
}

func optimizeWebSnapshotLabels(values map[string]WebMonitorSnapshotItem) {
	if len(values) == 0 {
		return
	}

	explicit := map[string]string{
		"go_native.cpu.usage":                     "CPU usage",
		"go_native.cpu.temp":                      "CPU temperature",
		"go_native.cpu.freq":                      "CPU frequency",
		"go_native.cpu.max_freq":                  "CPU max frequency",
		"go_native.cpu.model":                     "CPU model",
		"go_native.cpu.cores":                     "CPU cores",
		"go_native.memory.usage":                  "Memory usage",
		"go_native.memory.used":                   "Memory used",
		"go_native.memory.total":                  "Memory total",
		"go_native.memory.usage_text":             "Memory usage detail",
		"go_native.memory.usage_progress":         "Memory usage progress",
		"go_native.memory.swap_usage":             "Swap usage",
		"go_native.system.load_avg":               "System load average",
		"go_native.system.current_time":           "Current time",
		"go_native.system.hostname":               "Host name",
		"go_native.system.resolution":             "Display resolution",
		"go_native.system.refresh_rate":           "Display refresh rate",
		"go_native.system.display":                "Display mode",
		"go_native.system.collect.max_ms":         "Collect max ms",
		"go_native.system.collect.avg_ms":         "Collect avg ms",
		"go_native.system.render.max_ms":          "Render max ms",
		"go_native.system.render.avg_ms":          "Render avg ms",
		"go_native.system.output.max_ms":          "Output max ms",
		"go_native.system.output.avg_ms":          "Output avg ms",
		"go_native.system.output.memimg.max_ms":   "Output memimg max ms",
		"go_native.system.output.memimg.avg_ms":   "Output memimg avg ms",
		"go_native.system.output.ax206usb.max_ms": "Output ax206usb max ms",
		"go_native.system.output.ax206usb.avg_ms": "Output ax206usb avg ms",
		"alias.cpu.usage":                         "CPU usage",
		"alias.cpu.temp":                          "CPU temperature",
		"alias.cpu.freq":                          "CPU frequency",
		"alias.cpu.max_freq":                      "CPU max frequency",
		"alias.cpu.power":                         "CPU power",
		"alias.memory.usage":                      "Memory usage",
		"alias.memory.used":                       "Memory used",
		"alias.gpu.fps":                           "GPU FPS",
		"alias.gpu.usage":                         "GPU usage",
		"alias.gpu.power":                         "GPU power",
		"alias.gpu.vram":                          "GPU VRAM usage",
		"alias.gpu.temp":                          "GPU temperature",
		"alias.gpu.fan":                           "GPU fan speed",
		"alias.gpu.freq":                          "GPU frequency",
		"alias.gpu.max_freq":                      "GPU max frequency",
		"alias.net.upload":                        "Network upload",
		"alias.net.download":                      "Network download",
		"alias.net.ip":                            "IP address",
		"alias.net.interface":                     "Network interface",
		"alias.system.time":                       "System time",
		"alias.system.hostname":                   "Host name",
		"alias.system.load":                       "System load",
		"alias.system.resolution":                 "Display resolution",
		"alias.system.refresh_rate":               "Display refresh rate",
		"alias.system.display":                    "Display mode",
		"alias.disk.temp":                         "Disk temperature",
		"alias.fan.cpu":                           "CPU fan speed",
		"alias.fan.gpu":                           "GPU fan speed",
		"alias.fan.system":                        "System fan speed",
		"rtss_fps":                                "RTSS FPS",
		"rtss_frametime_ms":                       "RTSS frame time",
		"rtss_max_fps":                            "RTSS max FPS",
		"rtss_active_apps":                        "RTSS active apps",
		"rtss_foreground_pid":                     "RTSS foreground PID",
	}

	for name, item := range values {
		if label, ok := explicit[name]; ok {
			item.Label = label
			values[name] = item
			continue
		}
		if strings.TrimSpace(item.Label) == "" {
			item.Label = humanizeMonitorKey(name)
			values[name] = item
		}
	}

	for name, item := range values {
		index, suffix, ok := parseGoNativeIndexedWebMonitor(name, "go_native.net.")
		if ok {
			iface := resolveGoNativeIndexedMonitorAnchor(values, "go_native.net.", index, "interface")
			if iface == "" {
				iface = "net" + itoa(index)
			}
			switch suffix {
			case "upload":
				item.Label = "Net " + iface + " upload speed"
			case "download":
				item.Label = "Net " + iface + " download speed"
			case "ip":
				item.Label = "Net " + iface + " ip"
			case "interface":
				item.Label = "Net " + iface + " interface"
			}
			values[name] = item
			continue
		}

		index, suffix, ok = parseGoNativeIndexedWebMonitor(name, "go_native.disk.")
		if ok {
			diskName := resolveGoNativeIndexedMonitorAnchor(values, "go_native.disk.", index, "name")
			if diskName == "" {
				diskName = "disk" + itoa(index)
			}
			switch suffix {
			case "name":
				item.Label = "Disk " + diskName + " name"
			case "size":
				item.Label = "Disk " + diskName + " size"
			case "temp":
				item.Label = "Disk " + diskName + " temperature"
			}
			values[name] = item
		}
	}
}

func parseGoNativeIndexedWebMonitor(name, basePrefix string) (int, string, bool) {
	if !strings.HasPrefix(name, basePrefix) {
		return 0, "", false
	}
	rest := strings.TrimPrefix(name, basePrefix)
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return 0, "", false
	}
	indexText := parts[0]
	index := atoi(indexText)
	if index <= 0 {
		return 0, "", false
	}
	suffix := parts[1]
	if suffix == "" {
		return 0, "", false
	}
	return index, suffix, true
}

func resolveGoNativeIndexedMonitorAnchor(values map[string]WebMonitorSnapshotItem, basePrefix string, index int, anchor string) string {
	key := basePrefix + itoa(index) + "." + anchor
	item, ok := values[key]
	if !ok {
		return ""
	}
	text := strings.TrimSpace(item.Text)
	if text == "" || text == "-" {
		return ""
	}
	return text
}

func humanizeMonitorKey(key string) string {
	if strings.TrimSpace(key) == "" {
		return key
	}
	parts := strings.Split(key, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

func atoi(text string) int {
	value := 0
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			return 0
		}
		value = value*10 + int(ch-'0')
	}
	return value
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	v := value
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(buf[i:])
}
