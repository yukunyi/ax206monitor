package main

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const monitorAliasPrefix = "alias."

var monitorAliasLegacyMap = map[string]string{
	"gpu_fps":          "alias.gpu.fps",
	"alias.network.ip": "alias.net.ip",
}

var monitorAliasResolvers = map[string]func(items map[string]*CollectItem) string{
	"alias.cpu.usage":    func(items map[string]*CollectItem) string { return resolveByExactKeys(items, "go_native.cpu.usage") },
	"alias.cpu.temp":     func(items map[string]*CollectItem) string { return resolveByExactKeys(items, "go_native.cpu.temp") },
	"alias.cpu.freq":     func(items map[string]*CollectItem) string { return resolveByExactKeys(items, "go_native.cpu.freq") },
	"alias.cpu.max_freq": func(items map[string]*CollectItem) string { return resolveByExactKeys(items, "go_native.cpu.max_freq") },
	"alias.cpu.power": func(items map[string]*CollectItem) string {
		return resolveByTokenScore(items, aliasTokenRule{
			RequireAny: []string{"cpu", "package"},
			MustHave:   []string{"power"},
			PreferUnit: []string{"w"},
		})
	},
	"alias.memory.usage": func(items map[string]*CollectItem) string { return resolveByExactKeys(items, "go_native.memory.usage") },
	"alias.memory.used":  func(items map[string]*CollectItem) string { return resolveByExactKeys(items, "go_native.memory.used") },
	"alias.gpu.fps": func(items map[string]*CollectItem) string {
		if key := resolveByExactKeys(items, "rtss_fps"); key != "" {
			return key
		}
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"fps"},
			PreferUnit: []string{"fps"},
		})
	},
	"alias.gpu.usage": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"usage", "util", "utilization", "load"},
			PreferUnit: []string{"%"},
		})
	},
	"alias.gpu.power": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"power", "package power", "board power", "gpu power"},
			PreferUnit: []string{"w"},
		})
	},
	"alias.gpu.vram": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"vram", "video memory", "memory usage", "fb usage", "memory used"},
			PreferUnit: []string{"%"},
		})
	},
	"alias.gpu.temp": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"temp", "temperature"},
			PreferUnit: []string{"°c", "c"},
		})
	},
	"alias.gpu.fan": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"fan", "rpm"},
			PreferUnit: []string{"rpm"},
		})
	},
	"alias.gpu.freq": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"freq", "clock"},
			PreferUnit: []string{"mhz", "ghz", "hz"},
		})
	},
	"alias.gpu.max_freq": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"max", "boost", "limit"},
			RequireAny: []string{"clock", "freq"},
			PreferUnit: []string{"mhz", "ghz", "hz"},
		})
	},
	"alias.net.upload": func(items map[string]*CollectItem) string { return resolvePreferredNetMetric(items, "upload") },
	"alias.net.download": func(items map[string]*CollectItem) string {
		return resolvePreferredNetMetric(items, "download")
	},
	"alias.net.ip":        func(items map[string]*CollectItem) string { return resolvePreferredNetMetric(items, "ip") },
	"alias.net.interface": func(items map[string]*CollectItem) string { return resolvePreferredNetMetric(items, "interface") },
	"alias.system.time": func(items map[string]*CollectItem) string {
		return resolveByExactKeys(items, "go_native.system.current_time")
	},
	"alias.system.hostname": func(items map[string]*CollectItem) string {
		return resolveByExactKeys(items, "go_native.system.hostname")
	},
	"alias.system.load": func(items map[string]*CollectItem) string {
		return resolveByExactKeys(items, "go_native.system.load_avg")
	},
	"alias.system.resolution": func(items map[string]*CollectItem) string {
		return resolveByExactKeys(items, "go_native.system.resolution")
	},
	"alias.system.refresh_rate": func(items map[string]*CollectItem) string {
		return resolveByExactKeys(items, "go_native.system.refresh_rate")
	},
	"alias.system.display": func(items map[string]*CollectItem) string {
		return resolveByExactKeys(items, "go_native.system.display")
	},
	"alias.disk.temp": func(items map[string]*CollectItem) string {
		return resolveByIndexedPrefix(items, "go_native.disk.", ".temp")
	},
	"alias.fan.cpu": func(items map[string]*CollectItem) string {
		return resolveByTokenScore(items, aliasTokenRule{
			MustHave:   []string{"fan"},
			RequireAny: []string{"cpu"},
			PreferUnit: []string{"rpm"},
		})
	},
	"alias.fan.gpu": func(items map[string]*CollectItem) string {
		return resolveGPUByTokens(items, aliasTokenRule{
			MustHave:   []string{"fan"},
			PreferUnit: []string{"rpm"},
		})
	},
	"alias.fan.system": func(items map[string]*CollectItem) string {
		return resolveByTokenScore(items, aliasTokenRule{
			MustHave:   []string{"fan"},
			RequireAny: []string{"system", "chassis", "case"},
			PreferUnit: []string{"rpm"},
		})
	},
}

