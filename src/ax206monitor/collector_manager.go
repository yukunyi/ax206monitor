package main

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type CollectValue struct {
	Value     interface{}
	Unit      string
	Min       float64
	Max       float64
	Precision int
}

type BaseCollectItem struct {
	name      string
	label     string
	value     *CollectValue
	available bool
	enabled   bool
	mutex     sync.RWMutex
}

func NewBaseCollectItem(name, label string, min, max float64, unit string, precision int) *BaseCollectItem {
	return &BaseCollectItem{
		name:      name,
		label:     label,
		available: true,
		enabled:   true,
		value: &CollectValue{
			Value:     0.0,
			Unit:      unit,
			Min:       min,
			Max:       max,
			Precision: precision,
		},
	}
}

func (b *BaseCollectItem) GetName() string  { return b.name }
func (b *BaseCollectItem) GetLabel() string { return b.label }

func (b *BaseCollectItem) GetValue() *CollectValue {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.value == nil {
		return nil
	}
	copied := *b.value
	return &copied
}

func (b *BaseCollectItem) IsAvailable() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.available
}

func (b *BaseCollectItem) IsEnabled() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.enabled
}

func (b *BaseCollectItem) SetEnabled(enabled bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.enabled = enabled
}

func (b *BaseCollectItem) SetValue(value interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.value.Value = value
}

func (b *BaseCollectItem) SetUnit(unit string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.value.Unit = unit
}

func (b *BaseCollectItem) SetAvailable(available bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.available = available
}

func FormatCollectValue(value *CollectValue, showUnit bool, unitOverride string) string {
	numberText, unitText := FormatCollectValueParts(value, unitOverride)
	if !showUnit || unitText == "" {
		return numberText
	}
	return numberText + unitText
}

func FormatCollectValueParts(value *CollectValue, unitOverride string) (string, string) {
	if value == nil {
		return "N/A", ""
	}
	unit := value.Unit
	autoScale := true
	if unitOverride != "" {
		unit = unitOverride
		autoScale = false
	}
	switch v := value.Value.(type) {
	case string:
		return v, ""
	case float64, float32, int, int64, uint64:
		val := getFloat64Value(v)
		precision := value.Precision
		if autoScale {
			val, unit, precision = autoScaleUnitValue(val, unit, precision)
		}
		format := "%." + itoa(max(0, precision)) + "f"
		return fmt.Sprintf(format, val), unit
	default:
		return fmt.Sprintf("%v", value.Value), ""
	}
}

func autoScaleUnitValue(value float64, unit string, precision int) (float64, string, int) {
	trimmedUnit := strings.ToLower(strings.TrimSpace(unit))
	if trimmedUnit == "" {
		return value, unit, precision
	}
	family, index, scaleFactor, ok := getAutoScaleFamily(trimmedUnit)
	if !ok || index < 0 || index >= len(family) {
		return value, unit, precision
	}
	scaled := value
	absValue := math.Abs(value)
	if absValue > 0 {
		for absValue >= scaleFactor && index < len(family)-1 {
			scaled /= scaleFactor
			index++
			absValue = math.Abs(scaled)
		}
		for absValue > 0 && absValue < 1 && index > 0 {
			scaled *= scaleFactor
			index--
			absValue = math.Abs(scaled)
		}
	}
	scaledUnit := family[index]
	if strings.HasPrefix(unit, " ") {
		scaledUnit = " " + scaledUnit
	}
	return scaled, scaledUnit, autoScalePrecision(scaled, precision, scaledUnit != strings.TrimSpace(unit))
}

func getAutoScaleFamily(unit string) ([]string, int, float64, bool) {
	switch unit {
	case "b", "kb", "mb", "gb", "tb":
		return []string{"B", "KB", "MB", "GB", "TB"}, unitIndex(unit, []string{"b", "kb", "mb", "gb", "tb"}), 1024, true
	case "b/s", "kb/s", "mb/s", "gb/s", "tb/s":
		return []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}, unitIndex(unit, []string{"b/s", "kb/s", "mb/s", "gb/s", "tb/s"}), 1024, true
	case "kib", "mib", "gib", "tib":
		return []string{"B", "KiB", "MiB", "GiB", "TiB"}, unitIndex(unit, []string{"b", "kib", "mib", "gib", "tib"}), 1024, true
	case "kib/s", "mib/s", "gib/s", "tib/s":
		return []string{"B/s", "KiB/s", "MiB/s", "GiB/s", "TiB/s"}, unitIndex(unit, []string{"b/s", "kib/s", "mib/s", "gib/s", "tib/s"}), 1024, true
	case "hz", "khz", "mhz", "ghz", "thz":
		return []string{"Hz", "KHz", "MHz", "GHz", "THz"}, unitIndex(unit, []string{"hz", "khz", "mhz", "ghz", "thz"}), 1000, true
	default:
		return nil, -1, 0, false
	}
}

