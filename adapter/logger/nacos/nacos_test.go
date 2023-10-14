package nacos_test

import (
	"testing"

	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/adapter/logger/nacos"
)

var (
	_log *nacos.Nacos
)

func TestMain(t *testing.M) {
	logger.DefaultLogger = logger.New(logger.WithLevel(logger.DebugLevel))
	_log = nacos.NewNacos(logger.DefaultLogger)

	t.Run()
}

func TestNewNacos(t *testing.T) {
	_log = nacos.NewNacos(logger.DefaultLogger)
}

func TestNacos_Debug(t *testing.T) {
	_log.Debug("test")
}

func TestNacos_Debugf(t *testing.T) {
	_log.Debugf("test %s", "test")
}

func TestNacos_Error(t *testing.T) {
	_log.Error("test")
}

func TestNacos_Errorf(t *testing.T) {
	_log.Errorf("test %s", "test")
}

func TestNacos_Info(t *testing.T) {
	_log.Info("test")
}

func TestNacos_Infof(t *testing.T) {
	_log.Infof("test %s", "test")
}
