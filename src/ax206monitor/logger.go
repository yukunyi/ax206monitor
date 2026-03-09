package main

import (
	"ax206monitor/coolercontrol"
	"ax206monitor/output"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *logrus.Logger

// CustomFormatter provides a clean, standard log format
type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// Keep plain text output so log files are directly searchable/readable.
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelText = " INFO"
	case logrus.WarnLevel:
		levelText = " WARN"
	case logrus.ErrorLevel:
		levelText = "ERROR"
	case logrus.DebugLevel:
		levelText = "DEBUG"
	default:
		levelText = strings.ToUpper(entry.Level.String())
	}

	// Get module name from fields or use default
	module := "main"
	if moduleField, exists := entry.Data["module"]; exists {
		if moduleStr, ok := moduleField.(string); ok {
			module = moduleStr
		}
	}

	// Format: [LEVEL timestamp] [module] message
	return []byte(fmt.Sprintf("[%s %s] [%12s] %s\n", levelText, timestamp, module, entry.Message)), nil
}

func initLogger() {
	logger = logrus.New()

	logFilePath := resolveLogFilePath()
	rotatingFileWriter := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    100,
		MaxBackups: 2,
		LocalTime:  true,
		Compress:   false,
	}
	writers := []io.Writer{rotatingFileWriter}
	if runtime.GOOS != "windows" {
		writers = append([]io.Writer{os.Stdout}, writers...)
	}

	logger.SetOutput(io.MultiWriter(writers...))
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&CustomFormatter{})

	output.SetLoggerHooks(output.LoggerHooks{
		Info:        logInfo,
		Warn:        logWarn,
		Error:       logError,
		InfoModule:  logInfoModule,
		WarnModule:  logWarnModule,
		ErrorModule: logErrorModule,
		Debug:       logDebug,
	})
	coolercontrol.SetLoggerHooks(coolercontrol.LoggerHooks{
		WarnModule:  logWarnModule,
		DebugModule: logDebugModule,
	})
	logInfoModule("log", "log file: %s", logFilePath)
}

func resolveLogFilePath() string {
	executablePath, err := os.Executable()
	if err == nil {
		executablePath = strings.TrimSpace(executablePath)
		tempDir := filepath.Clean(os.TempDir())
		inGoBuildCache := tempDir != "" && strings.Contains(filepath.Clean(executablePath), filepath.Join(tempDir, "go-build"))
		if executablePath != "" && !inGoBuildCache {
			return filepath.Join(filepath.Dir(executablePath), "app.log")
		}
	}
	workingDir, err := os.Getwd()
	if err == nil {
		workingDir = strings.TrimSpace(workingDir)
		if workingDir != "" {
			return filepath.Join(workingDir, "app.log")
		}
	}
	return "app.log"
}

// Convenience functions with module support
func logInfo(msg string, args ...interface{}) {
	entry := logger.WithField("module", "main")
	if len(args) > 0 {
		entry.Infof(msg, args...)
	} else {
		entry.Info(msg)
	}
}

func logWarn(msg string, args ...interface{}) {
	entry := logger.WithField("module", "main")
	if len(args) > 0 {
		entry.Warnf(msg, args...)
	} else {
		entry.Warn(msg)
	}
}

func logError(msg string, args ...interface{}) {
	entry := logger.WithField("module", "main")
	if len(args) > 0 {
		entry.Errorf(msg, args...)
	} else {
		entry.Error(msg)
	}
}

func logDebug(msg string, args ...interface{}) {
	entry := logger.WithField("module", "main")
	if len(args) > 0 {
		entry.Debugf(msg, args...)
	} else {
		entry.Debug(msg)
	}
}

func logFatal(msg string, args ...interface{}) {
	entry := logger.WithField("module", "main")
	if len(args) > 0 {
		entry.Fatalf(msg, args...)
	} else {
		entry.Fatal(msg)
	}
}

// Module-specific logging functions
func logInfoModule(module, msg string, args ...interface{}) {
	entry := logger.WithField("module", module)
	if len(args) > 0 {
		entry.Infof(msg, args...)
	} else {
		entry.Info(msg)
	}
}

func logWarnModule(module, msg string, args ...interface{}) {
	entry := logger.WithField("module", module)
	if len(args) > 0 {
		entry.Warnf(msg, args...)
	} else {
		entry.Warn(msg)
	}
}

func logErrorModule(module, msg string, args ...interface{}) {
	entry := logger.WithField("module", module)
	if len(args) > 0 {
		entry.Errorf(msg, args...)
	} else {
		entry.Error(msg)
	}
}

func logDebugModule(module, msg string, args ...interface{}) {
	entry := logger.WithField("module", module)
	if len(args) > 0 {
		entry.Debugf(msg, args...)
	} else {
		entry.Debug(msg)
	}
}
