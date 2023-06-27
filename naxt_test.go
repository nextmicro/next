package next_test

import (
	"context"
	"testing"
	"time"

	"github.com/nextmicro/next"
	"github.com/nextmicro/next/config"
)

func TestNewNext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 准备测试数据和参数
	opts := []next.Option{
		next.ID("test-ID"),
		next.Name("test-Name"),
		next.Version("test-Version"),
		next.Context(ctx),
		// 添加其他选项参数
	}

	cfg, err := config.Init("")
	if err != nil {
		t.Fatal(err)
	}

	defer cfg.Close()

	// 调用被测试的函数
	app, err := next.New(opts...)
	if err != nil {
		t.Fatal(err)
	}

	if err = app.Run(); err != nil {
		t.Fatal(err)
	}

	// 停止应用
	err = app.Stop()
	if err != nil {
		t.Fatal(err)
	}
}
