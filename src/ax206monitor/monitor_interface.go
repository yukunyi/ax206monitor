package main

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type MonitorItemType int

const (
	TypeUsage MonitorItemType = iota
	TypeTemperature
	TypeFrequency
	TypeMemory
	TypeNetwork
	TypeString
	TypeInt
)

type MonitorValue struct {
	Value     interface{}
	Unit      string
	Min       float64
	Max       float64
	Precision int
}

type MonitorItem interface {
	GetName() string
	GetLabel() string
	Update() error
	GetValue() *MonitorValue
	IsAvailable() bool
}

type BaseMonitorItem struct {
	name      string
	label     string
	value     *MonitorValue
	available bool
	mutex     sync.RWMutex
}

func NewBaseMonitorItem(name, label string, min, max float64, unit string, precision int) *BaseMonitorItem {
	return &BaseMonitorItem{
		name:      name,
		label:     label,
		available: true,
		value: &MonitorValue{
			Value:     0.0,
			Unit:      unit,
			Min:       min,
			Max:       max,
			Precision: precision,
		},
	}
}

func (b *BaseMonitorItem) GetName() string  { return b.name }
func (b *BaseMonitorItem) GetLabel() string { return b.label }
func (b *BaseMonitorItem) GetValue() *MonitorValue {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.value == nil {
		return nil
	}
	copied := *b.value
	return &copied
}
func (b *BaseMonitorItem) IsAvailable() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.available
}
func (b *BaseMonitorItem) SetValue(value interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.value.Value = value
}

func (b *BaseMonitorItem) SetUnit(unit string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.value.Unit = unit
}

func (b *BaseMonitorItem) SetAvailable(available bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.available = available
}

func FormatMonitorValue(value *MonitorValue, showUnit bool, unitOverride string) string {
	numberText, unitText := FormatMonitorValueParts(value, unitOverride)
	if !showUnit || unitText == "" {
		return numberText
	}
	return numberText + unitText
}

func FormatMonitorValueParts(value *MonitorValue, unitOverride string) (string, string) {
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
		format := fmt.Sprintf("%%.%df", max(0, precision))
		return fmt.Sprintf(format, val), unit
	default:
		return fmt.Sprintf("%v", value.Value), ""
	}
}

func autoScaleUnitValue(value float64, unit string, precision int) (float64, string, int) {
	trimmedUnit := strings.TrimSpace(unit)
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

	scaledPrecision := autoScalePrecision(scaled, precision, scaledUnit != trimmedUnit)
	return scaled, scaledUnit, scaledPrecision
}

