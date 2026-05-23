package lifecycle_test

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nextmicro/next/container"
	"github.com/nextmicro/next/lifecycle"
)

// LoggerService 是一个基础设施 Component，只实现 Starter 接口。
type LoggerService struct{}

func (l *LoggerService) Name() string                     { return "logger" }
func (l *LoggerService) Start(_ context.Context) error    { fmt.Println("logger started"); return nil }
func (l *LoggerService) Stop(_ context.Context) error     { fmt.Println("logger stopped"); return nil }

// BrokerService 依赖 LoggerService。
type BrokerService struct{}

func (b *BrokerService) Name() string                  { return "broker" }
func (b *BrokerService) Start(_ context.Context) error { fmt.Println("broker started"); return nil }
func (b *BrokerService) Stop(_ context.Context) error  { fmt.Println("broker stopped"); return nil }
func (b *BrokerService) Deps() []reflect.Type {
	return []reflect.Type{container.TypeOf[*LoggerService]()}
}

// Example 展示 Lifecycle 的基本编排能力：
// LoggerService 先启动，BrokerService 后启动；Stop 时反向。
func Example() {
	lc := lifecycle.New()
	lc.Add(&LoggerService{}).Add(&BrokerService{})

	ctx := context.Background()
	if err := lc.Start(ctx); err != nil {
		fmt.Println("start error:", err)
		return
	}
	if err := lc.Stop(ctx); err != nil {
		fmt.Println("stop error:", err)
		return
	}

	// Output:
	// logger started
	// broker started
	// broker stopped
	// logger stopped
}
