package main

func getConfiguredLibreHardwareMonitorClient() *LibreHardwareMonitorClient {
	config := GetGlobalMonitorConfig()
	if config == nil {
		return nil
	}
	url := config.GetLibreHardwareMonitorURL()
	if url == "" {
		return nil
	}
	return GetLibreHardwareMonitorClient(url)
}

func getLibreHardwareMonitorData() (*LibreHardwareMonitorData, bool) {
	client := getConfiguredLibreHardwareMonitorClient()
	if client == nil {
		return nil, false
	}
	if err := client.FetchData(); err != nil {
		return nil, false
	}
	data := client.GetData()
	if data == nil {
		return nil, false
	}
	return data, true
}