var monitorAliasNamesSorted = func() []string {
	keys := make([]string, 0, len(monitorAliasResolvers))
	for name := range monitorAliasResolvers {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}()

var monitorAliasUpperTokenMap = map[string]string{
	"cpu":  "CPU",
	"gpu":  "GPU",
	"ip":   "IP",
	"fps":  "FPS",
	"vram": "VRAM",
}

func monitorAliasLabels() map[string]string {
	labels := make(map[string]string, len(monitorAliasNamesSorted))
	for _, name := range monitorAliasNamesSorted {
		if label := buildMonitorAliasLabel(name); label != "" {
			labels[name] = label
		}
	}
	return labels
}

func buildMonitorAliasLabel(name string) string {
	normalized := normalizeMonitorAliasInput(name)
	if !strings.HasPrefix(normalized, monitorAliasPrefix) {
		return ""
	}
	body := strings.TrimPrefix(normalized, monitorAliasPrefix)
	if body == "" {
		return ""
	}
	parts := strings.Split(body, ".")
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.ToLower(strings.TrimSpace(part))
		if token == "" {
			continue
		}
		if upper, ok := monitorAliasUpperTokenMap[token]; ok {
			words = append(words, upper)
			continue
		}
		words = append(words, strings.ToUpper(token[:1])+token[1:])
	}
	if len(words) == 0 {
		return ""
	}
	return strings.Join(words, " ")
}

func normalizeMonitorAliasInput(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	lowered := strings.ToLower(trimmed)
	if canonical, ok := monitorAliasLegacyMap[lowered]; ok {
		return canonical
	}
	if strings.HasPrefix(lowered, monitorAliasPrefix) {
		return lowered
	}
	return trimmed
}

func isMonitorAliasName(name string) bool {
	normalized := normalizeMonitorAliasInput(name)
	if normalized == "" {
		return false
	}
	_, ok := monitorAliasResolvers[normalized]
	return ok
}

func monitorAliasNames() []string {
	result := make([]string, len(monitorAliasNamesSorted))
	copy(result, monitorAliasNamesSorted)
	return result
}

func resolveMonitorAliasWithItems(name string, items map[string]*CollectItem) string {
	normalized := normalizeMonitorAliasInput(name)
	if normalized == "" {
		return ""
	}
	if items == nil {
		return normalized
	}
	if _, exists := items[normalized]; exists {
		return normalized
	}
	resolver, ok := monitorAliasResolvers[normalized]
	if !ok {
		return normalized
	}
	target := strings.TrimSpace(resolver(items))
	if target == "" {
		return normalized
	}
	if _, exists := items[target]; !exists {
		return normalized
	}
	return target
}

func buildMonitorAliasResolution(items map[string]*CollectItem) map[string]string {
	result := make(map[string]string, len(monitorAliasResolvers))
	for _, aliasName := range monitorAliasNamesSorted {
		target := resolveMonitorAliasWithItems(aliasName, items)
		if target == "" || target == aliasName {
			continue
		}
		result[aliasName] = target
	}
	return result
}

func resolveByExactKeys(items map[string]*CollectItem, keys ...string) string {
	for _, key := range keys {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		if _, exists := items[name]; exists {
			return name
		}
	}
	return ""
}

func resolveByIndexedPrefix(items map[string]*CollectItem, prefix string, suffix string) string {
	indexes := make([]int, 0, 4)
	seen := make(map[int]struct{}, 4)
	for key := range items {
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
			continue
		}
		trimmed := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
		idx, err := strconv.Atoi(trimmed)
		if err != nil || idx <= 0 {
			continue
		}
		if _, ok := seen[idx]; ok {
			continue
		}
		seen[idx] = struct{}{}
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	if len(indexes) == 0 {
		return ""
	}
	return fmt.Sprintf("%s%d%s", prefix, indexes[0], suffix)
}

type aliasTokenRule struct {
	MustHave   []string
	RequireAny []string
	PreferUnit []string
}

