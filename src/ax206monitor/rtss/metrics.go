package rtss

import "math"

type Metrics struct {
	ForegroundPID uint32
	ForegroundFPS float64
	MaxFPS        float64
	ActiveApps    int
}

func (m Metrics) ForegroundFrameTimeMS() float64 {
	if m.ForegroundFPS <= 0 {
		return 0
	}
	return 1000.0 / m.ForegroundFPS
}

func sanitizeFPS(raw uint32) float64 {
	if raw == 0 {
		return 0
	}
	fps := float64(raw) / 10.0
	if math.IsNaN(fps) || math.IsInf(fps, 0) || fps < 0 {
		return 0
	}
	return fps
}
