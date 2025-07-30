package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// CustomFormatter provides a clean, standard log format
type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// Color codes for different levels
	var levelColor string
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = "\033[36m" // Cyan
		levelText = " INFO"
	case logrus.WarnLevel:
		levelColor = "\033[33m" // Yellow
		levelText = " WARN"
	case logrus.ErrorLevel:
		levelColor = "\033[31m" // Red
		levelText = "ERROR"
	case logrus.DebugLevel:
		levelColor = "\033[37m" // White
		levelText = "DEBUG"
	default:
		levelColor = "\033[0m" // Reset
		levelText = strings.ToUpper(entry.Level.String())
	}

	reset := "\033[0m"

	// Get module name from fields or use default
	module := "main"
	if moduleField, exists := entry.Data["module"]; exists {
		if moduleStr, ok := moduleField.(string); ok {
			module = moduleStr
		}
	}

	// Format: [LEVEL timestamp] [module] message
	return []byte(fmt.Sprintf("[%s%s%s %s] [%12s] %s\n",
		levelColor, levelText, reset, timestamp, module, entry.Message)), nil
}

func initLogger() {
	logger = logrus.New()

	var output io.Writer = os.Stdout

	if runtime.GOOS == "windows" {
		if logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			output = io.MultiWriter(os.Stdout, logFile)
		}
	}

	logger.SetOutput(output)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&CustomFormatter{})
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