func resolveGPUByTokens(items map[string]*CollectItem, rule aliasTokenRule) string {
	bestName := ""
	bestScore := 0
	for key, item := range items {
		score := scoreTokenMatch(item, key, rule)
		if score <= 0 {
			continue
		}
		lowerText := itemTextLower(item, key)
		if !containsAnyToken(lowerText, "gpu", "graphics", "vram", "rtss", "nvidia", "geforce", "amd", "radeon", "intel", "igpu") {
			continue
		}
		score += scoreGPUPreference(lowerText)
		if score > bestScore || (score == bestScore && (bestName == "" || key < bestName)) {
			bestName = key
			bestScore = score
		}
	}
	return bestName
}

func resolveByTokenScore(items map[string]*CollectItem, rule aliasTokenRule) string {
	bestName := ""
	bestScore := 0
	for key, item := range items {
		score := scoreTokenMatch(item, key, rule)
		if score <= 0 {
			continue
		}
		if score > bestScore || (score == bestScore && (bestName == "" || key < bestName)) {
			bestName = key
			bestScore = score
		}
	}
	return bestName
}

func scoreTokenMatch(item *CollectItem, key string, rule aliasTokenRule) int {
	if item == nil {
		return 0
	}
	text := itemTextLower(item, key)
	score := 0
	if len(rule.MustHave) > 0 {
		hit := false
		for _, token := range rule.MustHave {
			normalized := strings.ToLower(strings.TrimSpace(token))
			if normalized == "" {
				continue
			}
			if strings.Contains(text, normalized) {
				score += 70
				hit = true
			}
		}
		if !hit {
			return 0
		}
	}
	if len(rule.RequireAny) > 0 && !containsAnyToken(text, rule.RequireAny...) {
		return 0
	}
	if len(rule.RequireAny) > 0 {
		score += 30
	}
	if unit := itemUnitLower(item); unit != "" && len(rule.PreferUnit) > 0 && containsAnyToken(unit, rule.PreferUnit...) {
		score += 35
	}
	if item.IsAvailable() {
		score += 8
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), "go_native.") {
		score += 12
	}
	return score
}

func itemTextLower(item *CollectItem, key string) string {
	name := strings.ToLower(strings.TrimSpace(key))
	label := ""
	if item != nil {
		label = strings.ToLower(strings.TrimSpace(item.GetLabel()))
	}
	if label == "" {
		return name
	}
	return name + " " + label
}

func itemUnitLower(item *CollectItem) string {
	if item == nil {
		return ""
	}
	value := item.GetValue()
	if value == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value.Unit))
}

func scoreGPUPreference(text string) int {
	score := 0
	if containsAnyToken(text, "nvidia", "geforce", "rtx", "gtx", "amd", "radeon", "rx") {
		score += 140
	}
	if containsAnyToken(text, "intel", "igpu", "integrated", "uhd", "iris", "apu") {
		score -= 90
	}
	return score
}

func containsAnyToken(text string, tokens ...string) bool {
	normalizedText := strings.ToLower(strings.TrimSpace(text))
	if normalizedText == "" {
		return false
	}
	for _, token := range tokens {
		needle := strings.ToLower(strings.TrimSpace(token))
		if needle == "" {
			continue
		}
		if strings.Contains(normalizedText, needle) {
			return true
		}
	}
	return false
}

func resolvePreferredNetMetric(items map[string]*CollectItem, metric string) string {
	metric = strings.ToLower(strings.TrimSpace(metric))
	if metric == "" {
		return ""
	}
	slot := selectPreferredNetworkSlot(items)
	if slot > 0 {
		key := fmt.Sprintf("go_native.net.%d.%s", slot, metric)
		if _, exists := items[key]; exists {
			return key
		}
	}
	bestKey := ""
	bestIndex := 0
	for key := range items {
		idx, suffix, ok := parseGoNativeNetKey(key)
		if !ok || suffix != metric {
			continue
		}
		if bestKey == "" || idx < bestIndex {
			bestKey = key
			bestIndex = idx
		}
	}
	return bestKey
}

func parseGoNativeNetKey(name string) (int, string, bool) {
	trimmed := strings.TrimSpace(name)
	if !strings.HasPrefix(trimmed, "go_native.net.") {
		return 0, "", false
	}
	rest := strings.TrimPrefix(trimmed, "go_native.net.")
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return 0, "", false
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil || index <= 0 {
		return 0, "", false
	}
	return index, strings.TrimSpace(parts[1]), true
}

type netSlotState struct {
	index    int
	iface    string
	ifaceOK  bool
	ip       net.IP
	ipOK     bool
	score    int
	hasEntry bool
}

