package main

import "metricsrendersender/coolercontrol"

type CoolerControlClient = coolercontrol.CoolerControlClient
type CoolerControlMonitorOption = coolercontrol.CoolerControlMonitorOption

func GetCoolerControlClient(baseURL, password string) *CoolerControlClient {
	return coolercontrol.GetCoolerControlClient(baseURL, password)
}

func coolerControlBuildShortLabel(deviceLabel string, metricText string) string {
	return coolercontrol.BuildShortLabel(deviceLabel, metricText)
}