func getAutoScaleFamily(unit string) ([]string, int, float64, bool) {
	lower := strings.ToLower(strings.TrimSpace(unit))
	switch lower {
	case "b", "kb", "mb", "gb", "tb":
		return []string{"B", "KB", "MB", "GB", "TB"}, unitIndex(lower, []string{"b", "kb", "mb", "gb", "tb"}), 1024, true
	case "b/s", "kb/s", "mb/s", "gb/s", "tb/s":
		return []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}, unitIndex(lower, []string{"b/s", "kb/s", "mb/s", "gb/s", "tb/s"}), 1024, true
	case "kib", "mib", "gib", "tib":
		return []string{"B", "KiB", "MiB", "GiB", "TiB"}, unitIndex(lower, []string{"b", "kib", "mib", "gib", "tib"}), 1024, true
	case "kib/s", "mib/s", "gib/s", "tib/s":
		return []string{"B/s", "KiB/s", "MiB/s", "GiB/s", "TiB/s"}, unitIndex(lower, []string{"b/s", "kib/s", "mib/s", "gib/s", "tib/s"}), 1024, true
	case "hz", "khz", "mhz", "ghz", "thz":
		return []string{"Hz", "KHz", "MHz", "GHz", "THz"}, unitIndex(lower, []string{"hz", "khz", "mhz", "ghz", "thz"}), 1000, true
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

type monitorRunState struct {
	running     int32
	lastStart   int64 // unix nano
	nextAllowed int64 // unix nano
	lastDropped int64 // unix nano
}

type monitorUpdateTask struct {
	item  MonitorItem
	state *monitorRunState
}

type MonitorRegistryOptions struct {
	WorkerCount       int
	QueueSize         int
	AutoTune          bool
	AutoTuneInterval  time.Duration
	AutoTuneSlowRate  float64
	AutoTuneStableRun int
	AutoTuneMaxScale  int
}

type MonitorRegistryStats struct {
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
}

type MonitorRegistry struct {
	items       map[string]MonitorItem
	mutex       sync.RWMutex
	states      map[string]*monitorRunState
	stateMu     sync.RWMutex
	taskCh      chan monitorUpdateTask
	stopCh      chan struct{}
	workerCount int
	queueSize   int
	workersWg   sync.WaitGroup

	autoTune           bool
	autoTuneInterval   time.Duration
	autoTuneSlowRate   float64
	autoTuneStableRun  int32
	autoTuneMaxScale   int32
	intervalScale      int32
	adaptiveStableRuns int32
	scheduledCount     int64
	droppedCount       int64
	slowCount          int64
	completedCount     int64

	totalScheduled int64
	totalDropped   int64
	totalSlow      int64
	totalCompleted int64

	lastWindowScheduled int64
	lastWindowDropped   int64
	lastWindowSlow      int64
	lastWindowCompleted int64
	closed              int32
}

func NewMonitorRegistry() *MonitorRegistry {
	return newMonitorRegistryWithOptions(defaultMonitorRegistryOptions())
}

func newMonitorRegistry(workerCount, queueSize int, autoTune bool) *MonitorRegistry {
	opts := defaultMonitorRegistryOptions()
	opts.WorkerCount = workerCount
	opts.QueueSize = queueSize
	opts.AutoTune = autoTune
	return newMonitorRegistryWithOptions(opts)
}

func defaultMonitorRegistryOptions() MonitorRegistryOptions {
	workers := defaultMonitorWorkerCount()
	return MonitorRegistryOptions{
		WorkerCount:       workers,
		QueueSize:         defaultMonitorQueueSize(workers),
		AutoTune:          true,
		AutoTuneInterval:  5 * time.Second,
		AutoTuneSlowRate:  0.20,
		AutoTuneStableRun: 3,
		AutoTuneMaxScale:  8,
	}
}

func normalizeMonitorRegistryOptions(opts MonitorRegistryOptions) MonitorRegistryOptions {
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = 1
	}
	if opts.QueueSize <= 0 {
		opts.QueueSize = defaultMonitorQueueSize(opts.WorkerCount)
	}
	if opts.QueueSize < opts.WorkerCount {
		opts.QueueSize = opts.WorkerCount
	}
	if opts.AutoTuneInterval <= 0 {
		opts.AutoTuneInterval = 5 * time.Second
	}
	if opts.AutoTuneSlowRate <= 0 {
		opts.AutoTuneSlowRate = 0.20
	}
	if opts.AutoTuneSlowRate > 1 {
		opts.AutoTuneSlowRate = 1
	}
	if opts.AutoTuneStableRun <= 0 {
		opts.AutoTuneStableRun = 3
	}
	if opts.AutoTuneStableRun > 60 {
		opts.AutoTuneStableRun = 60
	}
	if opts.AutoTuneMaxScale <= 0 {
		opts.AutoTuneMaxScale = 8
	}
	if opts.AutoTuneMaxScale > 64 {
		opts.AutoTuneMaxScale = 64
	}
	return opts
}

