package main

import (
	"ax206monitor/rtsssource"
	"ax206monitor/webui"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type WebServerOptions struct {
	Addr    string
	DevMode bool
	ViteURL string
}

type ConfigStore struct {
	path      string
	mu        sync.RWMutex
	cfg       *MonitorConfig
	runtime   *WebRuntime
	profiles  *ProfileManager
	wsMu      sync.RWMutex
	wsClients map[*webSocketClient]struct{}
}

type WebMetaResponse struct {
	ConfigPath           string   `json:"config_path"`
	Monitors             []string `json:"monitors"`
	Collectors           []string `json:"collectors,omitempty"`
	ItemTypes            []string `json:"item_types"`
	OutputTypes          []string `json:"output_types"`
	FontFamilies         []string `json:"font_families"`
	NetworkInterfaces    []string `json:"network_interfaces"`
	CustomMonitorTypes   []string `json:"custom_monitor_types"`
	CustomAggregateTypes []string `json:"custom_aggregate_types"`
	ActiveProfile        string   `json:"active_profile,omitempty"`
}

type WebConfigResponse struct {
	Config *MonitorConfig `json:"config"`
}

type ProfileInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
	Size      int64  `json:"size"`
	ReadOnly  bool   `json:"readonly,omitempty"`
	Builtin   bool   `json:"builtin,omitempty"`
}

type ProfileManager struct {
	currentConfigPath string
	profilesDir       string
	activeProfileFile string
	builtinProfiles   map[string]*MonitorConfig
	mu                sync.RWMutex
}

var profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

