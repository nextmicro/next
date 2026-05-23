// Quickstart 示例：演示如何用 next v2 启动一个最小的 HTTP 服务。
//
// 运行：
//
//	cd examples/quickstart
//	go run .
//
// 然后访问 http://127.0.0.1:8080/hello
package main

import (
	"context"
	"log"
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/nextmicro/next"
	"github.com/nextmicro/next/container"
	"github.com/nextmicro/next/lifecycle"
	"github.com/nextmicro/next/transport"
	"github.com/nextmicro/next/transport/httpx"

	// 显式启用全部内置 middleware（按需可改为精细化导入，
	// 见 next/middleware/all 包注释）。
	_ "github.com/nextmicro/next/middleware/all"
)

// GreeterService 业务服务示例。
type GreeterService struct {
	greeting string
}

// Hello 返回问候语。
func (s *GreeterService) Hello(name string) string {
	return s.greeting + ", " + name + "!"
}

// healthComponent 演示如何把一个简单后台组件挂到 lifecycle。
type healthComponent struct{ stopped chan struct{} }

func (*healthComponent) Name() string { return "health-printer" }

func (h *healthComponent) Start(ctx context.Context) error {
	h.stopped = make(chan struct{})
	go func() {
		defer close(h.stopped)
		<-ctx.Done()
	}()
	log.Println("[health-printer] started")
	return nil
}

func (h *healthComponent) Stop(_ context.Context) error {
	log.Println("[health-printer] stopping")
	return nil
}

func (*healthComponent) Health(_ context.Context) error { return nil }

func main() {
	// 1. 构造 HTTP server（kratos 原版 *khttp.Server）
	srv, err := transport.NewHTTPServer(transport.HTTPServerConfig{
		ServerOptions: append(httpx.ServerOptionsFromRouter(),
			khttp.Address(":8080"),
		),
	})
	if err != nil {
		log.Fatalf("transport.NewHTTPServer: %v", err)
	}

	// 2. 注册路由（httpx Router DSL）
	r := httpx.NewRouter(srv, "/")
	r.GET("/hello", func(c httpx.Context) error {
		name := c.Query().Get("name")
		if name == "" {
			name = "world"
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "hello, " + name,
		})
	})

	// 也可直接使用 kratos 原生 HandleFunc：
	// srv.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { ... })

	// 3. 构造 App：注入业务服务 + 挂载自定义 Component
	app, err := next.New(
		next.Name("quickstart"),
		next.Version("v1.0.0"),
		next.Server(srv),
		next.WithComponents(&healthComponent{}),
		next.WithContainer(func(c *container.Container) {
			_ = container.Register(c, &GreeterService{greeting: "Hello from next v2"})
		}),
	)
	if err != nil {
		log.Fatalf("next.New: %v", err)
	}

	// 4. 业务可在任意位置通过 Container 解析依赖
	greeter, err := container.Resolve[*GreeterService](app.Container())
	if err != nil {
		log.Fatalf("Resolve GreeterService: %v", err)
	}
	log.Printf("greeting prepared: %s", greeter.Hello("next"))

	// 5. 也可通过 Lifecycle 查询健康状态
	report := app.Lifecycle().Health(context.Background())
	log.Printf("lifecycle status: %s, components: %d", report.Status, len(report.Details))
	_ = lifecycle.HealthOK // 引用包以确保编译

	// 6. 启动应用，阻塞直到收到信号（SIGINT/SIGTERM）
	if err := app.Run(); err != nil {
		log.Fatalf("app.Run: %v", err)
	}
}