func newMonitorRegistryWithOptions(options MonitorRegistryOptions) *MonitorRegistry {
	opts := normalizeMonitorRegistryOptions(options)
	registry := &MonitorRegistry{
		items:             make(map[string]MonitorItem),
		states:            make(map[string]*monitorRunState),
		taskCh:            make(chan monitorUpdateTask, opts.QueueSize),
		stopCh:            make(chan struct{}),
		workerCount:       opts.WorkerCount,
		queueSize:         opts.QueueSize,
		autoTune:          opts.AutoTune,
		autoTuneInterval:  opts.AutoTuneInterval,
		autoTuneSlowRate:  opts.AutoTuneSlowRate,
		autoTuneStableRun: int32(opts.AutoTuneStableRun),
		autoTuneMaxScale:  int32(opts.AutoTuneMaxScale),
	}
	atomic.StoreInt32(&registry.intervalScale, 1)
	registry.startWorkers()
	registry.startAdaptiveTuner()
	return registry
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

func (r *MonitorRegistry) startWorkers() {
	for i := 0; i < r.workerCount; i++ {
		r.workersWg.Add(1)
		go r.workerLoop()
	}
}

func (r *MonitorRegistry) startAdaptiveTuner() {
	if !r.autoTune {
		return
	}
	r.workersWg.Add(1)
	go func() {
		defer r.workersWg.Done()
		ticker := time.NewTicker(r.autoTuneInterval)
		defer ticker.Stop()
		for {
			select {
			case <-r.stopCh:
				return
			case <-ticker.C:
				r.tuneOnce()
			}
		}
	}()
}

func (r *MonitorRegistry) tuneOnce() {
	scheduled := atomic.SwapInt64(&r.scheduledCount, 0)
	dropped := atomic.SwapInt64(&r.droppedCount, 0)
	slow := atomic.SwapInt64(&r.slowCount, 0)
	completed := atomic.SwapInt64(&r.completedCount, 0)
	atomic.StoreInt64(&r.lastWindowScheduled, scheduled)
	atomic.StoreInt64(&r.lastWindowDropped, dropped)
	atomic.StoreInt64(&r.lastWindowSlow, slow)
	atomic.StoreInt64(&r.lastWindowCompleted, completed)
	if scheduled == 0 {
		return
	}
	r.adjustAdaptiveScale(scheduled, dropped, slow, completed)
}

func (r *MonitorRegistry) adjustAdaptiveScale(scheduled, dropped, slow, completed int64) {
	current := atomic.LoadInt32(&r.intervalScale)
	if current <= 0 {
		current = 1
		atomic.StoreInt32(&r.intervalScale, current)
	}

	highPressure := dropped > 0
	if !highPressure && completed > 0 {
		highPressure = float64(slow)/float64(completed) >= r.autoTuneSlowRate
	}

	if highPressure {
		atomic.StoreInt32(&r.adaptiveStableRuns, 0)
		if current < r.autoTuneMaxScale && atomic.CompareAndSwapInt32(&r.intervalScale, current, current+1) {
			logInfo(
				"Monitor adaptive tuning: scale up to x%d (scheduled=%d dropped=%d slow=%d completed=%d)",
				current+1, scheduled, dropped, slow, completed,
			)
		}
		return
	}

	stableRuns := atomic.AddInt32(&r.adaptiveStableRuns, 1)
	if stableRuns < r.autoTuneStableRun || current <= 1 {
		return
	}
	if atomic.CompareAndSwapInt32(&r.intervalScale, current, current-1) {
		atomic.StoreInt32(&r.adaptiveStableRuns, 0)
		logInfo(
			"Monitor adaptive tuning: scale down to x%d (scheduled=%d dropped=%d slow=%d completed=%d)",
			current-1, scheduled, dropped, slow, completed,
		)
	}
}

func (r *MonitorRegistry) workerLoop() {
	defer r.workersWg.Done()
	for {
		select {
		case <-r.stopCh:
			return
		default:
		}

		select {
		case <-r.stopCh:
			return
		case task := <-r.taskCh:
			if task.item == nil || task.state == nil {
				continue
			}
			r.executeUpdateTask(task.item, task.state)
		}
	}
}

func (r *MonitorRegistry) Close() {
	if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		return
	}
	close(r.stopCh)
	waitDone := make(chan struct{})
	go func() {
		r.workersWg.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		logWarn("Monitor registry close timed out, leaving running workers in background")
	}
}

func (r *MonitorRegistry) executeUpdateTask(m MonitorItem, state *monitorRunState) {
	defer func() {
		if rec := recover(); rec != nil {
			logWarn("Monitor '%s' update panic: %v", m.GetName(), rec)
		}
		atomic.StoreInt32(&state.running, 0)
	}()
	start := time.Now()
	_ = m.Update()
	elapsed := time.Since(start)
	atomic.AddInt64(&r.completedCount, 1)
	atomic.AddInt64(&r.totalCompleted, 1)
	if elapsed > 500*time.Millisecond {
		atomic.AddInt64(&r.slowCount, 1)
		atomic.AddInt64(&r.totalSlow, 1)
		logWarn("Monitor '%s' slow update: %v", m.GetName(), elapsed)
	}
}