func RunWebServer(options WebServerOptions) error {
	configPath, err := getUserConfigPath()
	if err != nil {
		return err
	}

	initialConfig, err := loadUserConfigOrDefault(configPath)
	if err != nil {
		return err
	}

	profileManager, err := NewProfileManager(configPath)
	if err != nil {
		return err
	}
	initialConfig, err = profileManager.Initialize(initialConfig)
	if err != nil {
		return err
	}

	store := &ConfigStore{
		path:      configPath,
		cfg:       initialConfig,
		profiles:  profileManager,
		wsClients: make(map[*webSocketClient]struct{}),
	}
	runtime, err := NewWebRuntime(initialConfig)
	if err != nil {
		return err
	}
	runtime.SetIdleConfigProvider(func() (*MonitorConfig, error) {
		activeName := profileManager.ActiveName()
		if strings.TrimSpace(activeName) == "" {
			return store.getConfig(), nil
		}
		cfg, err := profileManager.LoadProfile(activeName)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	})
	store.runtime = runtime
	defer runtime.Close()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/api/meta", func(c echo.Context) error {
		store.touchRuntime()
		return c.JSON(http.StatusOK, buildWebMetaResponse(store))
	})

	e.GET("/api/coolercontrol/options", func(c echo.Context) error {
		store.touchRuntime()
		cfg := store.getRuntimeConfig()
		if cfg == nil {
			err := "runtime config unavailable"
			logWarnModule("web", "coolercontrol options request failed: %s", err)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"items": []CoolerControlMonitorOption{},
				"error": err,
			})
		}
		url := cfg.GetCoolerControlURL()
		if url == "" {
			return c.JSON(http.StatusOK, map[string]interface{}{"items": []CoolerControlMonitorOption{}})
		}
		client := GetCoolerControlClient(
			url,
			cfg.GetCoolerControlUsername(),
			cfg.GetCoolerControlPassword(),
		)
		options, err := client.ListMonitorOptions()
		if err != nil {
			logWarnModule("web", "coolercontrol options request failed (url=%s): %v", url, err)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"items": []CoolerControlMonitorOption{},
				"error": err.Error(),
			})
		}
		if options == nil {
			options = []CoolerControlMonitorOption{}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"items": options})
	})

	e.GET("/api/librehardwaremonitor/options", func(c echo.Context) error {
		store.touchRuntime()
		cfg := store.getRuntimeConfig()
		if cfg == nil {
			err := "runtime config unavailable"
			logWarnModule("web", "librehardwaremonitor options request failed: %s", err)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"items": []LibreHardwareMonitorMonitorOption{},
				"error": err,
			})
		}
		url := cfg.GetLibreHardwareMonitorURL()
		if url == "" {
			return c.JSON(http.StatusOK, map[string]interface{}{"items": []LibreHardwareMonitorMonitorOption{}})
		}
		client := GetLibreHardwareMonitorClient(url)
		items, err := client.ListMonitorOptions()
		if err != nil {
			logWarnModule("web", "librehardwaremonitor options request failed (url=%s): %v", url, err)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"items": []LibreHardwareMonitorMonitorOption{},
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"items": items})
	})

	e.GET("/api/config", func(c echo.Context) error {
		store.touchRuntime()
		return c.JSON(http.StatusOK, WebConfigResponse{Config: store.getConfig()})
	})

	e.POST("/api/preview/config", func(c echo.Context) error {
		var payload WebConfigResponse
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid payload: %v", err)})
		}
		if payload.Config == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing config"})
		}

		normalizeMonitorConfig(payload.Config)
		if err := store.applyPreviewConfigToRuntime(payload.Config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"ok": true})
	})

	e.PUT("/api/config", func(c echo.Context) error {
		var payload WebConfigResponse
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid payload: %v", err)})
		}
		if payload.Config == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing config"})
		}

		normalizeMonitorConfig(payload.Config)
		if active := store.profiles.ActiveName(); active != "" && store.profiles.isBuiltinProfileUnsafe(active) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("active profile '%s' is built-in read-only, copy it first", active)})
		}
		if err := saveUserConfig(store.path, payload.Config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if active := store.profiles.ActiveName(); active != "" {
			if err := store.profiles.SaveProfile(active, payload.Config); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
		}
		store.setConfig(payload.Config)
		if err := store.applyConfigToRuntime(payload.Config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"ok":   true,
			"meta": buildWebMetaResponse(store),
		})
	})

	e.GET("/api/profiles", func(c echo.Context) error {
		store.touchRuntime()
		items, err := store.profiles.List()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"active": store.profiles.ActiveName(),
			"items":  items,
		})
	})

	e.POST("/api/profiles", func(c echo.Context) error {
		var payload struct {
			Name   string         `json:"name"`
			Config *MonitorConfig `json:"config"`
			Switch bool           `json:"switch"`
		}
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid payload: %v", err)})
		}
		payload.Name = strings.TrimSpace(payload.Name)
		if payload.Name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing profile name"})
		}
		cfg := payload.Config
		if cfg == nil {
			cfg = store.getConfig()
		}
		normalizeMonitorConfig(cfg)
		if err := store.profiles.SaveProfile(payload.Name, cfg); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if payload.Switch {
			if _, err := store.profiles.Switch(payload.Name); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
			store.setConfig(cfg)
			if err := store.applyConfigToRuntime(cfg); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
		}
		items, _ := store.profiles.List()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"ok":     true,
			"active": store.profiles.ActiveName(),
			"items":  items,
		})
	})

	e.POST("/api/profiles/rename", func(c echo.Context) error {
		var payload struct {
			OldName string `json:"old_name"`
			NewName string `json:"new_name"`
		}
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid payload: %v", err)})
		}
		payload.OldName = strings.TrimSpace(payload.OldName)
		payload.NewName = strings.TrimSpace(payload.NewName)
		if payload.OldName == "" || payload.NewName == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing old_name or new_name"})
		}

		activeCfg, err := store.profiles.RenameProfile(payload.OldName, payload.NewName)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		if activeCfg != nil {
			store.setConfig(activeCfg)
			if err := store.applyConfigToRuntime(activeCfg); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
		}

		items, _ := store.profiles.List()
		resp := map[string]interface{}{
			"ok":     true,
			"active": store.profiles.ActiveName(),
			"items":  items,
		}
		if activeCfg != nil {
			resp["config"] = activeCfg
		}
		return c.JSON(http.StatusOK, resp)
	})

	e.GET("/api/profiles/:name", func(c echo.Context) error {
		store.touchRuntime()
		name := c.Param("name")
		cfg, err := store.profiles.LoadProfile(name)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"config": cfg})
	})

	e.PUT("/api/profiles/:name", func(c echo.Context) error {
		name := c.Param("name")
		var payload struct {
			Config *MonitorConfig `json:"config"`
		}
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid payload: %v", err)})
		}
		if payload.Config == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing config"})
		}
		normalizeMonitorConfig(payload.Config)
		if err := store.profiles.SaveProfile(name, payload.Config); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if store.profiles.ActiveName() == name {
			if err := saveUserConfig(store.path, payload.Config); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			store.setConfig(payload.Config)
			if err := store.applyConfigToRuntime(payload.Config); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
		}
		items, _ := store.profiles.List()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"ok":     true,
			"active": store.profiles.ActiveName(),
			"items":  items,
		})
	})

	e.POST("/api/profiles/switch", func(c echo.Context) error {
		var payload struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid payload: %v", err)})
		}
		cfg, err := store.profiles.Switch(payload.Name)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		store.setConfig(cfg)
		if err := store.applyConfigToRuntime(cfg); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		items, _ := store.profiles.List()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"ok":     true,
			"active": store.profiles.ActiveName(),
			"items":  items,
			"config": cfg,
		})
	})

	e.DELETE("/api/profiles/:name", func(c echo.Context) error {
		name := c.Param("name")
		if err := store.profiles.DeleteProfile(name); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		currentCfg, err := loadUserConfigOrDefault(store.path)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		store.setConfig(currentCfg)
		if err := store.applyConfigToRuntime(currentCfg); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		items, _ := store.profiles.List()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"ok":     true,
			"active": store.profiles.ActiveName(),
			"items":  items,
		})
	})

	e.GET("/api/snapshot", func(c echo.Context) error {
		return c.JSON(http.StatusOK, store.snapshot())
	})

	e.GET("/api/ws", func(c echo.Context) error {
		return serveWebSocket(c, store)
	})

	e.GET("/api/runtime/monitor", func(c echo.Context) error {
		store.touchRuntime()
		stats := store.monitorStats()
		if stats == nil {
			return c.JSON(http.StatusOK, map[string]interface{}{"available": false})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"available": true,
			"stats":     stats,
		})
	})

	e.GET("/api/collectors", func(c echo.Context) error {
		store.touchRuntime()
		names := store.collectorNames()
		states := store.collectorStates()
		items := make([]map[string]interface{}, 0, len(names))
		for _, name := range names {
			items = append(items, map[string]interface{}{
				"name":    name,
				"enabled": states[name],
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"items": items,
		})
	})

	e.POST("/api/collectors/:name/enable", func(c echo.Context) error {
		name := strings.TrimSpace(c.Param("name"))
		if name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing collector name"})
		}
		if !store.setCollectorEnabled(name, true) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "collector not found"})
		}
		cfg := store.getConfig()
		if cfg.CollectorConfig == nil {
			cfg.CollectorConfig = map[string]CollectorConfig{}
		}
		entry := cfg.CollectorConfig[name]
		entry.Enabled = boolPtr(true)
		if entry.Options == nil {
			entry.Options = map[string]interface{}{}
		}
		cfg.CollectorConfig[name] = entry
		store.setConfig(cfg)
		_ = saveUserConfig(store.path, cfg)
		return c.JSON(http.StatusOK, map[string]interface{}{"ok": true})
	})

	e.POST("/api/collectors/:name/disable", func(c echo.Context) error {
		name := strings.TrimSpace(c.Param("name"))
		if name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing collector name"})
		}
		if !store.setCollectorEnabled(name, false) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "collector not found"})
		}
		cfg := store.getConfig()
		if cfg.CollectorConfig == nil {
			cfg.CollectorConfig = map[string]CollectorConfig{}
		}
		entry := cfg.CollectorConfig[name]
		entry.Enabled = boolPtr(false)
		if entry.Options == nil {
			entry.Options = map[string]interface{}{}
		}
		cfg.CollectorConfig[name] = entry
		store.setConfig(cfg)
		_ = saveUserConfig(store.path, cfg)
		return c.JSON(http.StatusOK, map[string]interface{}{"ok": true})
	})

	e.GET("/api/preview", func(c echo.Context) error {
		store.touchRuntime()
		if pngData, ok := GetMemImgPNG(); ok {
			return c.Blob(http.StatusOK, "image/png", pngData)
		}
		return c.NoContent(http.StatusNoContent)
	})

	if options.DevMode {
		viteURL := options.ViteURL
		if viteURL == "" {
			viteURL = "http://127.0.0.1:18087"
		}
		if err := webui.RegisterDevProxy(e, viteURL); err != nil {
			return err
		}
		logInfoModule("web", "Web config server started in dev mode: http://%s", options.Addr)
		logInfoModule("web", "Proxy target: %s", viteURL)
	} else {
		staticFS, err := getEmbeddedWebAssetsFS()
		if err != nil {
			return fmt.Errorf("failed to load embedded frontend: %w", err)
		}
		webui.RegisterEmbeddedFrontend(e, staticFS)
		logInfoModule("web", "Web config server started: http://%s", options.Addr)
	}

	return e.Start(options.Addr)
}

