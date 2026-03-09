package main

import (
	"ax206monitor/rtsssource"
	"ax206monitor/webui"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	runtime   *WebAPI
	profiles  *ProfileManager
	wsMu      sync.RWMutex
	wsClients map[*webSocketClient]struct{}
}

type WebMetaResponse struct {
	ConfigPath           string   `json:"config_path"`
	Platform             string   `json:"platform,omitempty"`
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

var (
	runningWebServerMu sync.Mutex
	runningWebServer   *echo.Echo
)

func claimRunningWebServer(server *echo.Echo) error {
	if server == nil {
		return fmt.Errorf("web server instance is nil")
	}
	runningWebServerMu.Lock()
	defer runningWebServerMu.Unlock()
	if runningWebServer != nil {
		return fmt.Errorf("web server already running")
	}
	runningWebServer = server
	return nil
}

func releaseRunningWebServer(server *echo.Echo) {
	runningWebServerMu.Lock()
	defer runningWebServerMu.Unlock()
	if runningWebServer == server {
		runningWebServer = nil
	}
}

func stopRunningWebServer(timeout time.Duration) error {
	runningWebServerMu.Lock()
	server := runningWebServer
	runningWebServerMu.Unlock()
	if server == nil {
		return nil
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

type WebServerProcess struct {
	mu       sync.Mutex
	port     int
	devMode  bool
	viteURL  string
	done     chan error
	stopping bool
}

func NewWebServerProcess(port int, devMode bool, viteURL string) *WebServerProcess {
	return &WebServerProcess{
		port:    port,
		devMode: devMode,
		viteURL: viteURL,
	}
}

func (p *WebServerProcess) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", p.port)
}

func (p *WebServerProcess) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done != nil
}

func (p *WebServerProcess) Start() error {
	p.mu.Lock()
	if p.done != nil {
		p.mu.Unlock()
		return nil
	}
	done := make(chan error, 1)
	p.done = done
	p.stopping = false
	p.mu.Unlock()

	options := WebServerOptions{
		Addr:    fmt.Sprintf("127.0.0.1:%d", p.port),
		DevMode: p.devMode,
		ViteURL: p.viteURL,
	}

	go func() {
		err := RunWebServer(options)
		done <- err
		close(done)
	}()
	select {
	case err := <-done:
		p.mu.Lock()
		if p.done == done {
			p.done = nil
			p.stopping = false
		}
		p.mu.Unlock()
		if err == nil {
			return nil
		}
		return err
	case <-time.After(150 * time.Millisecond):
		go p.watchProcess(done)
		logInfoModule("tray", "web server started in-process pid=%d addr=%s", os.Getpid(), p.URL())
		return nil
	}
}

func (p *WebServerProcess) watchProcess(done chan error) {
	err, ok := <-done
	if !ok {
		err = nil
	}

	p.mu.Lock()
	stopping := p.stopping
	if p.done == done {
		p.done = nil
		p.stopping = false
	}
	p.mu.Unlock()

	if err != nil {
		if stopping {
			logInfoModule("tray", "web server stopped")
			return
		}
		logWarnModule("tray", "web server exited with error: %v", err)
		return
	}
	if stopping {
		logInfoModule("tray", "web server stopped")
		return
	}
	logInfoModule("tray", "web server exited")
}

func (p *WebServerProcess) Stop() error {
	p.mu.Lock()
	done := p.done
	if done != nil {
		p.stopping = true
	}
	p.mu.Unlock()

	if done == nil {
		return nil
	}

	if err := stopRunningWebServer(3 * time.Second); err != nil {
		return fmt.Errorf("stop web server failed: %w", err)
	}

	select {
	case err, ok := <-done:
		if !ok || err == nil {
			return nil
		}
		return err
	case <-time.After(4 * time.Second):
		return fmt.Errorf("wait web server stop timeout")
	}
}

func RunWebServer(options WebServerOptions) error {
	configPath, err := getUserConfigPath()
	if err != nil {
		return err
	}

	initialConfig, err := loadUserConfigOrDefault(configPath)
	if err != nil {
		return err
	}

	profileManager, initialConfig, err := InitializeGlobalProfileManager(configPath, initialConfig)
	if err != nil {
		return err
	}

	store := &ConfigStore{
		path:      configPath,
		cfg:       initialConfig,
		profiles:  profileManager,
		wsClients: make(map[*webSocketClient]struct{}),
	}
	runtime, err := AcquireSharedWebAPI(initialConfig)
	if err != nil {
		return err
	}
	defer ReleaseSharedWebAPI(runtime)
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
		options, err := listConfiguredCoolerControlOptions(cfg)
		if err != nil {
			url := cfg.GetCoolerControlURL()
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
		items, err := listConfiguredLibreHardwareMonitorOptions(cfg)
		if err != nil {
			url := cfg.GetLibreHardwareMonitorURL()
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

	if err := claimRunningWebServer(e); err != nil {
		return err
	}
	defer releaseRunningWebServer(e)
	err = e.Start(options.Addr)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func buildWebMetaResponse(store *ConfigStore) WebMetaResponse {
	config := store.getConfig()
	return WebMetaResponse{
		ConfigPath:           store.path,
		Platform:             goruntime.GOOS,
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

func collectMonitorNames(config *MonitorConfig, runtimeState *WebAPI) []string {
	monitorSet := make(map[string]struct{})
	registryConfig := getCollectorManagerConfig()
	for _, monitor := range registryConfig.Items {
		monitorSet[monitor.Name] = struct{}{}
	}
	for _, aliasName := range monitorAliasNames() {
		monitorSet[aliasName] = struct{}{}
	}
	if goruntime.GOOS == "windows" && config != nil && config.IsRTSSCollectEnabled() {
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
		Name:                    "web",
		Width:                   480,
		Height:                  320,
		DefaultFont:             getDefaultFontName(),
		DefaultFontSize:         16,
		DefaultValueFontSize:    18,
		DefaultLabelFontSize:    16,
		DefaultUnitFontSize:     14,
		DefaultColor:            "#f8fafc",
		DefaultBackground:       "#0b1220",
		LevelColors:             []string{"#22c55e", "#eab308", "#f97316", "#ef4444"},
		DefaultThresholds:       []float64{25, 50, 75, 100},
		FontFamilies:            getDefaultFontFamilies(),
		OutputTypes:             []string{outputTypeMemImg},
		RefreshInterval:         1000,
		CollectWarnMS:           100,
		RenderWaitMaxMS:         300,
		HistorySize:             150,
		DefaultHistoryPoints:    150,
		NetworkInterface:        "",
		LibreHardwareMonitorURL: defaultLibreHardwareMonitorURL,
		CoolerControlURL:        defaultCoolerControlURL,
		CustomMonitors:          []CustomMonitorConfig{},
		CollectorConfig: map[string]CollectorConfig{
			collectorGoNativeCPU:          {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			collectorGoNativeMemory:       {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			collectorGoNativeSystem:       {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			collectorGoNativeDisk:         {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			collectorGoNativeNetwork:      {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			collectorCustomAll:            {Enabled: boolPtr(true), Options: map[string]interface{}{}},
			collectorCoolerControl:        {Enabled: boolPtr(false), Options: map[string]interface{}{"url": defaultCoolerControlURL}},
			collectorLibreHardwareMonitor: {Enabled: boolPtr(false), Options: map[string]interface{}{"url": defaultLibreHardwareMonitorURL}},
			collectorRTSS:                 {Enabled: boolPtr(false), Options: map[string]interface{}{}},
		},
		Items: []ItemConfig{
			{
				Type:    itemTypeSimpleValue,
				Monitor: "go_native.cpu.temp",
				Unit:    "auto",
				X:       12,
				Y:       12,
				Width:   180,
				Height:  56,
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
	migrateLegacyCollectorConfig(cfg)
	builtinCooler, builtinLibre, builtinRTSS := builtinCollectorDefaults(cfg.Name)
	ensureCollectorConfigDefault(cfg, collectorGoNativeCPU, true)
	ensureCollectorConfigDefault(cfg, collectorGoNativeMemory, true)
	ensureCollectorConfigDefault(cfg, collectorGoNativeSystem, true)
	ensureCollectorConfigDefault(cfg, collectorGoNativeDisk, true)
	ensureCollectorConfigDefault(cfg, collectorGoNativeNetwork, true)
	ensureCollectorConfigDefault(cfg, collectorCustomAll, true)
	ensureCollectorConfigDefault(cfg, collectorCoolerControl, builtinCooler)
	ensureCollectorConfigDefault(cfg, collectorLibreHardwareMonitor, builtinLibre)
	defaultRTSS := builtinRTSS
	if goruntime.GOOS == "windows" {
		defaultRTSS = true
	}
	ensureCollectorConfigDefault(cfg, collectorRTSS, defaultRTSS)
	applyBuiltinCollectorDefaults(cfg, builtinCooler, builtinLibre, builtinRTSS)
	if goruntime.GOOS == "windows" && configNeedsRTSS(cfg) {
		setCollectorEnabled(cfg, collectorRTSS, true)
	}
	if cfg.EnableRTSSCollect {
		entry := cfg.CollectorConfig[collectorRTSS]
		enabled := true
		entry.Enabled = &enabled
		if entry.Options == nil {
			entry.Options = map[string]interface{}{}
		}
		cfg.CollectorConfig[collectorRTSS] = entry
	}
	if cfg.IsCollectorEnabled(collectorRTSS, false) {
		cfg.EnableRTSSCollect = true
	}
	enforceCollectorPlatformSupport(cfg)

	if cfg.FontFamilies == nil {
		cfg.FontFamilies = getDefaultFontFamilies()
	}
	if goruntime.GOOS == "windows" {
		if strings.Contains(strings.ToLower(strings.TrimSpace(cfg.DefaultFont)), "wqy") {
			cfg.DefaultFont = getDefaultFontName()
		}
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
	if cfg.RefreshInterval < 100 {
		cfg.RefreshInterval = 100
	}
	if cfg.RefreshInterval > 10000 {
		cfg.RefreshInterval = 10000
	}
	if cfg.CollectWarnMS <= 0 {
		cfg.CollectWarnMS = 100
	}
	if cfg.CollectWarnMS < 10 {
		cfg.CollectWarnMS = 10
	}
	if cfg.CollectWarnMS > 10000 {
		cfg.CollectWarnMS = 10000
	}
	if cfg.RenderWaitMaxMS <= 0 {
		cfg.RenderWaitMaxMS = 300
	}
	if cfg.RenderWaitMaxMS < 0 {
		cfg.RenderWaitMaxMS = 0
	}
	if cfg.RenderWaitMaxMS > cfg.RefreshInterval {
		cfg.RenderWaitMaxMS = cfg.RefreshInterval
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
		cfg.DefaultFont = getDefaultFontName()
	}
	sanitizeFontConfig(cfg)
	if cfg.DefaultFontSize <= 0 {
		cfg.DefaultFontSize = 16
	}
	if cfg.DefaultValueFontSize <= 0 {
		cfg.DefaultValueFontSize = cfg.GetDefaultFontSize() + 2
	}
	if cfg.DefaultValueFontSize < 8 {
		cfg.DefaultValueFontSize = 8
	}
	if cfg.DefaultLabelFontSize <= 0 {
		cfg.DefaultLabelFontSize = cfg.GetDefaultFontSize()
	}
	if cfg.DefaultLabelFontSize < 8 {
		cfg.DefaultLabelFontSize = 8
	}
	if cfg.DefaultUnitFontSize <= 0 {
		cfg.DefaultUnitFontSize = cfg.GetDefaultLabelFontSize() - 2
	}
	if cfg.DefaultUnitFontSize < 8 {
		cfg.DefaultUnitFontSize = 8
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
	if strings.TrimSpace(cfg.CoolerControlURL) == "" {
		cfg.CoolerControlURL = defaultCoolerControlURL
	}
	if strings.TrimSpace(cfg.LibreHardwareMonitorURL) == "" {
		cfg.LibreHardwareMonitorURL = defaultLibreHardwareMonitorURL
	}
	if cfg.DefaultHistoryPoints <= 0 {
		if cfg.HistorySize > 0 {
			cfg.DefaultHistoryPoints = cfg.HistorySize
		} else {
			cfg.DefaultHistoryPoints = 150
		}
	}
	if cfg.DefaultHistoryPoints < 10 {
		cfg.DefaultHistoryPoints = 10
	}
	if cfg.DefaultHistoryPoints > 5000 {
		cfg.DefaultHistoryPoints = 5000
	}
	if cfg.HistorySize <= 0 {
		cfg.HistorySize = cfg.DefaultHistoryPoints
	}
	ensureTypeDefaults(cfg)
	setCollectorOptionDefault(cfg, collectorCoolerControl, "url", defaultCoolerControlURL)
	setCollectorOptionDefault(cfg, collectorLibreHardwareMonitor, "url", defaultLibreHardwareMonitorURL)

	for idx := range cfg.Items {
		item := &cfg.Items[idx]
		item.Type = normalizeItemType(item.Type)
		item.Monitor = normalizeMonitorAlias(item.Monitor)
		item.EditUIName = defaultEditUIName(item.EditUIName, idx, item)
		if !cfg.AllowCustomStyle {
			item.CustomStyle = false
		}
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
		if isBuiltinProfileName(cfg.Name) {
			stripBuiltinItemStyleOverrides(item)
		}
		if item.Type == itemTypeSimpleChart {
			item.History = true
			if item.PointSize < 0 {
				item.PointSize = 0
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

func ensureTypeDefaults(cfg *MonitorConfig) {
	if cfg == nil {
		return
	}
	if cfg.TypeDefaults == nil {
		cfg.TypeDefaults = map[string]ItemTypeDefaults{}
	}
	for _, itemType := range webItemTypes() {
		entry := cfg.TypeDefaults[itemType]
		entry = normalizeTypeDefaultsEntry(cfg, itemType, entry)
		cfg.TypeDefaults[itemType] = entry
	}
}

func normalizeTypeDefaultsEntry(cfg *MonitorConfig, itemType string, entry ItemTypeDefaults) ItemTypeDefaults {
	base := defaultTypeDefaults(cfg, itemType)

	if entry.FontSize < 0 {
		entry.FontSize = 0
	}
	if entry.SmallFontSize < 0 {
		entry.SmallFontSize = 0
	}
	if entry.MediumFontSize < 0 {
		entry.MediumFontSize = 0
	}
	if entry.LargeFontSize < 0 {
		entry.LargeFontSize = 0
	}
	if entry.UnitFontSize < 0 {
		entry.UnitFontSize = 0
	}
	if entry.SmallFontSize == 0 && entry.UnitFontSize > 0 {
		entry.SmallFontSize = entry.UnitFontSize
	}
	if entry.MediumFontSize == 0 && entry.FontSize > 0 {
		entry.MediumFontSize = entry.FontSize
	}
	if entry.LargeFontSize == 0 && entry.FontSize > 0 {
		entry.LargeFontSize = entry.FontSize
	}
	if entry.PointSize < 0 {
		entry.PointSize = 0
	}
	if isHistoryItemType(itemType) {
		if entry.PointSize == 0 {
			entry.PointSize = base.PointSize
		}
		if entry.PointSize < 10 {
			entry.PointSize = 10
		}
	} else {
		entry.PointSize = 0
	}
	if itemType == itemTypeSimpleChart {
		if entry.RenderAttrsMap == nil {
			entry.RenderAttrsMap = map[string]interface{}{}
		}
		historyPoints := entry.PointSize
		if historyPoints <= 0 {
			if raw, exists := entry.RenderAttrsMap["history_points"]; exists && raw != nil {
				switch typed := raw.(type) {
				case int:
					historyPoints = typed
				case int32:
					historyPoints = int(typed)
				case int64:
					historyPoints = int(typed)
				case float32:
					historyPoints = int(typed)
				case float64:
					historyPoints = int(typed)
				}
			}
		}
		if historyPoints <= 0 {
			historyPoints = base.PointSize
		}
		if historyPoints < 10 {
			historyPoints = 10
		}
		entry.PointSize = historyPoints
		entry.RenderAttrsMap["history_points"] = historyPoints
	}
	if entry.BorderWidth < 0 {
		entry.BorderWidth = 0
	}
	if entry.Radius < 0 {
		entry.Radius = 0
	}
	if strings.TrimSpace(entry.Color) == "" {
		entry.Color = base.Color
	}
	if strings.TrimSpace(entry.Background) == "" {
		entry.Background = base.Background
	}
	if strings.TrimSpace(entry.UnitColor) == "" {
		entry.UnitColor = base.UnitColor
	}
	if strings.TrimSpace(entry.BorderColor) == "" {
		entry.BorderColor = base.BorderColor
	}
	entry.FontSize = 0
	entry.UnitFontSize = 0

	entry.RenderAttrsMap = mergeDefaultRenderAttrs(base.RenderAttrsMap, entry.RenderAttrsMap)
	return entry
}

func defaultTypeDefaults(cfg *MonitorConfig, itemType string) ItemTypeDefaults {
	defaultColor := "#f8fafc"
	defaultValueSize := 16
	defaultLabelSize := 14
	defaultHistoryPoints := 150
	if cfg != nil {
		defaultColor = cfg.GetDefaultTextColor()
		defaultValueSize = cfg.GetDefaultValueFontSize()
		defaultLabelSize = cfg.GetDefaultLabelFontSize()
		defaultHistoryPoints = cfg.GetDefaultHistoryPoints()
	}

	defaults := ItemTypeDefaults{
		FontSize:       0,
		SmallFontSize:  0,
		MediumFontSize: 0,
		LargeFontSize:  0,
		Color:          defaultColor,
		Background:     "",
		UnitColor:      defaultColor,
		UnitFontSize:   0,
		PointSize:      0,
		BorderColor:    "#475569",
		BorderWidth:    0,
		Radius:         0,
		RenderAttrsMap: map[string]interface{}{},
	}

	if isHistoryItemType(itemType) {
		defaults.PointSize = defaultHistoryPoints
	}

	switch itemType {
	case itemTypeSimpleChart:
		defaults.RenderAttrsMap = map[string]interface{}{
			"history_points": defaultHistoryPoints,
		}
	case itemTypeSimpleRect, itemTypeSimpleCircle:
		defaults.Background = "#33415566"
	case itemTypeLabelText1:
		defaults.RenderAttrsMap = map[string]interface{}{
			"content_padding": 3,
		}
	case itemTypeLabelText2:
		defaults.RenderAttrsMap = map[string]interface{}{
			"content_padding": 5,
		}
	case itemTypeFullChart:
		defaults.Background = "#111827c8"
		defaults.RenderAttrsMap = map[string]interface{}{
			"content_padding":         1,
			"body_gap":                4,
			"title_font_size":         defaultLabelSize,
			"value_font_size":         defaultValueSize,
			"header_divider":          true,
			"header_divider_offset":   3,
			"header_divider_color":    "#94a3b840",
			"history_points":          defaultHistoryPoints,
			"show_segment_lines":      true,
			"show_grid_lines":         true,
			"grid_lines":              4,
			"fill_area":               true,
			"line_width":              2.0,
			"show_avg_line":           true,
			"chart_color":             "#38bdf8",
			"chart_area_bg":           "",
			"chart_area_border_color": "",
		}
	case itemTypeFullProgress:
		defaults.Background = "#111827c8"
		defaults.RenderAttrsMap = map[string]interface{}{
			"content_padding":       1,
			"body_gap":              0,
			"title_font_size":       defaultLabelSize,
			"value_font_size":       defaultValueSize,
			"header_divider":        true,
			"header_divider_offset": 3,
			"header_divider_color":  "#94a3b840",
			"progress_style":        "gradient",
			"bar_height":            0.0,
			"track_color":           "#1f2937",
			"segments":              12,
			"segment_gap":           2.0,
		}
	}

	return defaults
}

func mergeDefaultRenderAttrs(base map[string]interface{}, overrides map[string]interface{}) map[string]interface{} {
	if len(base) == 0 && len(overrides) == 0 {
		return map[string]interface{}{}
	}
	merged := make(map[string]interface{}, len(base)+len(overrides))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}
	return merged
}

func stripBuiltinItemStyleOverrides(item *ItemConfig) {
	if item == nil {
		return
	}
	item.FontSize = 0
	item.Color = ""
	item.Background = ""
	item.UnitColor = ""
	item.UnitFontSize = 0
	item.BorderColor = ""
	item.BorderWidth = 0
	item.Radius = 0
	item.PointSize = 0
	if len(item.RenderAttrsMap) == 0 {
		return
	}
	styleKeys := map[string]struct{}{
		"content_padding":         {},
		"body_gap":                {},
		"value_font_size":         {},
		"label_font_size":         {},
		"meta_font_size":          {},
		"title_font_size":         {},
		"header_divider":          {},
		"header_divider_offset":   {},
		"header_divider_color":    {},
		"history_points":          {},
		"show_segment_lines":      {},
		"show_grid_lines":         {},
		"grid_lines":              {},
		"fill_area":               {},
		"line_width":              {},
		"show_avg_line":           {},
		"chart_color":             {},
		"chart_area_bg":           {},
		"chart_area_border_color": {},
		"progress_style":          {},
		"bar_height":              {},
		"track_color":             {},
		"segments":                {},
		"segment_gap":             {},
	}
	filtered := make(map[string]interface{}, len(item.RenderAttrsMap))
	for key, value := range item.RenderAttrsMap {
		if _, remove := styleKeys[key]; remove {
			continue
		}
		filtered[key] = value
	}
	item.RenderAttrsMap = filtered
}

func migrateLegacyCollectorConfig(cfg *MonitorConfig) {
	if cfg == nil || cfg.CollectorConfig == nil {
		return
	}
	migrateCollectorConfigKey(cfg, legacyCollectorCoolerControl, collectorCoolerControl)
	migrateCollectorConfigKey(cfg, legacyCollectorLibreHardwareMonitor, collectorLibreHardwareMonitor)
	migrateCollectorConfigKey(cfg, legacyCollectorRTSS, collectorRTSS)
}

func migrateCollectorConfigKey(cfg *MonitorConfig, oldKey, newKey string) {
	if cfg == nil || cfg.CollectorConfig == nil {
		return
	}
	oldEntry, oldExists := cfg.CollectorConfig[oldKey]
	newEntry, newExists := cfg.CollectorConfig[newKey]
	if !oldExists {
		return
	}
	if !newExists {
		cfg.CollectorConfig[newKey] = oldEntry
	} else {
		if newEntry.Enabled == nil {
			newEntry.Enabled = oldEntry.Enabled
		}
		if newEntry.Options == nil {
			newEntry.Options = map[string]interface{}{}
		}
		for key, value := range oldEntry.Options {
			if _, exists := newEntry.Options[key]; !exists {
				newEntry.Options[key] = value
			}
		}
		cfg.CollectorConfig[newKey] = newEntry
	}
	delete(cfg.CollectorConfig, oldKey)
}

func enforceCollectorPlatformSupport(cfg *MonitorConfig) {
	if cfg == nil || cfg.CollectorConfig == nil {
		return
	}
	for name, entry := range cfg.CollectorConfig {
		if isCollectorSupportedOnCurrentPlatform(name) {
			continue
		}
		disabled := false
		entry.Enabled = &disabled
		if entry.Options == nil {
			entry.Options = map[string]interface{}{}
		}
		cfg.CollectorConfig[name] = entry
	}
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func builtinCollectorDefaults(profileName string) (cooler bool, libre bool, rtss bool) {
	if !isBuiltinProfileName(profileName) {
		return false, false, false
	}
	switch goruntime.GOOS {
	case "windows":
		return false, true, true
	case "linux":
		return true, false, false
	default:
		return false, false, false
	}
}

func applyBuiltinCollectorDefaults(cfg *MonitorConfig, cooler, libre, rtss bool) {
	if cfg == nil || !isBuiltinProfileName(cfg.Name) {
		return
	}
	setCollectorEnabled(cfg, collectorCoolerControl, cooler)
	setCollectorEnabled(cfg, collectorLibreHardwareMonitor, libre)
	setCollectorEnabled(cfg, collectorRTSS, rtss)
}

func setCollectorEnabled(cfg *MonitorConfig, name string, enabled bool) {
	if cfg == nil {
		return
	}
	entry := cfg.CollectorConfig[name]
	entry.Enabled = boolPtr(enabled)
	if entry.Options == nil {
		entry.Options = map[string]interface{}{}
	}
	cfg.CollectorConfig[name] = entry
}

func isBuiltinProfileName(name string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	switch trimmed {
	case "builtin1", "builtin2", "builtin3", "builtin4":
		return true
	default:
		return false
	}
}

func configNeedsRTSS(cfg *MonitorConfig) bool {
	if cfg == nil {
		return false
	}
	for _, item := range cfg.Items {
		name := strings.ToLower(strings.TrimSpace(item.Monitor))
		if name == "" {
			continue
		}
		if name == "alias.gpu.fps" || strings.HasPrefix(name, "rtss_") {
			return true
		}
	}
	for _, custom := range cfg.CustomMonitors {
		name := strings.ToLower(strings.TrimSpace(custom.Name))
		if name == "alias.gpu.fps" || strings.HasPrefix(name, "rtss_") {
			return true
		}
		for _, source := range custom.Sources {
			src := strings.ToLower(strings.TrimSpace(source))
			if src == "alias.gpu.fps" || strings.HasPrefix(src, "rtss_") {
				return true
			}
		}
	}
	return false
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

func setCollectorOptionDefault(cfg *MonitorConfig, collectorName, optionKey, value string) {
	if cfg == nil || strings.TrimSpace(value) == "" {
		return
	}
	entry, exists := cfg.CollectorConfig[collectorName]
	if !exists {
		enabled := false
		entry = CollectorConfig{
			Enabled: &enabled,
			Options: map[string]interface{}{},
		}
	}
	if entry.Options == nil {
		entry.Options = map[string]interface{}{}
	}
	current := strings.TrimSpace(fmt.Sprintf("%v", entry.Options[optionKey]))
	if current == "" {
		entry.Options[optionKey] = value
	}
	cfg.CollectorConfig[collectorName] = entry
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
	return normalizeMonitorAliasInput(name)
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
			UpdatedAt: time.Now().Format(time.RFC3339Nano),
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
