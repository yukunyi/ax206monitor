package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	MonitorRuntime *MonitorRegistryStats             `json:"monitor_runtime,omitempty"`
}

type WebRuntime struct {
	mu            sync.RWMutex
	applyMu       sync.Mutex
	renderMu      sync.Mutex
	fontCache     *FontCache
	config        *MonitorConfig
	required      []string
	registry      *MonitorRegistry
	renderManager *RenderManager
	outputManager *OutputManager
	outputHasMem  bool
	idleProvider  func() (*MonitorConfig, error)
	previewOutput *MemImgOutputHandler
	lastProbeCC   string
	lastProbeLHM  string

	activityMu   sync.RWMutex
	lastActivity time.Time
	modeFull     bool
	updatedAt    time.Time
	realtimeConn int32

	stopOnce sync.Once
	stopCh   chan struct{}
	stopped  chan struct{}
}

func NewWebRuntime(cfg *MonitorConfig) (*WebRuntime, error) {
	fontCache, err := loadFontCache()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize web runtime fonts: %w", err)
	}

	runtime := &WebRuntime{
		fontCache:     fontCache,
		previewOutput: NewMemImgOutputHandler(),
		stopCh:        make(chan struct{}),
		stopped:       make(chan struct{}),
	}

	if err := runtime.applyConfigInternal(cfg, false); err != nil {
		return nil, err
	}

	go runtime.loop()
	return runtime, nil
}

func (r *WebRuntime) Touch() {
	r.activityMu.Lock()
	r.lastActivity = time.Now()
	r.activityMu.Unlock()
}

func (r *WebRuntime) ApplyConfig(cfg *MonitorConfig) error {
	if err := r.applyConfigInternal(cfg, false); err != nil {
		return err
	}
	_ = r.renderOnce(true)
	return nil
}

func (r *WebRuntime) ApplyPreviewConfig(cfg *MonitorConfig) error {
	// Preview edit should not rebuild USB outputs on every minor config change.
	if err := r.applyConfigInternal(cfg, false); err != nil {
		return err
	}
	return r.renderOnce(true)
}

func (r *WebRuntime) SetRealtimeConnectionCount(count int) {
	if count < 0 {
		count = 0
	}
	atomic.StoreInt32(&r.realtimeConn, int32(count))
}

func (r *WebRuntime) SetIdleConfigProvider(provider func() (*MonitorConfig, error)) {
	r.mu.Lock()
	r.idleProvider = provider
	r.mu.Unlock()
}

func (r *WebRuntime) renderOnce(forceFull bool) error {
	r.renderMu.Lock()
	defer r.renderMu.Unlock()

	modeFull := forceFull || r.isFullMode()
	r.setMode(modeFull)

	cfg, required, registry, renderManager, outputManager, outputHasMem := r.getRuntimeRefs()
	if cfg == nil || registry == nil || renderManager == nil || outputManager == nil {
		return nil
	}

	if modeFull {
		_ = registry.UpdateAll()
	} else {
		_ = registry.Update(required)
	}

	img, err := renderManager.Render(cfg)
	if err != nil {
		return err
	}
	if modeFull && !outputHasMem {
		if err := r.previewOutput.Output(img); err != nil {
			return err
		}
	}
	if err := outputManager.Output(img); err != nil {
		return err
	}

	r.setUpdatedAt(time.Now())
	return nil
}

