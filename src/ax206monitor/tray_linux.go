//go:build linux && cgo

package main

import (
	_ "embed"
	"fmt"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

//go:embed assets/tray.png
var trayIconPNG []byte

type linuxTray struct {
	web     *WebServerProcess
	readyCh chan struct{}
	stopCh  chan struct{}
	mu      sync.Mutex
	closed  bool
	openWeb *systray.MenuItem
	openUI  *systray.MenuItem
}

func StartTray(webController *WebServerProcess) (TrayHandle, error) {
	if webController == nil {
		return nil, fmt.Errorf("web controller is nil")
	}

	tray := &linuxTray{
		web:     webController,
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
	systray.SetTitle("AX206 Monitor")
	systray.SetTooltip("AX206 Monitor")

	t.openWeb = systray.AddMenuItem("Open Web Server", "Start web configuration server")
	t.openUI = systray.AddMenuItem("Open Web Editor", "Open web editor in browser")
	t.openUI.Disable()
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
		return
	}
	t.openWeb.SetTitle("Open Web Server")
	t.openUI.Disable()
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
