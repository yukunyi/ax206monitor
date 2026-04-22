//go:build (linux && cgo) || (windows && cgo)

package main

import (
	"fmt"
	"strings"

	"github.com/getlantern/systray"
)

type trayWebBindMenuState struct {
	root     *systray.MenuItem
	loopback *systray.MenuItem
	public   *systray.MenuItem
	web      *WebServerProcess
}

func newTrayWebBindMenuState(web *WebServerProcess) *trayWebBindMenuState {
	state := &trayWebBindMenuState{
		root: systray.AddMenuItem("Web Listen Address", "Switch web server listen address"),
		web:  web,
	}
	state.loopback = state.root.AddSubMenuItemCheckbox(defaultWebBindHost, "Listen on local loopback only", false)
	state.public = state.root.AddSubMenuItemCheckbox(publicWebBindHost, "Listen on all interfaces", false)
	state.refresh()
	go state.handleLoopbackClicks()
	go state.handlePublicClicks()
	return state
}

func (s *trayWebBindMenuState) currentHost() string {
	if s == nil || s.web == nil {
		return defaultWebBindHost
	}
	return s.web.ListenHost()
}

func (s *trayWebBindMenuState) refresh() {
	if s == nil {
		return
	}
	current := s.currentHost()
	s.root.SetTitle(fmt.Sprintf("Web Listen Address (%s)", current))

	if current == publicWebBindHost {
		s.public.Check()
		s.public.Disable()
		s.loopback.Uncheck()
		s.loopback.Enable()
		return
	}

	s.loopback.Check()
	s.loopback.Disable()
	s.public.Uncheck()
	s.public.Enable()
}

func (s *trayWebBindMenuState) handleLoopbackClicks() {
	if s == nil || s.loopback == nil {
		return
	}
	for {
		_, ok := <-s.loopback.ClickedCh
		if !ok {
			return
		}
		if err := s.switchHost(defaultWebBindHost); err != nil {
			logWarnModule("tray", "switch web bind host failed: %v", err)
		}
		s.refresh()
	}
}

func (s *trayWebBindMenuState) handlePublicClicks() {
	if s == nil || s.public == nil {
		return
	}
	for {
		_, ok := <-s.public.ClickedCh
		if !ok {
			return
		}
		if err := s.switchHost(publicWebBindHost); err != nil {
			logWarnModule("tray", "switch web bind host failed: %v", err)
		}
		s.refresh()
	}
}

func (s *trayWebBindMenuState) switchHost(host string) error {
	if s == nil || s.web == nil {
		return fmt.Errorf("web controller is nil")
	}
	nextHost := normalizeWebBindHost(host)
	if strings.TrimSpace(nextHost) == strings.TrimSpace(s.web.ListenHost()) {
		return nil
	}
	if err := saveWebBindHost(nextHost); err != nil {
		return err
	}
	wasRunning := s.web.IsRunning()
	s.web.SetListenHost(nextHost)
	if !wasRunning {
		logInfoModule("tray", "web listen host set to %s", nextHost)
		return nil
	}
	if err := s.web.Stop(); err != nil {
		return err
	}
	if err := s.web.Start(); err != nil {
		return err
	}
	logInfoModule("tray", "web listen host switched to %s", nextHost)
	return nil
}
