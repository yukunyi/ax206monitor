package main

import (
	"ax206monitor/builtinprofiles"
)

func loadBuiltinProfiles() (map[string]*MonitorConfig, error) {
	return builtinprofiles.Load[MonitorConfig](normalizeMonitorConfig, cloneMonitorConfig)
}

func sortedBuiltinProfileNames(items map[string]*MonitorConfig) []string {
	return builtinprofiles.SortedNames(items)
}
