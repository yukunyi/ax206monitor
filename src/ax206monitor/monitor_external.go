package main

import (
	"strings"

	"ax206monitor/rtsssource"
)

type CoolerControlExternalMonitor struct {
	*BaseMonitorItem
	client *CoolerControlClient
	source string
}

func NewCoolerControlExternalMonitor(option CoolerControlMonitorOption, client *CoolerControlClient) MonitorItem {
	unit := strings.TrimSpace(option.Unit)
	minValue := 0.0
	maxValue := 0.0
	precision := 2
	switch unit {
	case "°C":
		maxValue = 120
		precision = 1
	case "%":
		maxValue = 100
		precision = 0
	case "RPM":
		precision = 0
	case "MHz":
		precision = 0
	case "W":
		precision = 1
	}
	return &CoolerControlExternalMonitor{
		BaseMonitorItem: NewBaseMonitorItem(option.Name, option.Label, minValue, maxValue, unit, precision),
		client:          client,
		source:          option.Name,
	}
}

func (m *CoolerControlExternalMonitor) Update() error {
	if m.client == nil || strings.TrimSpace(m.source) == "" {
		m.SetAvailable(false)
		return nil
	}
	value, unit, ok, err := m.client.GetMonitorValueByName(m.source)
	if err != nil || !ok {
		m.SetAvailable(false)
		return nil
	}
	if strings.TrimSpace(unit) != "" {
		m.SetUnit(unit)
	}
	m.SetValue(value)
	m.SetAvailable(true)
	return nil
}

type LibreHardwareMonitorExternalMonitor struct {
	*BaseMonitorItem
	client *LibreHardwareMonitorClient
	source string
}

func NewLibreHardwareMonitorExternalMonitor(option LibreHardwareMonitorMonitorOption, client *LibreHardwareMonitorClient) MonitorItem {
	unit := strings.TrimSpace(option.Unit)
	minValue := 0.0
	maxValue := 0.0
	precision := 2
	switch strings.ToUpper(unit) {
	case "°C":
		maxValue = 120
		precision = 1
	case "%":
		maxValue = 100
		precision = 0
	case "RPM":
		precision = 0
	case "MHZ", "GHZ", "HZ":
		precision = 0
	}
	return &LibreHardwareMonitorExternalMonitor{
		BaseMonitorItem: NewBaseMonitorItem(option.Name, option.Label, minValue, maxValue, unit, precision),
		client:          client,
		source:          option.Name,
	}
}

func (m *LibreHardwareMonitorExternalMonitor) Update() error {
	if m.client == nil || strings.TrimSpace(m.source) == "" {
		m.SetAvailable(false)
		return nil
	}
	value, unit, ok, err := m.client.GetMonitorValueByName(m.source)
	if err != nil || !ok {
		m.SetAvailable(false)
		return nil
	}
	if strings.TrimSpace(unit) != "" {
		m.SetUnit(unit)
	}
	m.SetValue(value)
	m.SetAvailable(true)
	return nil
}

type RTSSExternalMonitor struct {
	*BaseMonitorItem
	client *rtsssource.RTSSClient
	source string
}

func NewRTSSExternalMonitor(option rtsssource.RTSSMonitorOption, client *rtsssource.RTSSClient) MonitorItem {
	unit := strings.TrimSpace(option.Unit)
	minValue := 0.0
	maxValue := 0.0
	precision := 1
	switch strings.ToLower(unit) {
	case "fps":
		precision = 1
	case "ms":
		precision = 2
	default:
		precision = 0
	}
	return &RTSSExternalMonitor{
		BaseMonitorItem: NewBaseMonitorItem(option.Name, option.Label, minValue, maxValue, unit, precision),
		client:          client,
		source:          option.Name,
	}
}

func (m *RTSSExternalMonitor) Update() error {
	if m.client == nil || strings.TrimSpace(m.source) == "" {
		m.SetAvailable(false)
		return nil
	}
	value, unit, ok, err := m.client.GetMonitorValueByName(m.source)
	if err != nil || !ok {
		m.SetAvailable(false)
		return nil
	}
	if strings.TrimSpace(unit) != "" {
		m.SetUnit(unit)
	}
	m.SetValue(value)
	m.SetAvailable(true)
	return nil
}

func initializeExternalMonitorItems(registry *MonitorRegistry) {
	config := GetGlobalMonitorConfig()
	if config == nil {
		return
	}

	ccURL := config.GetCoolerControlURL()
	if ccURL != "" {
		client := GetCoolerControlClient(ccURL, config.GetCoolerControlUsername(), config.CoolerControlPassword)
		options, err := client.ListMonitorOptions()
		if err != nil {
			logWarnModule("monitor", "coolercontrol monitor init failed (url=%s): %v", ccURL, err)
		} else {
			added := 0
			for _, option := range options {
				if strings.TrimSpace(option.Name) == "" {
					continue
				}
				if registry.Get(option.Name) != nil {
					continue
				}
				registry.Register(NewCoolerControlExternalMonitor(option, client))
				added++
			}
			logInfoModule("monitor", "registered %d coolercontrol monitors", added)
		}
	}

	lhmURL := config.GetLibreHardwareMonitorURL()
	if lhmURL != "" {
		client := GetLibreHardwareMonitorClient(lhmURL)
		options, err := client.ListMonitorOptions()
		if err != nil {
			logWarnModule("monitor", "librehardwaremonitor monitor init failed (url=%s): %v", lhmURL, err)
		} else {
			added := 0
			for _, option := range options {
				if strings.TrimSpace(option.Name) == "" {
					continue
				}
				if registry.Get(option.Name) != nil {
					continue
				}
				registry.Register(NewLibreHardwareMonitorExternalMonitor(option, client))
				added++
			}
			logInfoModule("monitor", "registered %d librehardwaremonitor monitors", added)
		}
	}

	if config.IsRTSSCollectEnabled() {
		client := rtsssource.GetRTSSClient()
		options := client.ListMonitorOptions()
		added := 0
		for _, option := range options {
			if strings.TrimSpace(option.Name) == "" {
				continue
			}
			if registry.Get(option.Name) != nil {
				continue
			}
			registry.Register(NewRTSSExternalMonitor(option, client))
			added++
		}
		logInfoModule("monitor", "registered %d rtss monitors", added)
	}
}
