//go:build linux && cgo

package main

import (
	_ "embed"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

//go:embed assets/tray.png
var trayIconPNG []byte

type linuxTray struct {
	web     *WebServerProcess
	updater *AppUpdater
	readyCh chan struct{}
	stopCh  chan struct{}
	mu      sync.Mutex
	closed  bool
	openWeb *systray.MenuItem
	openUI  *systray.MenuItem
	update  *systray.MenuItem
	autoRun *systray.MenuItem
	exit    *systray.MenuItem
}

func StartTray(webController *WebServerProcess) (TrayHandle, error) {
	if webController == nil {
		return nil, fmt.Errorf("web controller is nil")
	}

	tray := &linuxTray{
		web:     webController,
		updater: NewAppUpdater(RepositoryURL, Version),
		readyCh: make(chan struct{}),
		stopCh:  make(chan struct{}),
	}
	go systray.Run(tray.onReady, tray.onExit)

	select {
	case <-tray.readyCh:
		return tray, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("tray init timeout")
	}
}

func (t *linuxTray) onReady() {
	systray.SetIcon(trayIconPNG)
	systray.SetTitle("MetricsRenderSender")
	systray.SetTooltip("MetricsRenderSender")

	t.openWeb = systray.AddMenuItem("Open Web Server", "Start web configuration server")
	t.openUI = systray.AddMenuItem("Open Web Editor", "Open web editor in browser")
	systray.AddSeparator()
	t.update = systray.AddMenuItem("Check for Updates", "Check latest release on GitHub")
	systray.AddSeparator()
	t.autoRun = systray.AddMenuItem("Enable Auto Start", "Enable auto start for current user")
	systray.AddSeparator()
	t.exit = systray.AddMenuItem("Exit", "Exit application")
	t.openUI.Disable()
	t.updater.Start(t.stopCh, t.syncMenuState)
	t.syncMenuState()

	close(t.readyCh)

	go t.handleMenuEvents()
	go t.watchWebState()
}

func (t *linuxTray) onExit() {
	select {
	case <-t.readyCh:
	default:
		close(t.readyCh)
	}
}

func (t *linuxTray) handleMenuEvents() {
	for {
		select {
		case <-t.stopCh:
			return
		case <-t.openWeb.ClickedCh:
			if t.web.IsRunning() {
				if err := t.web.Stop(); err != nil {
					logWarnModule("tray", "close web server failed: %v", err)
				}
			} else {
				if err := t.web.Start(); err != nil {
					logWarnModule("tray", "open web server failed: %v", err)
				}
			}
			t.syncMenuState()
		case <-t.openUI.ClickedCh:
			if !t.web.IsRunning() {
				t.syncMenuState()
				continue
			}
			if err := openBrowserURL(t.web.URL()); err != nil {
				logWarnModule("tray", "open browser failed: %v", err)
			}
		case <-t.update.ClickedCh:
			state := t.updater.State()
			if state.UpdateAvailable {
				go t.performUpgrade()
				continue
			}
			if !t.updater.TriggerCheck() {
				t.syncMenuState()
			}
		case <-t.autoRun.ClickedCh:
			enabled, err := IsAutoStartEnabled()
			if err != nil {
				logWarnModule("tray", "query auto start failed: %v", err)
				t.syncMenuState()
				continue
			}
			if enabled {
				if err := DisableAutoStart(); err != nil {
					logWarnModule("tray", "disable auto start failed: %v", err)
				}
			} else {
				if err := EnableAutoStart(); err != nil {
					logWarnModule("tray", "enable auto start failed: %v", err)
				}
			}
			t.syncMenuState()
		case <-t.exit.ClickedCh:
			t.Close()
			os.Exit(0)
		}
	}
}

func (t *linuxTray) watchWebState() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.syncMenuState()
		}
	}
}

func (t *linuxTray) syncMenuState() {
	running := t.web.IsRunning()
	if running {
		t.openWeb.SetTitle("Close Web Server")
		t.openUI.Enable()
	} else {
		t.openWeb.SetTitle("Open Web Server")
		t.openUI.Disable()
	}

	updateState := t.updater.State()
	switch {
	case !updateState.Supported:
		t.update.SetTitle("Updates Unavailable")
		t.update.Disable()
	case updateState.Installing:
		t.update.SetTitle("Updating...")
		t.update.Disable()
	case updateState.Checking:
		t.update.SetTitle("Checking Updates...")
		t.update.Disable()
	case updateState.UpdateAvailable:
		t.update.SetTitle(fmt.Sprintf("Upgrade to v%s", updateState.LatestVersion))
		t.update.Enable()
	default:
		t.update.SetTitle("Check for Updates")
		t.update.Enable()
	}

	autoEnabled, err := IsAutoStartEnabled()
	if err != nil {
		t.autoRun.SetTitle("Auto Start Unavailable")
		t.autoRun.Disable()
		return
	}
	t.autoRun.Enable()
	if autoEnabled {
		t.autoRun.SetTitle("Disable Auto Start")
		return
	}
	t.autoRun.SetTitle("Enable Auto Start")
}

func (t *linuxTray) performUpgrade() {
	if err := t.updater.PrepareUpgrade(os.Args[1:]); err != nil {
		logWarnModule("update", "prepare upgrade failed: %v", err)
		t.syncMenuState()
		return
	}
	logInfoModule("update", "upgrade prepared, restarting application")
	t.Close()
	os.Exit(0)
}

func (t *linuxTray) Close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	close(t.stopCh)
	t.mu.Unlock()

	if err := t.web.Stop(); err != nil {
		logWarnModule("tray", "stop web server during tray close failed: %v", err)
	}
	systray.Quit()
}
