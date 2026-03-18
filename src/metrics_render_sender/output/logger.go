package output

import "fmt"

type LoggerHooks struct {
	Info        func(format string, args ...interface{})
	Warn        func(format string, args ...interface{})
	Error       func(format string, args ...interface{})
	InfoModule  func(module, format string, args ...interface{})
	WarnModule  func(module, format string, args ...interface{})
	ErrorModule func(module, format string, args ...interface{})
	Debug       func(format string, args ...interface{})
}

var loggerHooks LoggerHooks

func SetLoggerHooks(hooks LoggerHooks) {
	loggerHooks = hooks
}

func logInfoModule(module, format string, args ...interface{}) {
	if loggerHooks.InfoModule != nil {
		loggerHooks.InfoModule(module, format, args...)
		return
	}
	_ = fmt.Sprintf(format, args...)
}

func logWarn(format string, args ...interface{}) {
	if loggerHooks.Warn != nil {
		loggerHooks.Warn(format, args...)
		return
	}
	_ = fmt.Sprintf(format, args...)
}

func logWarnModule(module, format string, args ...interface{}) {
	if loggerHooks.WarnModule != nil {
		loggerHooks.WarnModule(module, format, args...)
		return
	}
	_ = fmt.Sprintf(format, args...)
}

func logErrorModule(module, format string, args ...interface{}) {
	if loggerHooks.ErrorModule != nil {
		loggerHooks.ErrorModule(module, format, args...)
		return
	}
	_ = fmt.Sprintf(format, args...)
}

func logDebug(format string, args ...interface{}) {
	if loggerHooks.Debug != nil {
		loggerHooks.Debug(format, args...)
		return
	}
	_ = fmt.Sprintf(format, args...)
}