func (r *MonitorRegistry) Register(item MonitorItem) {
	r.mutex.Lock()
	r.items[item.GetName()] = item
	r.mutex.Unlock()
	r.stateMu.Lock()
	if _, ok := r.states[item.GetName()]; !ok {
		r.states[item.GetName()] = &monitorRunState{}
	}
	r.stateMu.Unlock()
}

func (r *MonitorRegistry) Get(name string) MonitorItem {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.items[name]
}

func (r *MonitorRegistry) GetAll() map[string]MonitorItem {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	result := make(map[string]MonitorItem)
	for k, v := range r.items {
		result[k] = v
	}
	return result
}

func (r *MonitorRegistry) scheduleUpdate(item MonitorItem) {
	if atomic.LoadInt32(&r.closed) == 1 || item == nil {
		return
	}
	name := item.GetName()
	r.stateMu.RLock()
	st := r.states[name]
	r.stateMu.RUnlock()
	if st == nil {
		r.stateMu.Lock()
		if r.states[name] == nil {
			r.states[name] = &monitorRunState{}
		}
		st = r.states[name]
		r.stateMu.Unlock()
	}
	now := time.Now()
	nowUnix := now.UnixNano()
	nextAllowed := atomic.LoadInt64(&st.nextAllowed)
	if nextAllowed > nowUnix {
		return
	}

	// 看门狗：如果运行超过10秒，强制清理
	if atomic.LoadInt32(&st.running) == 1 {
		last := atomic.LoadInt64(&st.lastStart)
		if last > 0 && time.Since(time.Unix(0, last)) > 10*time.Second {
			if atomic.CompareAndSwapInt32(&st.running, 1, 0) {
				logWarn("Monitor '%s' previous update stuck >10s, clearing state", name)
			}
		}
	}
	if !atomic.CompareAndSwapInt32(&st.running, 0, 1) {
		return
	}
	scale := atomic.LoadInt32(&r.intervalScale)
	interval := scaledMonitorInterval(name, scale)
	if interval <= 0 {
		interval = time.Second
	}
	atomic.StoreInt64(&st.lastStart, nowUnix)
	atomic.StoreInt64(&st.nextAllowed, now.Add(interval).UnixNano())
	atomic.AddInt64(&r.scheduledCount, 1)
	atomic.AddInt64(&r.totalScheduled, 1)
	select {
	case <-r.stopCh:
		atomic.StoreInt32(&st.running, 0)
		return
	case r.taskCh <- monitorUpdateTask{item: item, state: st}:
		return
	default:
		// Queue backpressure: drop this scheduling attempt and allow next cycle to retry.
		atomic.StoreInt32(&st.running, 0)
		atomic.AddInt64(&r.droppedCount, 1)
		atomic.AddInt64(&r.totalDropped, 1)
		lastDropped := atomic.LoadInt64(&st.lastDropped)
		if nowUnix-lastDropped > int64(5*time.Second) {
			if atomic.CompareAndSwapInt64(&st.lastDropped, lastDropped, nowUnix) {
				logWarn("Monitor update queue full, dropped '%s' once (workers=%d queue=%d)", name, r.workerCount, r.queueSize)
			}
		}
	}
}

func (r *MonitorRegistry) Update(names []string) error {
	if atomic.LoadInt32(&r.closed) == 1 {
		return nil
	}
	r.mutex.RLock()
	items := make([]MonitorItem, 0, len(names))
	for _, name := range names {
		if item, ok := r.items[name]; ok {
			items = append(items, item)
		}
	}
	r.mutex.RUnlock()
	for _, item := range items {
		r.scheduleUpdate(item)
	}
	return nil
}

