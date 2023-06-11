package kratos

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	vlog "github.com/go-volo/logger"
)

func TestMain(t *testing.M) {
	log.DefaultLogger = New(vlog.DefaultLogger)
	t.Run()
}

func TestDefaultLogger_Log(t *testing.T) {
	// 设置预期的日志级别和消息
	level := log.LevelInfo
	message := "Test log message"

	// 调用 Log 方法进行日志记录
	_log := log.With(log.DefaultLogger,
		"service.id", "100",
		"service.name", "atommicro",
		"service.version", "v1.0.0",
	)
	err := _log.Log(level, "msg", message)
	if err != nil {
		t.Errorf("Log method returned an error: %v", err)
	}
}

func TestHelper(_ *testing.T) {
	//logger := log.With(log.DefaultLogger, "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)
	_log := log.NewHelper(log.DefaultLogger)
	_log.Log(log.LevelDebug, "msg", "test debug")
	_log.Debug("test debug")
	_log.Debugf("test %s", "debug")
	_log.Debugw("log", "test debug")

	_log.Warn("test warn")
	_log.Warnf("test %s", "warn")
	_log.Warnw("log", "test warn")
}

func TestHelperWithMsgKey(_ *testing.T) {
	logger := log.With(log.DefaultLogger, "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)
	_log := log.NewHelper(logger, log.WithMessageKey("message"))
	_log.Debugf("test %s", "debug")
	_log.Debugw("log", "test debug")
}

func TestHelperLevel(_ *testing.T) {
	_log := log.NewHelper(log.DefaultLogger)
	_log.Debug("test debug")
	_log.Info("test info")
	_log.Infof("test %s", "info")
	_log.Warn("test warn")
	_log.Error("test error")
	_log.Errorf("test %s", "error")
	_log.Errorw("log", "test error")
}

func BenchmarkHelperPrint(b *testing.B) {
	log := log.NewHelper(log.NewStdLogger(io.Discard))
	for i := 0; i < b.N; i++ {
		log.Debug("test")
	}
}

func BenchmarkHelperPrintf(b *testing.B) {
	_log := log.NewHelper(log.NewStdLogger(io.Discard))
	for i := 0; i < b.N; i++ {
		_log.Debugf("%s", "test")
	}
}

func BenchmarkHelperPrintw(b *testing.B) {
	_log := log.NewHelper(log.NewStdLogger(io.Discard))
	for i := 0; i < b.N; i++ {
		_log.Debugw("key", "value")
	}
}

type traceKey struct{}

func TestContext(_ *testing.T) {
	logger := log.With(log.DefaultLogger,
		"trace", Trace(),
	)
	_log := log.NewHelper(logger)
	ctx := context.WithValue(context.Background(), traceKey{}, "2233")
	_log.WithContext(ctx).Info("got trace!")
}

func Trace() log.Valuer {
	return func(ctx context.Context) interface{} {
		s, ok := ctx.Value(traceKey{}).(string)
		if !ok {
			return nil
		}
		return s
	}
}