func unitIndex(unit string, family []string) int {
	for idx, value := range family {
		if unit == value {
			return idx
		}
	}
	return -1
}

func autoScalePrecision(value float64, defaultPrecision int, scaled bool) int {
	precision := max(0, defaultPrecision)
	if !scaled {
		return precision
	}
	absValue := math.Abs(value)
	switch {
	case absValue >= 100:
		return 0
	case absValue >= 10:
		return max(1, min(precision, 1))
	case absValue >= 1:
		return max(1, min(precision, 2))
	default:
		return max(2, precision)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getFloat64Value(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return 0.0
	}
}

func defaultMonitorWorkerCount() int {
	workers := runtime.NumCPU()
	if workers < 2 {
		return 2
	}
	if workers > 8 {
		return 8
	}
	return workers
}

func defaultMonitorQueueSize(workers int) int {
	queue := workers * 8
	if queue < 16 {
		return 16
	}
	if queue > 256 {
		return 256
	}
	return queue
}

type CollectorManagerStats struct {
	WorkerCount int `json:"worker_count"`
	QueueSize   int `json:"queue_size"`
	QueueLen    int `json:"queue_len"`

	AutoTune          bool    `json:"auto_tune"`
	AutoTuneIntervalS int     `json:"auto_tune_interval_sec"`
	AutoTuneSlowRate  float64 `json:"auto_tune_slow_rate"`
	AutoTuneStableRun int     `json:"auto_tune_stable_runs"`
	AutoTuneMaxScale  int     `json:"auto_tune_max_scale"`
	IntervalScale     int     `json:"interval_scale"`

	ScheduledTotal int64 `json:"scheduled_total"`
	DroppedTotal   int64 `json:"dropped_total"`
	SlowTotal      int64 `json:"slow_total"`
	CompletedTotal int64 `json:"completed_total"`

	LastWindowScheduled int64 `json:"last_window_scheduled"`
	LastWindowDropped   int64 `json:"last_window_dropped"`
	LastWindowSlow      int64 `json:"last_window_slow"`
	LastWindowCompleted int64 `json:"last_window_completed"`

	CollectorCount        int `json:"collector_count"`
	EnabledCollectorCount int `json:"enabled_collector_count"`
	ItemCount             int `json:"item_count"`
	EnabledItemCount      int `json:"enabled_item_count"`
}

type CollectorManager struct {
	collectors       map[string]Collector
	collectorOrder   []string
	collectorEnabled map[string]bool
	items            map[string]*CollectItem
	itemToCollector  map[string]string

	requiredSet map[string]struct{}
	requiredSig string
	modeFull    bool

	missingRetryAttempts int
	missingRetryNextAt   time.Time

	lastWindowScheduled int64
	lastWindowDropped   int64
	lastWindowSlow      int64
	lastWindowCompleted int64
	totalScheduled      int64
	totalDropped        int64
	totalSlow           int64
	totalCompleted      int64

	closed int32
	mutex  sync.RWMutex
	update sync.Mutex
}

func NewCollectorManager() *CollectorManager {
	return &CollectorManager{
		collectors:       make(map[string]Collector),
		collectorEnabled: make(map[string]bool),
		items:            make(map[string]*CollectItem),
		itemToCollector:  make(map[string]string),
		requiredSet:      make(map[string]struct{}),
	}
}

func (m *CollectorManager) RegisterCollector(c Collector) {
	if c == nil {
		return
	}
	name := strings.TrimSpace(c.Name())
	if name == "" {
		return
	}
	m.mutex.Lock()
	if _, exists := m.collectors[name]; !exists {
		m.collectorOrder = append(m.collectorOrder, name)
	}
	m.collectors[name] = c
	if _, exists := m.collectorEnabled[name]; !exists {
		m.collectorEnabled[name] = true
	}
	m.mutex.Unlock()
}

type namedCollector struct {
	name      string
	collector Collector
}

func (m *CollectorManager) snapshotCollectors() []namedCollector {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	result := make([]namedCollector, 0, len(m.collectorOrder))
	for _, name := range m.collectorOrder {
		collector := m.collectors[name]
		if collector == nil {
			continue
		}
		result = append(result, namedCollector{name: name, collector: collector})
	}
	return result
}

func (m *CollectorManager) discoverFromCollectors(collectors []namedCollector) {
	if len(collectors) == 0 {
		return
	}
	discoveredItems := make(map[string]*CollectItem)
	discoveredOwners := make(map[string]string)
	for _, entry := range collectors {
		for key, item := range entry.collector.GetAllItems() {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" || item == nil {
				continue
			}
			discoveredItems[trimmed] = item
			discoveredOwners[trimmed] = entry.name
		}
	}
	if len(discoveredItems) == 0 {
		return
	}
	m.mutex.Lock()
	for key, item := range discoveredItems {
		m.items[key] = item
		m.itemToCollector[key] = discoveredOwners[key]
	}
	m.mutex.Unlock()
}