func (r *MonitorRegistry) UpdateAll() error {
	if atomic.LoadInt32(&r.closed) == 1 {
		return nil
	}
	r.mutex.RLock()
	items := make([]MonitorItem, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	r.mutex.RUnlock()
	for _, item := range items {
		r.scheduleUpdate(item)
	}
	return nil
}

func (r *MonitorRegistry) Stats() MonitorRegistryStats {
	return MonitorRegistryStats{
		WorkerCount: r.workerCount,
		QueueSize:   r.queueSize,
		QueueLen:    len(r.taskCh),

		AutoTune:          r.autoTune,
		AutoTuneIntervalS: int(r.autoTuneInterval / time.Second),
		AutoTuneSlowRate:  r.autoTuneSlowRate,
		AutoTuneStableRun: int(r.autoTuneStableRun),
		AutoTuneMaxScale:  int(r.autoTuneMaxScale),
		IntervalScale:     int(atomic.LoadInt32(&r.intervalScale)),

		ScheduledTotal: atomic.LoadInt64(&r.totalScheduled),
		DroppedTotal:   atomic.LoadInt64(&r.totalDropped),
		SlowTotal:      atomic.LoadInt64(&r.totalSlow),
		CompletedTotal: atomic.LoadInt64(&r.totalCompleted),

		LastWindowScheduled: atomic.LoadInt64(&r.lastWindowScheduled),
		LastWindowDropped:   atomic.LoadInt64(&r.lastWindowDropped),
		LastWindowSlow:      atomic.LoadInt64(&r.lastWindowSlow),
		LastWindowCompleted: atomic.LoadInt64(&r.lastWindowCompleted),
	}
}

func monitorMinUpdateInterval(name string) time.Duration {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return time.Second
	}

	switch trimmed {
	case "cpu_model", "cpu_cores", "memory_total":
		return 30 * time.Second
	}

	if _, suffix, ok := parseIndexedMonitorName(trimmed, "disk"); ok {
		switch suffix {
		case "name", "size":
			return 30 * time.Second
		}
	}

	if _, suffix, ok := parseIndexedMonitorName(trimmed, "net"); ok {
		switch suffix {
		case "interface":
			return 15 * time.Second
		case "ip":
			return 2 * time.Second
		}
	}

	return time.Second
}

func scaledMonitorInterval(name string, scale int32) time.Duration {
	if scale <= 0 {
		scale = 1
	}
	base := monitorMinUpdateInterval(name)
	interval := time.Duration(scale) * base
	if interval > 60*time.Second {
		return 60 * time.Second
	}
	return interval
}

func parseIndexedMonitorName(name, prefix string) (int, string, bool) {
	if !strings.HasPrefix(name, prefix) {
		return 0, "", false
	}
	underscore := strings.IndexByte(name, '_')
	if underscore <= len(prefix) {
		return 0, "", false
	}
	indexText := name[len(prefix):underscore]
	index, err := strconv.Atoi(indexText)
	if err != nil || index <= 0 {
		return 0, "", false
	}
	suffix := strings.TrimSpace(name[underscore+1:])
	if suffix == "" {
		return 0, "", false
	}
	return index, suffix, true
}

var (
	globalMonitorRegistry *MonitorRegistry
	globalMonitorConfig   *MonitorConfig
	globalMonitorMu       sync.Mutex
	globalMonitorCond     = sync.NewCond(&globalMonitorMu)
	globalRegistryReady   bool
	globalRegistryLoading bool
)

func GetMonitorRegistry() *MonitorRegistry {
	globalMonitorMu.Lock()
	for globalRegistryLoading {
		globalMonitorCond.Wait()
	}
	if globalMonitorRegistry == nil {
		globalMonitorRegistry = newMonitorRegistryFromConfig(globalMonitorConfig)
		globalRegistryReady = false
	}
	registry := globalMonitorRegistry
	if globalRegistryReady {
		globalMonitorMu.Unlock()
		return registry
	}
	globalRegistryLoading = true
	globalMonitorMu.Unlock()

	initializeMonitorItems(registry, nil, "")
	performInitialUpdate(registry)

	globalMonitorMu.Lock()
	if globalMonitorRegistry == registry {
		globalRegistryReady = true
	}
	globalRegistryLoading = false
	globalMonitorCond.Broadcast()
	globalMonitorMu.Unlock()

	return registry
}

