package container_test

import (
	"fmt"

	"github.com/nextmicro/next/container"
)

type Logger interface {
	Info(msg string)
}

type StdLogger struct{ prefix string }

func (l *StdLogger) Info(msg string) {
	fmt.Println(l.prefix, msg)
}

type UserService struct {
	log Logger
}

// ExampleRegister 展示按具体类型注册与解析的基本用法。
func Example() {
	c := container.New()

	_ = container.RegisterAs[Logger](c, &StdLogger{prefix: "[INFO]"})
	_ = container.Register(c, &UserService{log: container.MustResolve[Logger](c)})
	c.Seal()

	svc := container.MustResolve[*UserService](c)
	svc.log.Info("ready")

	// Output: [INFO] ready
}

// ExampleRegisterNamed 展示多实例命名注册。
func Example_named() {
	c := container.New()

	_ = container.RegisterNamed[Logger](c, "stdout", &StdLogger{prefix: "[OUT]"})
	_ = container.RegisterNamed[Logger](c, "stderr", &StdLogger{prefix: "[ERR]"})

	out, _ := container.ResolveNamed[Logger](c, "stdout")
	err, _ := container.ResolveNamed[Logger](c, "stderr")

	out.Info("hello")
	err.Info("world")

	// Output:
	// [OUT] hello
	// [ERR] world
}
