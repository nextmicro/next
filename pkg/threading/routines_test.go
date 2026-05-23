package threading

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/nextmicro/next/pkg/lang"
	"github.com/stretchr/testify/assert"
)

func TestRoutineId(t *testing.T) {
	assert.True(t, RoutineId() > 0)
}

// 注：此前的测试通过 `logger.DefaultLogger = logger.New(...)` 替换全局变量
// 来捕获 rescue.Recover 写入的日志输出，但这与 rescue 内部对 logger.DefaultLogger
// 的并发读取存在 race（go test -race 会报错）。
//
// 改造后只验证 RunSafe / RunSafeCtx / GoSafeCtx 能从 panic 中恢复且
// 通过 channel 完成同步，不再断言日志输出，从而避免对第三方 logger 全局变量
// 的并发写入。

func TestRunSafe(t *testing.T) {
	log.SetOutput(io.Discard)

	i := 0
	defer func() { assert.Equal(t, 1, i) }()

	ch := make(chan lang.PlaceholderType)
	go RunSafe(func() {
		defer func() { ch <- lang.Placeholder }()
		panic("panic")
	})
	<-ch
	i++
}

func TestRunSafeCtx(t *testing.T) {
	ctx := context.Background()
	ch := make(chan lang.PlaceholderType)

	i := 0
	defer func() { assert.Equal(t, 1, i) }()

	go RunSafeCtx(ctx, func() {
		defer func() { ch <- lang.Placeholder }()
		panic("panic")
	})
	<-ch
	i++
}

func TestGoSafeCtx(t *testing.T) {
	ctx := context.Background()
	ch := make(chan lang.PlaceholderType)

	i := 0
	defer func() { assert.Equal(t, 1, i) }()

	GoSafeCtx(ctx, func() {
		defer func() { ch <- lang.Placeholder }()
		panic("panic")
	})
	<-ch
	i++
}