func buildWebMetaResponse(store *ConfigStore) WebMetaResponse {
	config := store.getConfig()
	return WebMetaResponse{
		ConfigPath:           store.path,
		Monitors:             collectMonitorNames(config, store.runtime),
		Collectors:           store.collectorNames(),
		ItemTypes:            webItemTypes(),
		OutputTypes:          []string{outputTypeMemImg, outputTypeAX206USB},
		FontFamilies:         collectFontFamilies(config),
		NetworkInterfaces:    listNetworkInterfaces(),
		CustomMonitorTypes:   []string{"file", "mixed", "coolercontrol", "librehardwaremonitor"},
		CustomAggregateTypes: []string{"max", "min", "avg"},
		ActiveProfile:        store.profiles.ActiveName(),
	}
}

func collectMonitorNames(config *MonitorConfig, runtimeState *WebRuntime) []string {
	monitorSet := make(map[string]struct{})
	registryConfig := getCollectorManagerConfig()
	for _, monitor := range registryConfig.Items {
		monitorSet[monitor.Name] = struct{}{}
	}
	if goruntime.GOOS == "windows" {
		for _, option := range rtsssource.GetRTSSClient().ListMonitorOptions() {
			if strings.TrimSpace(option.Name) == "" {
				continue
			}
			monitorSet[option.Name] = struct{}{}
		}
	}
	if runtimeState != nil {
		for _, name := range runtimeState.AllMonitorNames() {
			monitorSet[name] = struct{}{}
		}
	}

	if config != nil {
		for _, item := range config.Items {
			if strings.TrimSpace(item.Monitor) != "" {
				monitorSet[item.Monitor] = struct{}{}
			}
		}
		for _, custom := range config.CustomMonitors {
			if strings.TrimSpace(custom.Name) != "" {
				monitorSet[custom.Name] = struct{}{}
			}
		}
	}

	monitors := make([]string, 0, len(monitorSet))
	for monitor := range monitorSet {
		monitors = append(monitors, monitor)
	}
	sort.Strings(monitors)
	return monitors
}

