package main

import (
	"sync"
	"time"
)

type LockScreenMonitor interface {
	Close()
}

type lockScreenPollingMonitor struct {
	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func (m *lockScreenPollingMonitor) Close() {
	if m == nil {
		return
	}
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
	<-m.doneCh
}

func StartLockScreenMonitor(onChange func(bool)) (LockScreenMonitor, error) {
	if onChange == nil {
		return nil, nil
	}
	return startPlatformLockScreenMonitor(onChange)
}

func startLockPollingMonitor(
	interval time.Duration,
	detect func() (bool, bool),
	onChange func(bool),
) LockScreenMonitor {
	if interval <= 0 {
		interval = time.Second
	}
	monitor := &lockScreenPollingMonitor{
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go func() {
		defer close(monitor.doneCh)
		lastState := false
		hasState := false
		if detect != nil {
			if state, ok := detect(); ok {
				lastState = state
				hasState = true
				onChange(state)
			}
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-monitor.stopCh:
				return
			case <-ticker.C:
				if detect == nil {
					continue
				}
				state, ok := detect()
				if !ok {
					continue
				}
				if !hasState || state != lastState {
					lastState = state
					hasState = true
					onChange(state)
				}
			}
		}
	}()
	return monitor
}
