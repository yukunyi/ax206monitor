package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type WebServerProcess struct {
	mu       sync.Mutex
	port     int
	devMode  bool
	viteURL  string
	cmd      *exec.Cmd
	done     chan error
	stopping bool
}

func NewWebServerProcess(port int, devMode bool, viteURL string) *WebServerProcess {
	return &WebServerProcess{
		port:    port,
		devMode: devMode,
		viteURL: viteURL,
	}
}

func (p *WebServerProcess) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", p.port)
}

func (p *WebServerProcess) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil
}

func (p *WebServerProcess) Start() error {
	p.mu.Lock()
	if p.cmd != nil {
		p.mu.Unlock()
		return nil
	}

	execPath, err := os.Executable()
	if err != nil {
		p.mu.Unlock()
		return fmt.Errorf("resolve executable failed: %w", err)
	}

	args := []string{"--port", strconv.Itoa(p.port)}

	cmd := exec.Command(execPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := append([]string{}, os.Environ()...)
	env = append(env, "AX206_MONITOR_WEB=1")
	if p.devMode {
		devURL := strings.TrimSpace(p.viteURL)
		if devURL == "" {
			devURL = "http://127.0.0.1:18087"
		}
		env = append(env, "AX206_MONITOR_DEV_URL="+devURL)
	}
	cmd.Env = env

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("start web server process failed: %w", err)
	}
	p.cmd = cmd
	p.done = done
	p.stopping = false
	pid := cmd.Process.Pid
	p.mu.Unlock()

	go p.watchProcess(cmd, done)
	logInfoModule("tray", "web server process started pid=%d addr=%s", pid, p.URL())
	return nil
}

func (p *WebServerProcess) watchProcess(cmd *exec.Cmd, done chan error) {
	err := cmd.Wait()
	done <- err
	close(done)

	p.mu.Lock()
	stopping := p.stopping
	if p.cmd == cmd {
		p.cmd = nil
		p.done = nil
		p.stopping = false
	}
	p.mu.Unlock()

	if err != nil {
		if stopping {
			logInfoModule("tray", "web server process stopped")
			return
		}
		logWarnModule("tray", "web server process exited with error: %v", err)
		return
	}
	logInfoModule("tray", "web server process exited")
}

func (p *WebServerProcess) Stop() error {
	p.mu.Lock()
	cmd := p.cmd
	done := p.done
	if cmd != nil && done != nil {
		p.stopping = true
	}
	p.mu.Unlock()

	if cmd == nil || done == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill web server process failed: %w", err)
		}
	} else {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			logWarnModule("tray", "interrupt web server process failed: %v", err)
			if err := cmd.Process.Kill(); err != nil {
				return fmt.Errorf("kill web server process failed: %w", err)
			}
		}
	}

	select {
	case err, ok := <-done:
		if !ok {
			return nil
		}
		if err != nil && !isExpectedStopError(err) {
			return fmt.Errorf("web server process stop failed: %w", err)
		}
		return nil
	case <-time.After(3 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill timed-out web server process failed: %w", err)
		}
		_, _ = <-done
		return nil
	}
}

func isExpectedStopError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrProcessDone) {
		return true
	}
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}
