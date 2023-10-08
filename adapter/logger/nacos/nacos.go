package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	log "github.com/nextmicro/logger"
)

type nacos struct {
	log.Logger
}

func NewNacos(log log.Logger) *nacos {
	return &nacos{log}
}

func (l *nacos) SetLogger() {
	logger.SetLogger(l)
}

func (l *nacos) Info(args ...interface{}) {
	l.Logger.WithCallDepth(1).Info(args...)
}

func (l *nacos) Warn(args ...interface{}) {
	l.Logger.WithCallDepth(1).Warn(args...)
}

func (l *nacos) Error(args ...interface{}) {
	l.Logger.WithCallDepth(1).Error(args...)
}

func (l *nacos) Debug(args ...interface{}) {
	l.Logger.WithCallDepth(1).Debug(args...)
}

func (l *nacos) Infof(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Infof(fmt, args...)
}

func (l *nacos) Warnf(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Warnf(fmt, args...)
}

func (l *nacos) Errorf(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Errorf(fmt, args...)
}

func (l *nacos) Debugf(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Debugf(fmt, args...)
}
