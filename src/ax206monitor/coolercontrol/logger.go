package coolercontrol

import "fmt"

type LoggerHooks struct {
	WarnModule  func(module, format string, args ...interface{})
	DebugModule func(module, format string, args ...interface{})
}

var loggerHooks LoggerHooks

func SetLoggerHooks(hooks LoggerHooks) {
	loggerHooks = hooks
}

func logWarnModule(module, format string, args ...interface{}) {
	if loggerHooks.WarnModule != nil {
		loggerHooks.WarnModule(module, format, args...)
		return
	}
	_ = fmt.Sprintf(format, args...)
}

func logDebugModule(module, format string, args ...interface{}) {
	if loggerHooks.DebugModule != nil {
		loggerHooks.DebugModule(module, format, args...)
		return
	}
	_ = fmt.Sprintf(format, args...)
}
