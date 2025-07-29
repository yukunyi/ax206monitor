package main

import (
	"sync"
)

type FanInfo struct {
	Name  string
	Speed int
}

var (
	cachedGPUModel string
	cacheInitMutex sync.Once
)

func initializeCache() {
	cacheInitMutex.Do(func() {
		cachedGPUModel = detectGPUModel()
	})
}

func detectGPUModel() string {
	return "Generic GPU"
}
