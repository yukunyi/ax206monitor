package main

import (
	"ax206monitor/output"
)

const (
	outputTypeMemImg   = output.TypeMemImg
	outputTypeAX206USB = output.TypeAX206USB
	outputTypeHTTPPush = output.TypeHTTPPush
	outputTypeTCPPush  = output.TypeTCPPush
)

var supportedOutputTypes = []string{
	outputTypeMemImg,
	outputTypeAX206USB,
	outputTypeHTTPPush,
	outputTypeTCPPush,
}

func getSupportedOutputTypes() []string {
	out := make([]string, len(supportedOutputTypes))
	copy(out, supportedOutputTypes)
	return out
}

func getDefaultOutputConfigs() []OutputConfig {
	return []OutputConfig{}
}

func getDefaultOutputTypes() []string {
	return []string{}
}

func normalizeOutputConfigs(configs []OutputConfig) []OutputConfig {
	return output.NormalizeConfigs(configs)
}

func outputConfigsFromTypes(types []string) []OutputConfig {
	return output.ConfigsFromTypes(types)
}

func outputEnabledTypeNames(configs []OutputConfig) []string {
	return output.EnabledTypeNames(configs)
}

func describeOutputConfigs(configs []OutputConfig) OutputConfigSummary {
	return output.DescribeConfigs(configs)
}

func outputConfigsEqual(left, right []OutputConfig) bool {
	return output.EqualConfigs(left, right)
}

func resolveOutputConfigSummaryFromList(configs []OutputConfig, forceMemImg bool) OutputConfigSummary {
	return output.ResolveConfigSummary(configs, forceMemImg)
}

func buildOutputManager(cfg *MonitorConfig, forceMemImg bool) (*OutputManager, []OutputConfig) {
	var configs []OutputConfig
	if cfg != nil {
		configs = cfg.Outputs
	}
	return output.BuildManager(configs, forceMemImg)
}
