package main

import "ax206monitor/coolercontrol"

type CoolerControlClient = coolercontrol.CoolerControlClient
type CoolerControlMonitorOption = coolercontrol.CoolerControlMonitorOption

func GetCoolerControlClient(baseURL, username, password string) *CoolerControlClient {
	return coolercontrol.GetCoolerControlClient(baseURL, username, password)
}

func coolerControlBuildShortLabel(deviceLabel string, metricText string) string {
	return coolercontrol.BuildShortLabel(deviceLabel, metricText)
}