func (m *CollectorManager) discoverAll() {
	m.discoverFromCollectors(m.snapshotCollectors())
}

func signatureOfNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	uniq := make([]string, 0, len(set))
	for key := range set {
		uniq = append(uniq, key)
	}
	sort.Strings(uniq)
	return strings.Join(uniq, "\n")
}

func (m *CollectorManager) setRequiredItems(names []string) {
	sig := signatureOfNames(names)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if sig == m.requiredSig {
		return
	}
	m.requiredSig = sig
	m.requiredSet = make(map[string]struct{})
	if sig != "" {
		for _, name := range strings.Split(sig, "\n") {
			if name != "" {
				m.requiredSet[name] = struct{}{}
			}
		}
	}
	m.missingRetryAttempts = 0
	m.missingRetryNextAt = time.Time{}
}

func (m *CollectorManager) requiredItemsSnapshot() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	result := make([]string, 0, len(m.requiredSet))
	for name := range m.requiredSet {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (m *CollectorManager) setItemEnabledStatesLocked() {
	for name, item := range m.items {
		if item == nil {
			continue
		}
		collectorName := m.itemToCollector[name]
		collectorEnabled := m.collectorEnabled[collectorName]
		enabled := collectorEnabled
		if !m.modeFull {
			_, required := m.requiredSet[name]
			enabled = collectorEnabled && required
		}
		item.SetEnabled(enabled)
	}
}

func (m *CollectorManager) hasEnabledItemsLocked(collectorName string) bool {
	for itemName, owner := range m.itemToCollector {
		if owner != collectorName {
			continue
		}
		item := m.items[itemName]
		if item != nil && item.IsEnabled() {
			return true
		}
	}
	return false
}

func (m *CollectorManager) maybeRetryDiscover(now time.Time) {
	m.mutex.Lock()
	if len(m.requiredSet) == 0 {
		m.mutex.Unlock()
		return
	}
	missing := false
	for key := range m.requiredSet {
		if _, exists := m.items[key]; !exists {
			missing = true
			break
		}
	}
	if !missing {
		m.missingRetryAttempts = 0
		m.missingRetryNextAt = time.Time{}
		m.mutex.Unlock()
		return
	}
	if m.missingRetryAttempts >= 10 {
		m.mutex.Unlock()
		return
	}
	if !m.missingRetryNextAt.IsZero() && now.Before(m.missingRetryNextAt) {
		m.mutex.Unlock()
		return
	}
	collectors := make([]namedCollector, 0, len(m.collectorOrder))
	for _, name := range m.collectorOrder {
		if collector := m.collectors[name]; collector != nil {
			collectors = append(collectors, namedCollector{name: name, collector: collector})
		}
	}
	m.mutex.Unlock()

	m.discoverFromCollectors(collectors)

	m.mutex.Lock()
	defer m.mutex.Unlock()
	resolved := true
	for key := range m.requiredSet {
		if _, exists := m.items[key]; !exists {
			resolved = false
			break
		}
	}
	if resolved {
		m.missingRetryAttempts = 0
		m.missingRetryNextAt = time.Time{}
		return
	}
	m.missingRetryAttempts++
	m.missingRetryNextAt = now.Add(time.Minute)
}

func (m *CollectorManager) updateOnce() error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return nil
	}
	m.update.Lock()
	defer m.update.Unlock()

	var firstErr error
	cycleScheduled := int64(0)
	cycleDropped := int64(0)
	cycleSlow := int64(0)
	cycleCompleted := int64(0)

	m.mutex.Lock()
	m.setItemEnabledStatesLocked()
	order := append([]string(nil), m.collectorOrder...)
	m.mutex.Unlock()

	for _, collectorName := range order {
		m.mutex.RLock()
		enabled := m.collectorEnabled[collectorName]
		hasEnabled := m.hasEnabledItemsLocked(collectorName)
		collector := m.collectors[collectorName]
		m.mutex.RUnlock()
		if !enabled || !hasEnabled || collector == nil {
			continue
		}
		cycleScheduled++
		started := time.Now()
		err := collector.UpdateItems()
		if time.Since(started) > 500*time.Millisecond {
			cycleSlow++
		}
		if err != nil {
			cycleDropped++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		cycleCompleted++
	}

	m.mutex.Lock()
	m.lastWindowScheduled = cycleScheduled
	m.lastWindowDropped = cycleDropped
	m.lastWindowSlow = cycleSlow
	m.lastWindowCompleted = cycleCompleted
	m.totalScheduled += cycleScheduled
	m.totalDropped += cycleDropped
	m.totalSlow += cycleSlow
	m.totalCompleted += cycleCompleted
	m.mutex.Unlock()
	m.maybeRetryDiscover(time.Now())

	return firstErr
}