func GetMonitorRegistryWithConfig(requiredMonitors []string, networkInterface string) *MonitorRegistry {
	globalMonitorMu.Lock()
	for globalRegistryLoading {
		globalMonitorCond.Wait()
	}
	if globalMonitorRegistry == nil {
		globalMonitorRegistry = newMonitorRegistryFromConfig(globalMonitorConfig)
		globalRegistryReady = false
	}
	registry := globalMonitorRegistry
	if globalRegistryReady {
		globalMonitorMu.Unlock()
		return registry
	}
	globalRegistryLoading = true
	globalMonitorMu.Unlock()

	initializeMonitorItems(registry, requiredMonitors, networkInterface)
	performInitialUpdate(registry)

	globalMonitorMu.Lock()
	if globalMonitorRegistry == registry {
		globalRegistryReady = true
	}
	globalRegistryLoading = false
	globalMonitorCond.Broadcast()
	globalMonitorMu.Unlock()

	return registry
}

func newMonitorRegistryFromConfig(cfg *MonitorConfig) *MonitorRegistry {
	if cfg == nil {
		return NewMonitorRegistry()
	}
	workers := cfg.GetMonitorUpdateWorkers()
	queueSize := cfg.GetMonitorUpdateQueueSize(workers)
	autoTune := cfg.GetMonitorAutoTune()
	autoTuneInterval := cfg.GetMonitorAutoTuneInterval()
	autoTuneSlowRate := cfg.GetMonitorAutoTuneSlowRate()
	autoTuneStableRun := cfg.GetMonitorAutoTuneStableRuns()
	autoTuneMaxScale := cfg.GetMonitorAutoTuneMaxScale()
	registry := newMonitorRegistryWithOptions(MonitorRegistryOptions{
		WorkerCount:       workers,
		QueueSize:         queueSize,
		AutoTune:          autoTune,
		AutoTuneInterval:  autoTuneInterval,
		AutoTuneSlowRate:  autoTuneSlowRate,
		AutoTuneStableRun: autoTuneStableRun,
		AutoTuneMaxScale:  autoTuneMaxScale,
	})
	logInfo(
		"Monitor registry workers=%d queue=%d auto_tune=%t interval=%s slow_rate=%.2f stable_runs=%d max_scale=%d",
		workers, queueSize, autoTune, autoTuneInterval, autoTuneSlowRate, autoTuneStableRun, autoTuneMaxScale,
	)
	return registry
}

// Set/Get global config
func SetGlobalMonitorConfig(config *MonitorConfig) {
	globalMonitorMu.Lock()
	globalMonitorConfig = config
	globalMonitorMu.Unlock()
}

func GetGlobalMonitorConfig() *MonitorConfig {
	globalMonitorMu.Lock()
	defer globalMonitorMu.Unlock()
	return globalMonitorConfig
}

func ResetGlobalMonitorRegistry() {
	globalMonitorMu.Lock()
	for globalRegistryLoading {
		globalMonitorCond.Wait()
	}
	oldRegistry := globalMonitorRegistry
	globalMonitorRegistry = nil
	globalRegistryReady = false
	globalMonitorMu.Unlock()
	if oldRegistry != nil {
		oldRegistry.Close()
	}
}

func performInitialUpdate(registry *MonitorRegistry) {
	if registry == nil {
		return
	}
	// 改为异步调度，避免初始化阶段阻塞
	_ = registry.UpdateAll()
}

type MonitorItemConfig struct {
	Name     string
	Creator  func() MonitorItem
	Required bool
}

type MonitorRegistryConfig struct{ Monitors []MonitorItemConfig }