func (r *WebRuntime) applyConfigInternal(cfg *MonitorConfig, forceMemImg bool) error {
	r.applyMu.Lock()
	defer r.applyMu.Unlock()

	// Prevent output manager swap/close from racing with ongoing render output.
	r.renderMu.Lock()
	defer r.renderMu.Unlock()

	configCopy := cloneMonitorConfig(cfg)
	normalizeMonitorConfig(configCopy)

	SetGlobalMonitorConfig(configCopy)
	ResetGlobalMonitorRegistry()
	initializeCache()

	required := getRequiredMonitors(configCopy)
	registry := GetMonitorRegistryWithConfig(required, configCopy.GetNetworkInterface())
	renderManager := NewRenderManager(r.fontCache, registry)

	targetOutputTypes := resolveOutputTypes(configCopy, forceMemImg)
	oldOutputTypes := resolveOutputTypes(r.CurrentConfig(), forceMemImg)
	reuseOutputManager := false

	r.mu.RLock()
	if r.outputManager != nil && sameStringSlice(targetOutputTypes, oldOutputTypes) {
		reuseOutputManager = true
	}
	r.mu.RUnlock()

	var outputManager *OutputManager
	var outputTypes []string
	if reuseOutputManager {
		r.mu.RLock()
		outputManager = r.outputManager
		r.mu.RUnlock()
		outputTypes = append([]string(nil), targetOutputTypes...)
	} else {
		var builtTypes []string
		outputManager, builtTypes = buildOutputManager(configCopy, forceMemImg)
		outputTypes = append([]string(nil), builtTypes...)
	}

	outputHasMem := false
	for _, typeName := range outputTypes {
		if typeName == outputTypeMemImg {
			outputHasMem = true
			break
		}
	}

	r.mu.Lock()
	oldOutputManager := r.outputManager
	r.config = configCopy
	r.required = required
	r.registry = registry
	r.renderManager = renderManager
	r.outputManager = outputManager
	r.outputHasMem = outputHasMem
	r.mu.Unlock()

	if !reuseOutputManager && oldOutputManager != nil {
		oldOutputManager.Close()
	}
	r.maybeProbeDataSources(configCopy)
	return nil
}

func sameStringSlice(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r *WebRuntime) maybeProbeDataSources(cfg *MonitorConfig) {
	ccURL := cfg.GetCoolerControlURL()
	lhmURL := cfg.GetLibreHardwareMonitorURL()

	shouldProbeCC := false
	shouldProbeLHM := false

	r.mu.Lock()
	if ccURL != "" && ccURL != r.lastProbeCC {
		r.lastProbeCC = ccURL
		shouldProbeCC = true
	}
	if lhmURL != "" && lhmURL != r.lastProbeLHM {
		r.lastProbeLHM = lhmURL
		shouldProbeLHM = true
	}
	r.mu.Unlock()

	if shouldProbeCC {
		username := cfg.GetCoolerControlUsername()
		password := cfg.CoolerControlPassword
		go func(url, user, pass string) {
			options, err := GetCoolerControlClient(url, user, pass).ListMonitorOptions()
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
		}(ccURL, username, password)
	}

	if shouldProbeLHM {
		go func(url string) {
			items, err := GetLibreHardwareMonitorClient(url).ListMonitorOptions()
			if err != nil {
				logWarnModule("web", "librehardwaremonitor probe failed (url=%s): %v", url, err)
				return
			}
			logInfoModule("web", "librehardwaremonitor probe success (url=%s, monitors=%d)", url, len(items))
		}(lhmURL)
	}
}

func (r *WebRuntime) loop() {
	defer close(r.stopped)
	ticker := time.NewTicker(webTickerInterval)
	defer ticker.Stop()

	lastRender := time.Time{}
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

		refreshInterval := time.Duration(cfg.RefreshInterval) * time.Millisecond
		if refreshInterval <= 0 {
			refreshInterval = time.Second
		}
		if !lastRender.IsZero() && time.Since(lastRender) < refreshInterval {
			continue
		}

		if err := r.renderOnce(false); err != nil {
			logDebugModule("web", "render runtime image failed: %v", err)
			continue
		}

		lastRender = time.Now()
	}
}

