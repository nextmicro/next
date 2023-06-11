package kratos

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-volo/logger"
)

const defaultCallerSkipCount = 2

// defaultLogger is an implementation of log.Logger interface.
type defaultLogger struct {
	base logger.Logger
}

// New returns a new defaultLogger instance.
func New(base logger.Logger) log.Logger {
	l := &defaultLogger{
		base: base,
	}

	return l
}

func (l *defaultLogger) Log(level log.Level, keyvals ...interface{}) error {
	base := l.base.WithCallDepth(defaultCallerSkipCount)

	msg := ""
	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if keyvals[i] == "msg" {
			msg = keyvals[i+1].(string)
		} else {
			fields[keyvals[i].(string)] = keyvals[i+1]
		}
	}
	if len(fields) > 0 {
		base = base.WithFields(fields)
	}

	switch level {
	case log.LevelDebug:
		base.Debug(msg)
	case log.LevelInfo:
		base.Info(msg)
	case log.LevelWarn:
		base.Warn(msg)
	case log.LevelError:
		base.Error(msg)
	case log.LevelFatal:
		base.Fatal(msg)
	default:
		base.Info(msg)
	}

	return nil
}
