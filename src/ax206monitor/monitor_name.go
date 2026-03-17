package main

import "strings"

func normalizeMonitorNameInput(name string) string {
	return strings.TrimSpace(name)
}

func normalizeMonitorAliasInput(name string) string {
	return normalizeMonitorNameInput(name)
}

func normalizeMonitorAlias(name string) string {
	return normalizeMonitorNameInput(name)
}

func isMonitorAliasName(string) bool {
	return false
}

func monitorAliasNames() []string {
	return nil
}

func monitorAliasLabels() map[string]string {
	return nil
}

func resolveMonitorAliasWithItems(string, map[string]*CollectItem) string {
	return ""
}

func buildMonitorAliasResolution(map[string]*CollectItem) map[string]string {
	return map[string]string{}
}
