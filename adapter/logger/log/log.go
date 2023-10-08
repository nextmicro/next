package log

import (
	"bytes"
	std "log"

	log "github.com/nextmicro/logger"
)

type logWriter struct {
	logFunc func() func(msg string, fields ...interface{})
}

func New(logger log.Logger) *std.Logger {
	stdLogger := std.New(logWriter{
		logFunc: func() func(msg string, args ...interface{}) {
			return logger.WithCallDepth(3).Infof
		},
	}, "", 0)
	return stdLogger
}

func (l logWriter) Write(p []byte) (int, error) {
	p = bytes.TrimSpace(p)
	if l.logFunc != nil {
		l.logFunc()(string(p))
	}
	return len(p), nil
}
