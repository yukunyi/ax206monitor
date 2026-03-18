package main

import "strings"

func collectItemMonitorRefs(item *ItemConfig) []string {
	if item == nil {
		return nil
	}
	if item.Type == itemTypeFullTable {
		return fullTableMonitorRefs(item)
	}
	name := normalizeMonitorAlias(item.Monitor)
	if name == "" {
		return nil
	}
	return []string{name}
}

func appendUniqueMonitorRefs(dst []string, seen map[string]struct{}, refs []string) []string {
	for _, ref := range refs {
		name := normalizeMonitorAlias(ref)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		dst = append(dst, name)
	}
	return dst
}

func isRTSSMonitorRef(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(normalized, "rtss_")
}