func getMonitorRegistryConfig() *MonitorRegistryConfig {
	monitors := []MonitorItemConfig{
		{"cpu_usage", func() MonitorItem { return NewCPUUsageMonitor() }, true},
		{"cpu_temp", func() MonitorItem { return NewCPUTempMonitor() }, true},
		{"cpu_freq", func() MonitorItem { return NewCPUFreqMonitor() }, true},
		{"cpu_model", func() MonitorItem { return NewCPUModelMonitor() }, true},
		{"cpu_cores", func() MonitorItem { return NewCPUCoresMonitor() }, true},
		{"memory_usage", func() MonitorItem { return NewMemoryUsageMonitor() }, true},
		{"memory_used", func() MonitorItem { return NewMemoryUsedMonitor() }, true},
		{"memory_total", func() MonitorItem { return NewMemoryTotalMonitor() }, true},
		{"memory_usage_text", func() MonitorItem { return NewMemoryUsageTextMonitor() }, true},
		{"memory_usage_progress", func() MonitorItem { return NewMemoryUsageProgressMonitor() }, true},
		{"swap_usage", func() MonitorItem { return NewSwapUsageMonitor() }, true},
		{"load_avg", func() MonitorItem { return NewLoadAvgMonitor() }, true},
		{"current_time", func() MonitorItem { return NewCurrentTimeMonitor() }, true},
	}
	return &MonitorRegistryConfig{Monitors: monitors}
}

func initializeMonitorItems(registry *MonitorRegistry, requiredMonitors []string, networkInterface string) {
	if registry == nil {
		return
	}
	config := getMonitorRegistryConfig()
	for _, monitorConfig := range config.Monitors {
		registry.Register(monitorConfig.Creator())
	}
	diskRequiredMax := maxIndexedMonitor(requiredMonitors, "disk", []string{"name", "size", "temp"})
	diskDetected := detectNamedDiskCount()
	diskSlots := max(diskRequiredMax, diskDetected)
	if diskSlots > 16 {
		diskSlots = 16
	}
	for diskIndex := 1; diskIndex <= diskSlots; diskIndex++ {
		registry.Register(NewDiskNameMonitor(diskIndex))
		registry.Register(NewDiskSizeMonitor(diskIndex))
		registry.Register(NewDiskTempMonitorByIndex(diskIndex))
	}
	networkRequiredMax := maxIndexedMonitor(requiredMonitors, "net", []string{"upload", "download", "ip", "interface"})
	networkDetected := len(getActiveNetworkInterfaces())
	networkSlots := max(networkRequiredMax, networkDetected)
	if networkSlots > 16 {
		networkSlots = 16
	}
	for networkIndex := 1; networkIndex <= networkSlots; networkIndex++ {
		if upload := NewNetworkInterfaceMonitorByIndex(networkIndex, "upload"); upload != nil {
			registry.Register(upload)
		}
		if download := NewNetworkInterfaceMonitorByIndex(networkIndex, "download"); download != nil {
			registry.Register(download)
		}
		if ip := NewNetworkInterfaceMonitorByIndex(networkIndex, "ip"); ip != nil {
			registry.Register(ip)
		}
		if iface := NewNetworkInterfaceMonitorByIndex(networkIndex, "name"); iface != nil {
			registry.Register(iface)
		}
	}
	initializeExternalMonitorItems(registry)
	initializeCustomMonitors(registry)
}

func detectNamedDiskCount() int {
	count := 0
	for _, disk := range detectDiskInfo() {
		if disk == nil {
			continue
		}
		if strings.TrimSpace(disk.Name) == "" {
			continue
		}
		count++
	}
	return count
}

func maxIndexedMonitor(requiredMonitors []string, prefix string, allowedSuffixes []string) int {
	if len(requiredMonitors) == 0 {
		return 0
	}
	suffixSet := make(map[string]struct{}, len(allowedSuffixes))
	for _, suffix := range allowedSuffixes {
		suffixSet[suffix] = struct{}{}
	}

	maxIndex := 0
	for _, name := range requiredMonitors {
		trimmed := strings.TrimSpace(name)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		underscore := strings.IndexByte(trimmed, '_')
		if underscore <= len(prefix) {
			continue
		}
		indexText := trimmed[len(prefix):underscore]
		index, err := strconv.Atoi(indexText)
		if err != nil || index <= 0 {
			continue
		}
		suffix := trimmed[underscore+1:]
		if _, ok := suffixSet[suffix]; !ok {
			continue
		}
		if index > maxIndex {
			maxIndex = index
		}
	}
	return maxIndex
}
