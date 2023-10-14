package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	log "github.com/nextmicro/logger"
)

type Nacos struct {
	log.Logger
}

func NewNacos(log log.Logger) *Nacos {
	return &Nacos{log}
}

func (l *Nacos) SetLogger() {
	logger.SetLogger(l)
}

func (l *Nacos) Info(args ...interface{}) {
	l.Logger.WithCallDepth(1).Info(args...)
}

func (l *Nacos) Warn(args ...interface{}) {
	l.Logger.WithCallDepth(1).Warn(args...)
}

func (l *Nacos) Error(args ...interface{}) {
	l.Logger.WithCallDepth(1).Error(args...)
}

func (l *Nacos) Debug(args ...interface{}) {
	l.Logger.WithCallDepth(1).Debug(args...)
}

func (l *Nacos) Infof(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Infof(fmt, args...)
}

func (l *Nacos) Warnf(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Warnf(fmt, args...)
}

func (l *Nacos) Errorf(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Errorf(fmt, args...)
}

func (l *Nacos) Debugf(fmt string, args ...interface{}) {
	l.Logger.WithCallDepth(1).Debugf(fmt, args...)
}
