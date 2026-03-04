//go:build !windows

package rtss

func ReadMetrics() (Metrics, bool) {
	return Metrics{}, false
}