func (m *CollectorManager) Get(name string) *CollectItem {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.items[name]
}

func (m *CollectorManager) GetAll() map[string]*CollectItem {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	result := make(map[string]*CollectItem, len(m.items))
	for key, item := range m.items {
		result[key] = item
	}
	return result
}

func (m *CollectorManager) Update(requiredItems []string) error {
	m.modeFull = false
	m.setRequiredItems(requiredItems)
	return m.updateOnce()
}

func (m *CollectorManager) UpdateAll() error {
	m.modeFull = true
	return m.updateOnce()
}

func (m *CollectorManager) SetCollectorEnabled(name string, enabled bool) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, exists := m.collectors[trimmed]; !exists {
		return false
	}
	m.collectorEnabled[trimmed] = enabled
	return true
}

func (m *CollectorManager) CollectorStates() map[string]bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	states := make(map[string]bool, len(m.collectorEnabled))
	for name, enabled := range m.collectorEnabled {
		states[name] = enabled
	}
	return states
}

func (m *CollectorManager) CollectorNames() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	names := append([]string(nil), m.collectorOrder...)
	sort.Strings(names)
	return names
}

func (m *CollectorManager) Stats() CollectorManagerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	enabledCollectorCount := 0
	for _, enabled := range m.collectorEnabled {
		if enabled {
			enabledCollectorCount++
		}
	}
	enabledItemCount := 0
	for _, item := range m.items {
		if item != nil && item.IsEnabled() {
			enabledItemCount++
		}
	}
	return CollectorManagerStats{
		WorkerCount: int(enabledCollectorCount),
		QueueSize:   0,
		QueueLen:    0,

		AutoTune:          false,
		AutoTuneIntervalS: 0,
		AutoTuneSlowRate:  0,
		AutoTuneStableRun: 0,
		AutoTuneMaxScale:  0,
		IntervalScale:     1,

		ScheduledTotal: m.totalScheduled,
		DroppedTotal:   m.totalDropped,
		SlowTotal:      m.totalSlow,
		CompletedTotal: m.totalCompleted,

		LastWindowScheduled: m.lastWindowScheduled,
		LastWindowDropped:   m.lastWindowDropped,
		LastWindowSlow:      m.lastWindowSlow,
		LastWindowCompleted: m.lastWindowCompleted,

		CollectorCount:        len(m.collectors),
		EnabledCollectorCount: enabledCollectorCount,
		ItemCount:             len(m.items),
		EnabledItemCount:      enabledItemCount,
	}
}

func (m *CollectorManager) Close() {
	atomic.StoreInt32(&m.closed, 1)
}

type CollectItemConfig struct {
	Name     string
	Required bool
}

type CollectorManagerConfig struct {
	Items []CollectItemConfig
}