func selectPreferredNetworkSlot(items map[string]*CollectItem) int {
	if len(items) == 0 {
		return 0
	}
	states := make(map[int]*netSlotState)
	for key := range items {
		idx, metric, ok := parseGoNativeNetKey(key)
		if !ok {
			continue
		}
		state := states[idx]
		if state == nil {
			state = &netSlotState{index: idx}
			states[idx] = state
		}
		state.hasEntry = true
		switch metric {
		case "interface":
			value := collectItemStringValue(items[key])
			if value != "" {
				state.iface = value
				state.ifaceOK = true
			}
		case "ip":
			value := collectItemStringValue(items[key])
			if ip := net.ParseIP(strings.TrimSpace(value)); ip != nil {
				state.ip = ip
				state.ipOK = true
			}
		}
	}
	if len(states) == 0 {
		return 0
	}

	preferredIface := strings.ToLower(strings.TrimSpace(getDefaultRouteInterfaceNameCached()))
	bestIndex := 0
	bestScore := -1
	for _, state := range states {
		score := 1
		if state.ifaceOK {
			ifaceLower := strings.ToLower(strings.TrimSpace(state.iface))
			if ifaceLower != "" && ifaceLower == preferredIface {
				score += 500
			}
			if ifaceLower != "" && !isLikelyVirtualInterface(ifaceLower) {
				score += 70
			}
		}
		if state.ipOK {
			if !state.ip.IsLoopback() && !state.ip.IsLinkLocalUnicast() {
				score += 260
			}
			if state.ip.IsGlobalUnicast() {
				score += 120
			}
		}
		state.score = score
		if score > bestScore || (score == bestScore && (bestIndex == 0 || state.index < bestIndex)) {
			bestScore = score
			bestIndex = state.index
		}
	}
	return bestIndex
}

func collectItemStringValue(item *CollectItem) string {
	if item == nil {
		return ""
	}
	value := item.GetValue()
	if value == nil || value.Value == nil {
		return ""
	}
	switch v := value.Value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "-" {
			return ""
		}
		return trimmed
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", v))
		if text == "-" {
			return ""
		}
		return text
	}
}

func isLikelyVirtualInterface(name string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return true
	}
	virtualPrefixes := []string{
		"docker", "br-", "veth", "virbr", "vmnet", "vboxnet",
		"tap", "tun", "lo", "dummy", "bond", "team", "vlan",
	}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

var defaultRouteInterfaceCache struct {
	mutex      sync.Mutex
	name       string
	expires    time.Time
	refreshing bool
}

func getDefaultRouteInterfaceNameCached() string {
	now := time.Now()
	defaultRouteInterfaceCache.mutex.Lock()
	if now.Before(defaultRouteInterfaceCache.expires) {
		name := defaultRouteInterfaceCache.name
		defaultRouteInterfaceCache.mutex.Unlock()
		return name
	}
	name := defaultRouteInterfaceCache.name
	if !defaultRouteInterfaceCache.refreshing {
		defaultRouteInterfaceCache.refreshing = true
		go refreshDefaultRouteInterfaceName()
	}
	defaultRouteInterfaceCache.mutex.Unlock()
	return name
}

func refreshDefaultRouteInterfaceName() {
	name := detectDefaultRouteInterfaceName()
	now := time.Now()
	defaultRouteInterfaceCache.mutex.Lock()
	defaultRouteInterfaceCache.name = name
	defaultRouteInterfaceCache.expires = now.Add(8 * time.Second)
	defaultRouteInterfaceCache.refreshing = false
	defaultRouteInterfaceCache.mutex.Unlock()
}

func detectDefaultRouteInterfaceName() string {
	conn, err := net.DialTimeout("udp4", "8.8.8.8:53", 120*time.Millisecond)
	if err != nil {
		return ""
	}
	defer conn.Close()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr == nil || addr.IP == nil {
		return ""
	}
	targetIP := addr.IP

	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		addrs, addrErr := iface.Addrs()
		if addrErr != nil {
			continue
		}
		for _, address := range addrs {
			ip := parseIPFromAddr(address.String())
			if ip == nil {
				continue
			}
			if ip.Equal(targetIP) {
				return iface.Name
			}
		}
	}
	return ""
}

func parseIPFromAddr(raw string) net.IP {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, "/") {
		ip, _, err := net.ParseCIDR(trimmed)
		if err == nil {
			return ip
		}
	}
	return net.ParseIP(trimmed)
}
