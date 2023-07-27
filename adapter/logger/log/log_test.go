package log_test

import (
	std "log"
	"testing"

	"github.com/go-volo/logger"
	"github.com/nextmicro/next/adapter/logger/log"
)

func TestMain(t *testing.M) {
	log.New(logger.DefaultLogger)
	t.Run()
}

func TestLogWriter_Write(t *testing.T) {
	std.Println("test")
	std.Printf("test %s", "msg")
}