func getCollectorManagerConfig() *CollectorManagerConfig {
	names := []string{
		"go_native.cpu.usage",
		"go_native.cpu.temp",
		"go_native.cpu.freq",
		"go_native.cpu.model",
		"go_native.cpu.cores",
		"go_native.memory.usage",
		"go_native.memory.used",
		"go_native.memory.total",
		"go_native.memory.usage_text",
		"go_native.memory.usage_progress",
		"go_native.memory.swap_usage",
		"go_native.system.load_avg",
		"go_native.system.current_time",
	}
	items := make([]CollectItemConfig, 0, len(names))
	for _, name := range names {
		items = append(items, CollectItemConfig{Name: name, Required: true})
	}
	return &CollectorManagerConfig{Items: items}
}

func initializeCollectors(manager *CollectorManager, cfg *MonitorConfig) {
	if manager == nil {
		return
	}
	if cfg == nil {
		cfg = &MonitorConfig{}
	}
	registerCollectorWithConfig(manager, cfg, NewGoNativeCPUCollector(), true)
	registerCollectorWithConfig(manager, cfg, NewGoNativeMemoryCollector(), true)
	registerCollectorWithConfig(manager, cfg, NewGoNativeSystemCollector(), true)
	registerCollectorWithConfig(manager, cfg, NewGoNativeDiskCollector(manager.requiredItemsSnapshot), true)
	registerCollectorWithConfig(manager, cfg, NewGoNativeNetworkCollector(manager.requiredItemsSnapshot), true)
	if cc := NewCoolerControlCollector(cfg); cc != nil {
		registerCollectorWithConfig(manager, cfg, cc, false)
	}
	if lhm := NewLibreHardwareMonitorCollector(cfg); lhm != nil {
		registerCollectorWithConfig(manager, cfg, lhm, false)
	}
	if rtss := NewRTSSCollector(cfg); rtss != nil {
		registerCollectorWithConfig(manager, cfg, rtss, false)
	}
	registerCollectorWithConfig(manager, cfg, NewCustomCollector(cfg, manager.Get), true)
	manager.discoverAll()
}

func registerCollectorWithConfig(manager *CollectorManager, cfg *MonitorConfig, collector Collector, defaultEnabled bool) {
	if manager == nil || collector == nil {
		return
	}
	manager.RegisterCollector(collector)
	name := collector.Name()
	manager.collectorEnabled[name] = true
	if cfg != nil {
		manager.collectorEnabled[name] = cfg.IsCollectorEnabled(name, defaultEnabled)
	}
}

func newCollectorManagerFromConfig(cfg *MonitorConfig, requiredItems []string) *CollectorManager {
	manager := NewCollectorManager()
	manager.setRequiredItems(requiredItems)
	initializeCollectors(manager, cfg)
	_ = manager.UpdateAll()
	return manager
}

var (
	globalCollectorManager *CollectorManager
	globalCollectorConfig  *MonitorConfig
	globalCollectorMu      sync.Mutex
)

func GetCollectorManager() *CollectorManager {
	globalCollectorMu.Lock()
	defer globalCollectorMu.Unlock()
	if globalCollectorManager == nil {
		globalCollectorManager = newCollectorManagerFromConfig(globalCollectorConfig, nil)
	}
	return globalCollectorManager
}

func GetCollectorManagerWithConfig(requiredItems []string, networkInterface string) *CollectorManager {
	_ = networkInterface
	globalCollectorMu.Lock()
	defer globalCollectorMu.Unlock()
	if globalCollectorManager == nil {
		globalCollectorManager = newCollectorManagerFromConfig(globalCollectorConfig, requiredItems)
		return globalCollectorManager
	}
	globalCollectorManager.setRequiredItems(requiredItems)
	return globalCollectorManager
}

func SetGlobalCollectorConfig(config *MonitorConfig) {
	globalCollectorMu.Lock()
	defer globalCollectorMu.Unlock()
	globalCollectorConfig = config
}

func GetGlobalCollectorConfig() *MonitorConfig {
	globalCollectorMu.Lock()
	defer globalCollectorMu.Unlock()
	return globalCollectorConfig
}

func ResetGlobalCollectorManager() {
	globalCollectorMu.Lock()
	oldManager := globalCollectorManager
	globalCollectorManager = nil
	globalCollectorMu.Unlock()
	if oldManager != nil {
		oldManager.Close()
	}
}