func listNetworkInterfaces() []string {
	options := []string{}
	interfaces, err := net.Interfaces()
	if err != nil {
		return options
	}
	for _, iface := range interfaces {
		if strings.TrimSpace(iface.Name) == "" {
			continue
		}
		options = append(options, iface.Name)
	}
	sort.Strings(options)
	return options
}

func collectFontFamilies(config *MonitorConfig) []string {
	familySet := make(map[string]struct{})
	for _, name := range getDefaultFontFamilies() {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		familySet[trimmed] = struct{}{}
	}
	if config != nil {
		for _, name := range config.FontFamilies {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				continue
			}
			familySet[trimmed] = struct{}{}
		}
	}
	items := make([]string, 0, len(familySet))
	for name := range familySet {
		items = append(items, name)
	}
	sort.Strings(items)
	return items
}

func getUserConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "ax206monitor", "config.json"), nil
}

func loadUserConfigOrDefault(path string) (*MonitorConfig, error) {
	if data, err := os.ReadFile(path); err == nil {
		var cfg MonitorConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("invalid user config %s: %w", path, err)
		}
		normalizeMonitorConfig(&cfg)
		return &cfg, nil
	}

	cfg := &MonitorConfig{
		Name:              "web",
		Width:             480,
		Height:            320,
		DefaultFont:       "DejaVu Sans Mono",
		DefaultFontSize:   16,
		DefaultColor:      "#f8fafc",
		DefaultBackground: "#0b1220",
		LevelColors:       []string{"#22c55e", "#eab308", "#f97316", "#ef4444"},
		DefaultThresholds: []float64{25, 50, 75, 100},
		FontFamilies:      getDefaultFontFamilies(),
		OutputTypes:       []string{outputTypeMemImg},
		RefreshInterval:   1000,
		HistorySize:       150,
		NetworkInterface:  "",
		CustomMonitors:    []CustomMonitorConfig{},
		CollectorConfig: map[string]CollectorConfig{
			"go_native.cpu":                 {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			"go_native.memory":              {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			"go_native.system":              {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			"go_native.disk":                {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			"go_native.network":             {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			"custom.all":                    {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			"external.coolercontrol":        {Enabled: boolPtr(false), Options: map[string]interface{}{}},
			"external.librehardwaremonitor": {Enabled: boolPtr(false), Options: map[string]interface{}{}},
			"external.rtss":                 {Enabled: boolPtr(false), Options: map[string]interface{}{}},
		},
		Items: []ItemConfig{
			{
				Type:     itemTypeSimpleValue,
				Monitor:  "go_native.cpu.temp",
				Unit:     "auto",
				X:        12,
				Y:        12,
				Width:    180,
				Height:   56,
				FontSize: 16,
			},
		},
	}
	normalizeMonitorConfig(cfg)
	return cfg, nil
}

func saveUserConfig(path string, cfg *MonitorConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func normalizeMonitorConfig(cfg *MonitorConfig) {
	if cfg.CollectorConfig == nil {
		cfg.CollectorConfig = make(map[string]CollectorConfig)
	}
	ensureCollectorConfigDefault(cfg, "go_native.cpu", true)
	ensureCollectorConfigDefault(cfg, "go_native.memory", true)
	ensureCollectorConfigDefault(cfg, "go_native.system", true)
	ensureCollectorConfigDefault(cfg, "go_native.disk", true)
	ensureCollectorConfigDefault(cfg, "go_native.network", true)
	ensureCollectorConfigDefault(cfg, "custom.all", true)
	ensureCollectorConfigDefault(cfg, "external.coolercontrol", false)
	ensureCollectorConfigDefault(cfg, "external.librehardwaremonitor", false)
	ensureCollectorConfigDefault(cfg, "external.rtss", false)

	if cfg.FontFamilies == nil {
		cfg.FontFamilies = getDefaultFontFamilies()
	}
	if cfg.CustomMonitors == nil {
		cfg.CustomMonitors = []CustomMonitorConfig{}
	}
	if cfg.Items == nil {
		cfg.Items = []ItemConfig{}
	}
	if cfg.Width <= 0 {
		cfg.Width = 480
	}
	if cfg.Height <= 0 {
		cfg.Height = 320
	}
	minEdge := cfg.Width
	if cfg.Height < minEdge {
		minEdge = cfg.Height
	}
	maxLayoutPadding := (minEdge - 10) / 2
	if maxLayoutPadding < 0 {
		maxLayoutPadding = 0
	}
	if cfg.LayoutPadding < 0 {
		cfg.LayoutPadding = 0
	}
	if cfg.LayoutPadding > maxLayoutPadding {
		cfg.LayoutPadding = maxLayoutPadding
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 1000
	}
	if cfg.MonitorUpdateWorkers < 0 {
		cfg.MonitorUpdateWorkers = 0
	}
	if cfg.MonitorUpdateQueueSize < 0 {
		cfg.MonitorUpdateQueueSize = 0
	}
	if cfg.MonitorAutoTuneInterval < 0 {
		cfg.MonitorAutoTuneInterval = 0
	}
	if cfg.MonitorAutoTuneSlowRate < 0 {
		cfg.MonitorAutoTuneSlowRate = 0
	}
	if cfg.MonitorAutoTuneStable < 0 {
		cfg.MonitorAutoTuneStable = 0
	}
	if cfg.MonitorAutoTuneMaxScale < 0 {
		cfg.MonitorAutoTuneMaxScale = 0
	}
	if strings.TrimSpace(cfg.DefaultFont) == "" {
		cfg.DefaultFont = "DejaVu Sans Mono"
	}
	if cfg.DefaultFontSize <= 0 {
		cfg.DefaultFontSize = 16
	}
	if strings.TrimSpace(cfg.DefaultColor) == "" {
		cfg.DefaultColor = "#f8fafc"
	}
	if strings.TrimSpace(cfg.DefaultBackground) == "" {
		cfg.DefaultBackground = "#0b1220"
	}
	cfg.LevelColors = normalizeLevelColors(cfg.LevelColors, []string{"#22c55e", "#eab308", "#f97316", "#ef4444"})
	cfg.DefaultThresholds = normalizeThresholds(cfg.DefaultThresholds, 0, 100)
	if len(cfg.DefaultThresholds) != 4 {
		cfg.DefaultThresholds = []float64{25, 50, 75, 100}
	}
	cfg.OutputTypes = normalizeOutputTypes(cfg.OutputTypes)
	cfg.NetworkInterface = strings.TrimSpace(cfg.NetworkInterface)
	if strings.EqualFold(cfg.NetworkInterface, "auto") {
		cfg.NetworkInterface = ""
	}

	for idx := range cfg.Items {
		item := &cfg.Items[idx]
		item.Type = normalizeItemType(item.Type)
		item.Monitor = normalizeMonitorAlias(item.Monitor)
		item.EditUIName = defaultEditUIName(item.EditUIName, idx, item)
		if item.Width <= 0 {
			item.Width = 120
		}
		if item.Height <= 0 {
			item.Height = 40
		}
		if isCollectorItemType(item.Type) {
			if strings.TrimSpace(item.Unit) == "" {
				item.Unit = "auto"
			}
		} else {
			item.Unit = ""
			item.UnitColor = ""
			item.UnitFontSize = 0
		}
		if item.UnitFontSize < 0 {
			item.UnitFontSize = 0
		}
		if item.Type == itemTypeSimpleChart {
			item.History = true
			if item.PointSize <= 0 {
				item.PointSize = cfg.HistorySize
			}
			if item.PointSize < 10 {
				item.PointSize = 10
			}
		} else {
			item.History = false
			item.PointSize = 0
		}
		if !isRangeItemType(item.Type) {
			item.Max = 0
			item.MaxValue = nil
			item.MinValue = nil
		}
		if strings.TrimSpace(item.Background) == "" {
			item.Background = ""
		}
		item.LevelColors = normalizeLevelColors(item.LevelColors, nil)
		item.Thresholds = normalizeThresholds(item.Thresholds, 0, 100)
	}

	for idx := range cfg.CustomMonitors {
		custom := &cfg.CustomMonitors[idx]
		custom.Source = normalizeMonitorAlias(custom.Source)
		if len(custom.Sources) == 0 {
			continue
		}
		normalized := make([]string, 0, len(custom.Sources))
		for _, source := range custom.Sources {
			name := normalizeMonitorAlias(source)
			if name == "" {
				continue
			}
			normalized = append(normalized, name)
		}
		custom.Sources = normalized
	}
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func ensureCollectorConfigDefault(cfg *MonitorConfig, name string, defaultEnabled bool) {
	if cfg == nil {
		return
	}
	current, exists := cfg.CollectorConfig[name]
	if !exists {
		enabled := defaultEnabled
		cfg.CollectorConfig[name] = CollectorConfig{
			Enabled: &enabled,
			Options: map[string]interface{}{},
		}
		return
	}
	if current.Enabled == nil {
		enabled := defaultEnabled
		current.Enabled = &enabled
	}
	if current.Options == nil {
		current.Options = map[string]interface{}{}
	}
	cfg.CollectorConfig[name] = current
}

func defaultEditUIName(current string, idx int, item *ItemConfig) string {
	if strings.TrimSpace(current) != "" {
		return strings.TrimSpace(current)
	}
	name := strings.TrimSpace(item.Monitor)
	if name == "" {
		name = strings.TrimSpace(item.Type)
	}
	if name == "" {
		name = "item"
	}
	return fmt.Sprintf("%d_%s", idx+1, name)
}

func normalizeItemType(itemType string) string {
	return normalizeItemTypeName(itemType)
}

func normalizeMonitorAlias(name string) string {
	trimmed := strings.TrimSpace(name)
	switch trimmed {
	case "gpu_fps":
		return "rtss_fps"
	default:
		return trimmed
	}
}

func normalizeLevelColors(colors []string, fallback []string) []string {
	result := make([]string, 0, 4)
	for _, color := range colors {
		trimmed := strings.TrimSpace(color)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
		if len(result) == 4 {
			break
		}
	}
	if len(result) == 0 && len(fallback) > 0 {
		for _, color := range fallback {
			trimmed := strings.TrimSpace(color)
			if trimmed != "" {
				result = append(result, trimmed)
			}
			if len(result) == 4 {
				break
			}
		}
	}
	return result
}

func cloneMonitorConfig(cfg *MonitorConfig) *MonitorConfig {
	data, _ := json.Marshal(cfg)
	var copyCfg MonitorConfig
	_ = json.Unmarshal(data, &copyCfg)
	return &copyCfg
}

func (s *ConfigStore) getConfig() *MonitorConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneMonitorConfig(s.cfg)
}

func (s *ConfigStore) setConfig(cfg *MonitorConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cloneMonitorConfig(cfg)
}

func (s *ConfigStore) touchRuntime() {
	if s.runtime != nil {
		s.runtime.Touch()
	}
}

func (s *ConfigStore) applyConfigToRuntime(cfg *MonitorConfig) error {
	if s.runtime == nil {
		return nil
	}
	return s.runtime.ApplyConfig(cfg)
}

func (s *ConfigStore) applyPreviewConfigToRuntime(cfg *MonitorConfig) error {
	if s.runtime == nil {
		return nil
	}
	return s.runtime.ApplyPreviewConfig(cfg)
}

func (s *ConfigStore) getRuntimeConfig() *MonitorConfig {
	if s.runtime == nil {
		return nil
	}
	return s.runtime.CurrentConfig()
}

func (s *ConfigStore) snapshot() WebSnapshotResponse {
	if s.runtime == nil {
		return WebSnapshotResponse{
			Mode:      "required",
			UpdatedAt: time.Now().Format(time.RFC3339),
			Monitors:  []string{},
			Values:    map[string]WebMonitorSnapshotItem{},
		}
	}
	return s.runtime.Snapshot()
}

func (s *ConfigStore) monitorStats() *CollectorManagerStats {
	if s.runtime == nil {
		return nil
	}
	return s.runtime.MonitorStats()
}

func (s *ConfigStore) collectorStates() map[string]bool {
	if s.runtime == nil {
		return map[string]bool{}
	}
	return s.runtime.CollectorStates()
}

func (s *ConfigStore) collectorNames() []string {
	if s.runtime == nil {
		return []string{}
	}
	return s.runtime.CollectorNames()
}

func (s *ConfigStore) setCollectorEnabled(name string, enabled bool) bool {
	if s.runtime == nil {
		return false
	}
	return s.runtime.SetCollectorEnabled(name, enabled)
}

func NewProfileManager(currentConfigPath string) (*ProfileManager, error) {
	configDir := filepath.Dir(currentConfigPath)
	builtins, err := loadBuiltinProfiles()
	if err != nil {
		return nil, err
	}
	pm := &ProfileManager{
		currentConfigPath: currentConfigPath,
		profilesDir:       filepath.Join(configDir, "profiles"),
		activeProfileFile: filepath.Join(configDir, "profiles", "active-profile"),
		builtinProfiles:   builtins,
	}
	if err := os.MkdirAll(pm.profilesDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create profiles directory: %w", err)
	}
	return pm, nil
}

func (pm *ProfileManager) Initialize(baseConfig *MonitorConfig) (*MonitorConfig, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.customProfileExistsUnsafe("default") {
		seed := cloneMonitorConfig(baseConfig)
		if builtin, ok := pm.builtinProfiles["normal"]; ok {
			seed = cloneMonitorConfig(builtin)
		}
		if err := pm.saveProfileUnsafe("default", seed); err != nil {
			return nil, err
		}
	}

	items, err := pm.listUnsafe()
	if err != nil || len(items) == 0 {
		return nil, fmt.Errorf("no available profile")
	}

	active := pm.activeUnsafe()
	if active == "" || !pm.profileExistsUnsafe(active) {
		if pm.profileExistsUnsafe("default") {
			active = "default"
		} else {
			active = items[0].Name
		}
		if err := pm.setActiveUnsafe(active); err != nil {
			return nil, err
		}
	}

	cfg, err := pm.loadProfileUnsafe(active)
	if err != nil {
		if err := pm.saveProfileUnsafe(active, baseConfig); err != nil {
			return nil, err
		}
		cfg = cloneMonitorConfig(baseConfig)
		normalizeMonitorConfig(cfg)
	}

	if err := saveUserConfig(pm.currentConfigPath, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (pm *ProfileManager) ActiveName() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.activeUnsafe()
}

func (pm *ProfileManager) List() ([]ProfileInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.listUnsafe()
}

func (pm *ProfileManager) SaveProfile(name string, cfg *MonitorConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.isBuiltinProfileUnsafe(name) {
		return fmt.Errorf("built-in profile is read-only: %s", name)
	}
	return pm.saveProfileUnsafe(name, cfg)
}

func (pm *ProfileManager) LoadProfile(name string) (*MonitorConfig, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.loadProfileUnsafe(name)
}

func (pm *ProfileManager) RenameProfile(oldName, newName string) (*MonitorConfig, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if err := validateProfileName(oldName); err != nil {
		return nil, err
	}
	if err := validateProfileName(newName); err != nil {
		return nil, err
	}
	if oldName == newName {
		return nil, nil
	}
	if pm.isBuiltinProfileUnsafe(oldName) {
		return nil, fmt.Errorf("built-in profile is read-only: %s", oldName)
	}
	if !pm.customProfileExistsUnsafe(oldName) {
		return nil, fmt.Errorf("profile not found: %s", oldName)
	}
	if pm.profileExistsUnsafe(newName) {
		return nil, fmt.Errorf("target profile already exists: %s", newName)
	}

	if err := os.Rename(pm.profilePath(oldName), pm.profilePath(newName)); err != nil {
		return nil, fmt.Errorf("failed to rename profile: %w", err)
	}

	if pm.activeUnsafe() != oldName {
		return nil, nil
	}
	if err := pm.setActiveUnsafe(newName); err != nil {
		return nil, err
	}
	cfg, err := pm.loadProfileUnsafe(newName)
	if err != nil {
		return nil, err
	}
	if err := saveUserConfig(pm.currentConfigPath, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (pm *ProfileManager) Switch(name string) (*MonitorConfig, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cfg, err := pm.loadProfileUnsafe(name)
	if err != nil {
		return nil, err
	}
	if err := pm.setActiveUnsafe(name); err != nil {
		return nil, err
	}
	if err := saveUserConfig(pm.currentConfigPath, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (pm *ProfileManager) DeleteProfile(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if err := validateProfileName(name); err != nil {
		return err
	}
	if pm.isBuiltinProfileUnsafe(name) {
		return fmt.Errorf("built-in profile is read-only: %s", name)
	}
	targetPath := pm.profilePath(name)
	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("profile not found: %s", name)
	}

	active := pm.activeUnsafe()
	if active == name {
		items, err := pm.listUnsafe()
		if err != nil {
			return err
		}
		var next string
		for _, item := range items {
			if item.Name != name {
				next = item.Name
				break
			}
		}
		if next == "" {
			return fmt.Errorf("no fallback profile available")
		}
		cfg, err := pm.loadProfileUnsafe(next)
		if err != nil {
			return err
		}
		if err := pm.setActiveUnsafe(next); err != nil {
			return err
		}
		if err := saveUserConfig(pm.currentConfigPath, cfg); err != nil {
			return err
		}
	}

	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	return nil
}

func (pm *ProfileManager) profilePath(name string) string {
	return filepath.Join(pm.profilesDir, name+".json")
}

func (pm *ProfileManager) activeUnsafe() string {
	data, err := os.ReadFile(pm.activeProfileFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (pm *ProfileManager) setActiveUnsafe(name string) error {
	if err := validateProfileName(name); err != nil {
		return err
	}
	return os.WriteFile(pm.activeProfileFile, []byte(name+"\n"), 0o644)
}

func (pm *ProfileManager) profileExistsUnsafe(name string) bool {
	if err := validateProfileName(name); err != nil {
		return false
	}
	if pm.isBuiltinProfileUnsafe(name) {
		return true
	}
	_, err := os.Stat(pm.profilePath(name))
	return err == nil
}

func (pm *ProfileManager) customProfileExistsUnsafe(name string) bool {
	if err := validateProfileName(name); err != nil {
		return false
	}
	_, err := os.Stat(pm.profilePath(name))
	return err == nil
}

func (pm *ProfileManager) isBuiltinProfileUnsafe(name string) bool {
	if pm.builtinProfiles == nil {
		return false
	}
	_, ok := pm.builtinProfiles[strings.TrimSpace(name)]
	return ok
}

func (pm *ProfileManager) saveProfileUnsafe(name string, cfg *MonitorConfig) error {
	if err := validateProfileName(name); err != nil {
		return err
	}
	if pm.isBuiltinProfileUnsafe(name) {
		return fmt.Errorf("built-in profile is read-only: %s", name)
	}
	normalizeMonitorConfig(cfg)
	cfg.Name = name
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode profile: %w", err)
	}
	if err := os.WriteFile(pm.profilePath(name), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}
	return nil
}

func (pm *ProfileManager) loadProfileUnsafe(name string) (*MonitorConfig, error) {
	if err := validateProfileName(name); err != nil {
		return nil, err
	}
	if builtin, ok := pm.builtinProfiles[name]; ok {
		cfg := cloneMonitorConfig(builtin)
		normalizeMonitorConfig(cfg)
		cfg.Name = name
		return cfg, nil
	}
	data, err := os.ReadFile(pm.profilePath(name))
	if err != nil {
		return nil, fmt.Errorf("failed to read profile '%s': %w", name, err)
	}

	var cfg MonitorConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid profile '%s': %w", name, err)
	}
	normalizeMonitorConfig(&cfg)
	cfg.Name = name
	return &cfg, nil
}

func (pm *ProfileManager) listUnsafe() ([]ProfileInfo, error) {
	entries, err := os.ReadDir(pm.profilesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles directory: %w", err)
	}

	items := make([]ProfileInfo, 0, len(entries)+len(pm.builtinProfiles))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		if err := validateProfileName(name); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, ProfileInfo{
			Name:      name,
			Path:      pm.profilePath(name),
			UpdatedAt: info.ModTime().Format(time.RFC3339),
			Size:      info.Size(),
		})
	}

	for _, name := range sortedBuiltinProfileNames(pm.builtinProfiles) {
		if pm.customProfileExistsUnsafe(name) {
			continue
		}
		items = append(items, ProfileInfo{
			Name:      name,
			Path:      "[builtin]",
			UpdatedAt: "",
			Size:      0,
			ReadOnly:  true,
			Builtin:   true,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func validateProfileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name: %s", name)
	}
	return nil
}
