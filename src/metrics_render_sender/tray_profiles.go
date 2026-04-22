//go:build (linux && cgo) || (windows && cgo)

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

const trayProfileRefreshInterval = 2 * time.Second

type trayProfileMenuState struct {
	mu          sync.Mutex
	root        *systray.MenuItem
	placeholder *systray.MenuItem
	entries     map[string]*trayProfileMenuEntry
	configPath  string
	lastRefresh time.Time
}

type trayProfileMenuEntry struct {
	name string
	item *systray.MenuItem
}

func newTrayProfileMenuState() *trayProfileMenuState {
	state := &trayProfileMenuState{
		root:    systray.AddMenuItem("Switch Profile", "Switch active configuration profile"),
		entries: make(map[string]*trayProfileMenuEntry),
	}
	state.placeholder = state.root.AddSubMenuItem("Loading...", "Loading profiles")
	state.placeholder.Disable()

	configPath, err := getUserConfigPath()
	if err != nil {
		state.root.SetTitle("Profiles Unavailable")
		state.root.Disable()
		state.placeholder.SetTitle("Config path unavailable")
		return state
	}
	state.configPath = configPath
	state.refresh(true)
	return state
}

func (s *trayProfileMenuState) refresh(force bool) {
	if s == nil {
		return
	}

	s.mu.Lock()
	if !force && !s.lastRefresh.IsZero() && time.Since(s.lastRefresh) < trayProfileRefreshInterval {
		s.mu.Unlock()
		return
	}
	s.lastRefresh = time.Now()
	configPath := s.configPath
	s.mu.Unlock()

	if strings.TrimSpace(configPath) == "" {
		s.root.SetTitle("Profiles Unavailable")
		s.root.Disable()
		s.placeholder.SetTitle("Config path unavailable")
		s.placeholder.Show()
		return
	}

	profiles, err := GetProfileManagerWithPath(configPath)
	if err != nil {
		s.root.SetTitle("Profiles Unavailable")
		s.root.Disable()
		s.placeholder.SetTitle("Profiles unavailable")
		s.placeholder.Show()
		return
	}

	items, err := profiles.List()
	if err != nil {
		s.root.SetTitle("Profiles Unavailable")
		s.root.Disable()
		s.placeholder.SetTitle("Profiles unavailable")
		s.placeholder.Show()
		return
	}

	active := strings.TrimSpace(profiles.ActiveName())
	s.root.SetTitle("Switch Profile")
	s.root.Enable()

	s.mu.Lock()
	defer s.mu.Unlock()

	visible := make(map[string]struct{}, len(items))
	for _, info := range items {
		name := strings.TrimSpace(info.Name)
		if name == "" {
			continue
		}
		visible[name] = struct{}{}
		entry := s.entries[name]
		if entry == nil {
			item := s.root.AddSubMenuItemCheckbox(name, "Switch active configuration profile", false)
			entry = &trayProfileMenuEntry{name: name, item: item}
			s.entries[name] = entry
			go s.handleProfileClicks(entry)
		}
		entry.item.SetTitle(name)
		entry.item.Show()
		if name == active {
			entry.item.Check()
			entry.item.Disable()
			entry.item.SetTooltip(fmt.Sprintf("Active profile: %s", name))
		} else {
			entry.item.Uncheck()
			entry.item.Enable()
			entry.item.SetTooltip(fmt.Sprintf("Switch to profile: %s", name))
		}
	}

	for name, entry := range s.entries {
		if _, ok := visible[name]; ok {
			continue
		}
		entry.item.Hide()
	}

	if len(visible) == 0 {
		s.placeholder.SetTitle("No profiles")
		s.placeholder.Show()
		return
	}
	s.placeholder.Hide()
}

func (s *trayProfileMenuState) handleProfileClicks(entry *trayProfileMenuEntry) {
	if s == nil || entry == nil || entry.item == nil {
		return
	}
	for {
		_, ok := <-entry.item.ClickedCh
		if !ok {
			return
		}
		if err := switchTrayProfile(entry.name); err != nil {
			logWarnModule("tray", "switch profile failed: %v", err)
		}
		s.refresh(true)
	}
}

func switchTrayProfile(name string) error {
	profileName := strings.TrimSpace(name)
	if profileName == "" {
		return fmt.Errorf("profile name is empty")
	}

	configPath, err := getUserConfigPath()
	if err != nil {
		return err
	}
	profiles, err := GetProfileManagerWithPath(configPath)
	if err != nil {
		return err
	}

	cfg, err := profiles.Switch(profileName)
	if err != nil {
		return err
	}
	if err := ApplyConfigToSharedWebAPI(cfg); err != nil {
		return err
	}
	UpdateRunningConfigStore(cfg)
	logInfoModule("tray", "active profile switched to %s", profileName)
	return nil
}
