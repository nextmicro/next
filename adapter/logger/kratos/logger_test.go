package kratos

import (
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	vlog "github.com/nextmicro/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.M) {
	New(vlog.DefaultLogger).SetLogger()
	t.Run()
}

func TestDefaultLogger_Log(t *testing.T) {
	// 设置预期的日志级别和消息
	level := log.LevelInfo

	// 调用 Log 方法进行日志记录
	_log := log.With(log.DefaultLogger,
		"service.id", "100",
		"service.name", "atommicro",
		"service.version", "v1.0.0",
	)

	err := _log.Log(level, "msg", "value1")
	assert.NoError(t, err)
	err = _log.Log(level, log.DefaultMessageKey, "test log")
	assert.NoError(t, err)
	err = _log.Log(level, "tttt", "key1", "ccc", "vvv")
	assert.NoError(t, err)
}
