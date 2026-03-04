package builtinprofiles

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed builtin_configs/*.json
var profileAssets embed.FS

func Load[T any](normalize func(*T), clone func(*T) *T) (map[string]*T, error) {
	result := make(map[string]*T)
	entries, err := fs.ReadDir(profileAssets, "builtin_configs")
	if err != nil {
		return nil, fmt.Errorf("failed to read built-in profiles: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		data, err := fs.ReadFile(profileAssets, "builtin_configs/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read built-in profile %s: %w", entry.Name(), err)
		}
		var cfg T
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("invalid built-in profile %s: %w", entry.Name(), err)
		}
		if normalize != nil {
			normalize(&cfg)
		}
		if clone != nil {
			result[name] = clone(&cfg)
			continue
		}
		cfgCopy := cfg
		result[name] = &cfgCopy
	}

	return result, nil
}

func SortedNames[T any](items map[string]*T) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
