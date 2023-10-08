package kratos

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nextmicro/logger"
)

const defaultCallerSkipCount = 2

// defaultLogger is an implementation of log.Logger interface.
type defaultLogger struct {
	base logger.Logger
}

// New returns a new defaultLogger instance.
func New(base logger.Logger) *defaultLogger {
	l := &defaultLogger{
		base: base,
	}

	return l
}

func (l *defaultLogger) SetLogger() {
	log.DefaultLogger = l
	log.SetLogger(l)
}

func (l *defaultLogger) Log(level log.Level, keyvals ...interface{}) error {
	base := l.base.WithCallDepth(defaultCallerSkipCount)

	if len(keyvals) == 0 {
		return nil
	}
	if (len(keyvals) & 1) == 1 {
		keyvals = append(keyvals, "KEYVALS UNPAIRED")
	}

	msg := ""
	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if keyvals[i] == log.DefaultMessageKey {
			msg = fmt.Sprint(keyvals[i+1])
		} else {
			fields[fmt.Sprint(keyvals[i])] = keyvals[i+1]
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
