package output

import (
	"fmt"
	"strings"
)

const (
	TypeMemImg   = "memimg"
	TypeAX206USB = "ax206usb"
	TypeHTTPPush = "httppush"
)

type ConfigSummary struct {
	Configs   []OutputConfig
	Types     []string
	HasMemImg bool
}

type OutputConfig struct {
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	Format      string `json:"format,omitempty"`
	Quality     int    `json:"quality,omitempty"`
	ReconnectMS int    `json:"reconnect_ms,omitempty"`
}

func normalizeOutputTypeName(typeName string) string {
	return strings.ToLower(strings.TrimSpace(typeName))
}

func normalizeHTTPPushFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "png":
		return "png"
	case "jpg", "jpeg":
		return "jpeg"
	default:
		return "jpeg"
	}
}

func normalizeHTTPPushQuality(quality int) int {
	if quality <= 0 {
		return 80
	}
	if quality < 1 {
		return 1
	}
	if quality > 100 {
		return 100
	}
	return quality
}

func normalizeAX206ReconnectMS(reconnectMS int) int {
	if reconnectMS <= 0 {
		return 3000
	}
	if reconnectMS < 100 {
		return 100
	}
	if reconnectMS > 60000 {
		return 60000
	}
	return reconnectMS
}

func normalizeSingleConfig(raw OutputConfig) (OutputConfig, bool) {
	cfg := OutputConfig{
		Type: normalizeOutputTypeName(raw.Type),
	}
	switch cfg.Type {
	case TypeMemImg:
		return cfg, true
	case TypeAX206USB:
		cfg.ReconnectMS = normalizeAX206ReconnectMS(raw.ReconnectMS)
		return cfg, true
	case TypeHTTPPush:
		cfg.URL = strings.TrimSpace(raw.URL)
		cfg.Format = normalizeHTTPPushFormat(raw.Format)
		cfg.Quality = normalizeHTTPPushQuality(raw.Quality)
		return cfg, true
	default:
		return OutputConfig{}, false
	}
}

func NormalizeConfigs(configs []OutputConfig) []OutputConfig {
	normalized := make([]OutputConfig, 0, len(configs))
	seenSingleton := map[string]struct{}{}

	for _, raw := range configs {
		cfg, ok := normalizeSingleConfig(raw)
		if !ok {
			continue
		}
		if _, exists := seenSingleton[cfg.Type]; exists {
			continue
		}
		seenSingleton[cfg.Type] = struct{}{}
		normalized = append(normalized, cfg)
	}

	if len(normalized) == 0 {
		return []OutputConfig{}
	}
	return normalized
}

func ResolveConfigs(configs []OutputConfig, forceMemImg bool) []OutputConfig {
	resolved := NormalizeConfigs(configs)
	if !forceMemImg {
		return resolved
	}
	for _, cfg := range resolved {
		if cfg.Type == TypeMemImg {
			return resolved
		}
	}
	return append(resolved, OutputConfig{Type: TypeMemImg})
}

func ConfigsFromTypes(types []string) []OutputConfig {
	configs := make([]OutputConfig, 0, len(types))
	for _, item := range types {
		configs = append(configs, OutputConfig{Type: item})
	}
	return configs
}

func TypeNames(configs []OutputConfig) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(configs))
	for _, item := range configs {
		typeName := normalizeOutputTypeName(item.Type)
		if typeName == "" {
			continue
		}
		if _, exists := seen[typeName]; exists {
			continue
		}
		seen[typeName] = struct{}{}
		out = append(out, typeName)
	}
	return out
}

func HasType(configs []OutputConfig, typeName string) bool {
	target := normalizeOutputTypeName(typeName)
	if target == "" {
		return false
	}
	for _, item := range configs {
		if normalizeOutputTypeName(item.Type) == target {
			return true
		}
	}
	return false
}

func DescribeConfigs(configs []OutputConfig) ConfigSummary {
	return ConfigSummary{
		Configs:   configs,
		Types:     TypeNames(configs),
		HasMemImg: HasType(configs, TypeMemImg),
	}
}

func EqualConfigs(left, right []OutputConfig) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		lCfg, lOK := normalizeSingleConfig(left[idx])
		rCfg, rOK := normalizeSingleConfig(right[idx])
		if !lOK || !rOK {
			return false
		}
		if lCfg.Type != rCfg.Type {
			return false
		}
		if lCfg.URL != rCfg.URL {
			return false
		}
		if lCfg.Format != rCfg.Format {
			return false
		}
		if lCfg.Quality != rCfg.Quality {
			return false
		}
		if lCfg.ReconnectMS != rCfg.ReconnectMS {
			return false
		}
	}
	return true
}

func NormalizeTypes(types []string) []string {
	return TypeNames(NormalizeConfigs(ConfigsFromTypes(types)))
}

func ResolveTypes(types []string, forceMemImg bool) []string {
	return TypeNames(ResolveConfigs(ConfigsFromTypes(types), forceMemImg))
}

func ResolveConfigSummary(configs []OutputConfig, forceMemImg bool) ConfigSummary {
	return DescribeConfigs(ResolveConfigs(configs, forceMemImg))
}

func BuildManager(configs []OutputConfig, forceMemImg bool) (*OutputManager, []OutputConfig) {
	summary := ResolveConfigSummary(configs, forceMemImg)
	manager := NewOutputManager()

	httpPushIndex := 0
	for _, cfg := range summary.Configs {
		switch cfg.Type {
		case TypeMemImg:
			manager.AddHandler(NewMemImgOutputHandler())
		case TypeAX206USB:
			handler, err := NewSharedAX206USBOutputHandler(cfg)
			if err != nil {
				logErrorModule("ax206usb", "Handler creation failed: %v", err)
				continue
			}
			manager.AddHandler(handler)
		case TypeHTTPPush:
			httpPushIndex++
			typeName := TypeHTTPPush
			if httpPushIndex > 1 {
				typeName = fmt.Sprintf("%s_%d", TypeHTTPPush, httpPushIndex)
			}
			handler := NewHTTPPushOutputHandler(cfg, typeName)
			manager.AddHandler(handler)
		}
	}

	return manager, summary.Configs
}