func (r *WebRuntime) Snapshot() WebSnapshotResponse {
	modeFull := r.isFullMode()
	r.setMode(modeFull)

	cfg, required, registry, _, _, _ := r.getRuntimeRefs()
	values := make(map[string]WebMonitorSnapshotItem)
	if registry == nil {
		return WebSnapshotResponse{
			Mode:      "required",
			UpdatedAt: time.Now().Format(time.RFC3339),
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
				item.Text = FormatMonitorValue(value, true, "")
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
		UpdatedAt:      updatedAt.Format(time.RFC3339),
		Monitors:       names,
		Values:         values,
		MonitorRuntime: &monitorStats,
	}
}

func (r *WebRuntime) MonitorStats() *MonitorRegistryStats {
	_, _, registry, _, _, _ := r.getRuntimeRefs()
	if registry == nil {
		return nil
	}
	stats := registry.Stats()
	return &stats
}

func (r *WebRuntime) AllMonitorNames() []string {
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

func (r *WebRuntime) CurrentConfig() *MonitorConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.config == nil {
		return nil
	}
	return cloneMonitorConfig(r.config)
}

func (r *WebRuntime) Close() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
		<-r.stopped
	})

	r.mu.Lock()
	oldOutputManager := r.outputManager
	oldRegistry := r.registry
	r.outputManager = nil
	r.registry = nil
	r.renderManager = nil
	r.config = nil
	r.required = nil
	r.outputHasMem = false
	r.mu.Unlock()

	if oldOutputManager != nil {
		oldOutputManager.Close()
	}
	if oldRegistry != nil {
		oldRegistry.Close()
	}
	ResetGlobalMonitorRegistry()
}

func (r *WebRuntime) getRuntimeRefs() (*MonitorConfig, []string, *MonitorRegistry, *RenderManager, *OutputManager, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	requiredCopy := append([]string(nil), r.required...)
	return r.config, requiredCopy, r.registry, r.renderManager, r.outputManager, r.outputHasMem
}

func (r *WebRuntime) restoreIdleConfig() {
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
	_ = r.renderOnce(false)
}

func (r *WebRuntime) isFullMode() bool {
	return atomic.LoadInt32(&r.realtimeConn) > 0
}

func (r *WebRuntime) setMode(modeFull bool) {
	r.activityMu.Lock()
	r.modeFull = modeFull
	r.activityMu.Unlock()
}

func (r *WebRuntime) setUpdatedAt(updatedAt time.Time) {
	r.activityMu.Lock()
	r.updatedAt = updatedAt
	r.activityMu.Unlock()
}

func (r *WebRuntime) getUpdatedAt() time.Time {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()
	return r.updatedAt
}

func optimizeWebSnapshotLabels(values map[string]WebMonitorSnapshotItem) {
	if len(values) == 0 {
		return
	}

	explicit := map[string]string{
		"cpu_usage":             "CPU usage",
		"cpu_temp":              "CPU temperature",
		"cpu_freq":              "CPU frequency",
		"cpu_model":             "CPU model",
		"cpu_cores":             "CPU cores",
		"memory_usage":          "Memory usage",
		"memory_used":           "Memory used",
		"memory_total":          "Memory total",
		"memory_usage_text":     "Memory usage detail",
		"memory_usage_progress": "Memory usage progress",
		"swap_usage":            "Swap usage",
		"load_avg":              "System load average",
		"current_time":          "Current time",
		"rtss_fps":              "RTSS FPS",
		"rtss_frametime_ms":     "RTSS frame time",
		"rtss_max_fps":          "RTSS max FPS",
		"rtss_active_apps":      "RTSS active apps",
		"rtss_foreground_pid":   "RTSS foreground PID",
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
		index, suffix, ok := parseIndexedWebMonitor(name, "net")
		if ok {
			iface := resolveIndexedMonitorAnchor(values, "net", index, "interface")
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

		index, suffix, ok = parseIndexedWebMonitor(name, "disk")
		if ok {
			diskName := resolveIndexedMonitorAnchor(values, "disk", index, "name")
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

func parseIndexedWebMonitor(name, prefix string) (int, string, bool) {
	if !strings.HasPrefix(name, prefix) {
		return 0, "", false
	}
	underscore := strings.IndexByte(name, '_')
	if underscore <= len(prefix) {
		return 0, "", false
	}
	indexText := name[len(prefix):underscore]
	index := atoi(indexText)
	if index <= 0 {
		return 0, "", false
	}
	suffix := name[underscore+1:]
	if suffix == "" {
		return 0, "", false
	}
	return index, suffix, true
}

func resolveIndexedMonitorAnchor(values map[string]WebMonitorSnapshotItem, prefix string, index int, anchor string) string {
	key := prefix + itoa(index) + "_" + anchor
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
