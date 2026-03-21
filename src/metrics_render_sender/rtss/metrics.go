package rtss

import "math"

type Metrics struct {
	// Connected indicates whether we were able to read RTSS shared memory.
	Connected bool

	ForegroundPID uint32
	ForegroundFPS float64
	MaxFPS        float64
	ActiveApps    int

	// ForegroundFrameTimeInstantMS is the instantaneous frame time derived from RTSS
	// dwFrameTime (microseconds per frame), converted to milliseconds.
	ForegroundFrameTimeInstantMS float64

	// ForegroundFPSAvg is computed from the foreground frame time buffer.
	ForegroundFPSAvg float64

	// ForegroundFPS1PLow and ForegroundFPS01PLow are computed from percentiles of the
	// foreground frame time buffer (1% low: p99 frametime; 0.1% low: p99.9 frametime).
	ForegroundFPS1PLow  float64
	ForegroundFPS01PLow float64
	ForegroundFTMinMS   float64
	ForegroundFTAvgMS   float64
	ForegroundFTMaxMS   float64
	ForegroundFTP99MS   float64
	ForegroundFTP999MS  float64
}

func (m Metrics) ForegroundFrameTimeMS() float64 {
	// Prefer direct frame time from RTSS if available.
	if m.ForegroundFrameTimeInstantMS > 0 {
		return m.ForegroundFrameTimeInstantMS
	}
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
	return sanitizeFPSFloat(fps)
}

func sanitizeFPSFloat(fps float64) float64 {
	if math.IsNaN(fps) || math.IsInf(fps, 0) || fps < 0 {
		return 0
	}
	return fps
}
