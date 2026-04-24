//go:build windows && cgo

package main

import (
	_ "embed"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

//go:embed assets/tray.ico
var trayIconICO []byte

type windowsTray struct {
	web         *WebServerProcess
	updater     *AppUpdater
	readyCh     chan struct{}
	stopCh      chan struct{}
	mu          sync.Mutex
	closed      bool
	openWeb     *systray.MenuItem
	openUI      *systray.MenuItem
	bindHost    *trayWebBindMenuState
	profiles    *trayProfileMenuState
	viewLog     *systray.MenuItem
	checkUpdate *systray.MenuItem
	upgrade     *systray.MenuItem
	autoRun     *systray.MenuItem
	exit        *systray.MenuItem
}

func StartTray(webController *WebServerProcess) (TrayHandle, error) {
	if webController == nil {
		return nil, fmt.Errorf("web controller is nil")
	}

	tray := &windowsTray{
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

func (t *windowsTray) onReady() {
	systray.SetIcon(trayIconICO)
	systray.SetTitle("MetricsRenderSender")
	systray.SetTooltip("MetricsRenderSender")

	t.openWeb = systray.AddMenuItem("Open Web Server", "Start web configuration server")
	t.openUI = systray.AddMenuItem("Open Web Editor", "Open web editor in browser")
	t.bindHost = newTrayWebBindMenuState(t.web)
	t.profiles = newTrayProfileMenuState()
	t.viewLog = systray.AddMenuItem("Open Log Directory", "Open application log directory")
	systray.AddSeparator()
	t.checkUpdate = systray.AddMenuItem("Check for Updates", "Check latest release on GitHub")
	t.upgrade = systray.AddMenuItem("Upgrade Unavailable", "Install latest release")
	t.upgrade.Disable()
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

func (t *windowsTray) onExit() {
	select {
	case <-t.readyCh:
	default:
		close(t.readyCh)
	}
}

func (t *windowsTray) handleMenuEvents() {
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
		case <-t.viewLog.ClickedCh:
			if err := openFileSystemPath(resolveLogDirectoryPath()); err != nil {
				logWarnModule("tray", "open log directory failed: %v", err)
			}
		case <-t.checkUpdate.ClickedCh:
			if !t.updater.TriggerCheck() {
				t.syncMenuState()
			}
		case <-t.upgrade.ClickedCh:
			state := t.updater.State()
			if state.UpdateAvailable {
				go t.performUpgrade(false)
				continue
			}
			if state.LatestVersion != "" {
				go t.performUpgrade(true)
				continue
			}
			t.syncMenuState()
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

func (t *windowsTray) watchWebState() {
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

func (t *windowsTray) syncMenuState() {
	if t.bindHost != nil {
		t.bindHost.refresh()
	}
	if t.profiles != nil {
		t.profiles.refresh(false)
	}
	running := t.web.IsRunning()
	if running {
		t.openWeb.SetTitle("Close Web Server")
		t.openUI.Enable()
	} else {
		t.openWeb.SetTitle("Open Web Server")
		t.openUI.Disable()
	}

	updateMenu := resolveTrayUpdateMenuConfig(t.updater.State())
	t.checkUpdate.SetTitle(updateMenu.checkTitle)
	if updateMenu.checkEnabled {
		t.checkUpdate.Enable()
	} else {
		t.checkUpdate.Disable()
	}
	t.upgrade.SetTitle(updateMenu.upgradeTitle)
	if updateMenu.upgradeEnabled {
		t.upgrade.Enable()
	} else {
		t.upgrade.Disable()
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

func (t *windowsTray) performUpgrade(force bool) {
	var err error
	if force {
		err = t.updater.ForceUpgrade(os.Args[1:])
	} else {
		err = t.updater.PrepareUpgrade(os.Args[1:])
	}
	if err != nil {
		logWarnModule("update", "prepare upgrade failed: %v", err)
		t.syncMenuState()
		return
	}
	logInfoModule("update", "upgrade prepared, restarting application")
	t.Close()
	os.Exit(0)
}

func (t *windowsTray) Close() {
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
