//go:build linux || darwin

package proc

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nextmicro/logger"
)

const profileDuration = time.Minute

var (
	once sync.Once
	done = make(chan struct{})
)

// SignalProcessor 定义了处理特定信号的逻辑的接口
type SignalProcessor interface {
	ProcessSignal(chan os.Signal, os.Signal)
}

// signalHandler 封装了信号处理逻辑
type signalHandler struct {
	processors map[os.Signal]SignalProcessor
}

func newSignalHandler() *signalHandler {
	return &signalHandler{
		processors: make(map[os.Signal]SignalProcessor),
	}
}

// RegisterProcessor 注册一个信号处理器
func (h *signalHandler) RegisterProcessor(sig os.Signal, processor SignalProcessor) {
	h.processors[sig] = processor
}

// Start 开始监听并处理信号
func (h *signalHandler) Start() {
	signals := make(chan os.Signal, len(h.processors))
	signal.Notify(signals, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for sig := range signals {
			if processor, ok := h.processors[sig]; ok {
				processor.ProcessSignal(signals, sig)
			} else {
				logger.Error("Received unregistered signal:", sig)
			}
		}
	}()
}

// InitSignalHandling 初始化信号处理
func InitSignalHandling() {
	once.Do(func() {
		handler := newSignalHandler()

		// 注册默认处理器，可以根据需要添加更多处理器
		handler.RegisterProcessor(syscall.SIGUSR1, &DefaultProcessor{})
		handler.RegisterProcessor(syscall.SIGUSR2, &DefaultProcessor{})
		handler.RegisterProcessor(syscall.SIGTERM, &DefaultProcessor{})
		handler.RegisterProcessor(syscall.SIGINT, &DefaultProcessor{})

		handler.Start()
	})
}

// Done 返回一个channel，用于通知进程退出
func Done() <-chan struct{} {
	return done
}

func stopOnSignal() {
	select {
	case <-done:
		// channel已关闭
	default:
		close(done)
	}
}

// DefaultProcessor 实现了SignalProcessor接口，提供默认信号处理逻辑
type DefaultProcessor struct{}

func (dp *DefaultProcessor) ProcessSignal(signals chan os.Signal, sig os.Signal) {
	switch sig {
	case syscall.SIGUSR1:
		DumpGoroutines(fileCreator{})
	case syscall.SIGUSR2:
		profiler := StartProfile()
		time.AfterFunc(profileDuration, profiler.Stop)
	case syscall.SIGTERM:
		stopOnSignal()
		GracefulStop(signals, syscall.SIGTERM)
	case syscall.SIGINT:
		stopOnSignal()
		GracefulStop(signals, syscall.SIGINT)
	default:
		logger.Error("Got unregistered signal:", sig)
	}
}
