package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type ProfileInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
	Size      int64  `json:"size"`
}

type ProfileManager struct {
	currentConfigPath string
	profilesDir       string
	activeProfileFile string
	mu                sync.RWMutex
}

var profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

var (
	globalProfileManagerMu sync.Mutex
	globalProfileManager   *ProfileManager
	globalProfilePath      string
)

func NewProfileManager(currentConfigPath string) (*ProfileManager, error) {
	configDir := filepath.Dir(currentConfigPath)
	pm := &ProfileManager{
		currentConfigPath: currentConfigPath,
		profilesDir:       filepath.Join(configDir, "profiles"),
		activeProfileFile: filepath.Join(configDir, "profiles", "active-profile"),
	}
	if err := os.MkdirAll(pm.profilesDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create profiles directory: %w", err)
	}
	return pm, nil
}

func GetProfileManagerWithPath(currentConfigPath string) (*ProfileManager, error) {
	globalProfileManagerMu.Lock()
	defer globalProfileManagerMu.Unlock()
	if strings.TrimSpace(currentConfigPath) == "" {
		return nil, fmt.Errorf("profile manager config path is empty")
	}
	if globalProfileManager != nil && globalProfilePath == currentConfigPath {
		return globalProfileManager, nil
	}
	pm, err := NewProfileManager(currentConfigPath)
	if err != nil {
		return nil, err
	}
	globalProfileManager = pm
	globalProfilePath = currentConfigPath
	return globalProfileManager, nil
}

func InitializeGlobalProfileManager(currentConfigPath string, baseConfig *MonitorConfig) (*ProfileManager, *MonitorConfig, error) {
	pm, err := GetProfileManagerWithPath(currentConfigPath)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := pm.Initialize(baseConfig)
	if err != nil {
		return nil, nil, err
	}
	return pm, cfg, nil
}

func (pm *ProfileManager) Initialize(baseConfig *MonitorConfig) (*MonitorConfig, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.customProfileExistsUnsafe("default") {
		seed := cloneMonitorConfig(baseConfig)
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

func (pm *ProfileManager) saveProfileUnsafe(name string, cfg *MonitorConfig) error {
	if err := validateProfileName(name); err != nil {
		return err
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

	items := make([]ProfileInfo, 0, len(entries))
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
