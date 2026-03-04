package main

import "ax206monitor/output"

const (
	outputTypeMemImg   = output.TypeMemImg
	outputTypeAX206USB = output.TypeAX206USB
)

func normalizeOutputTypes(types []string) []string {
	return output.NormalizeTypes(types)
}

func resolveOutputTypes(cfg *MonitorConfig, forceMemImg bool) []string {
	var types []string
	if cfg != nil {
		types = cfg.OutputTypes
	}
	return output.ResolveTypes(types, forceMemImg)
}

func buildOutputManager(cfg *MonitorConfig, forceMemImg bool) (*OutputManager, []string) {
	var types []string
	if cfg != nil {
		types = cfg.OutputTypes
	}
	return output.BuildManager(types, forceMemImg)
}
