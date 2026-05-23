package httpx_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/nextmicro/next/internal/host"
	"github.com/nextmicro/next/transport"
	"github.com/nextmicro/next/transport/httpx"
)

const appJSONStr = "application/json"

type User struct {
	Name string `json:"name"`
}

func corsFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", r.Method)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authFilter(next httpx.HandlerFunc) httpx.HandlerFunc {
	return func(c httpx.Context) error {
		return next(c)
	}
}

func PreMiddleware(next httpx.HandlerFunc) httpx.HandlerFunc {
	return func(c httpx.Context) error {
		fmt.Println("PreMiddleware start")
		err := next(c)
		fmt.Println("PreMiddleware end")
		return err
	}
}

func CustomMiddleware(next httpx.HandlerFunc) httpx.HandlerFunc {
	return func(c httpx.Context) error {
		fmt.Println("CustomMiddleware start")
		err := next(c)
		fmt.Println("CustomMiddleware end")
		return err
	}
}

func TestRoute(t *testing.T) {
	ctx := context.Background()
	srv, err := transport.NewHTTPServer(transport.HTTPServerConfig{
		ServerOptions: append(httpx.ServerOptionsFromRouter(),
			khttp.Filter(corsFilter),
		),
	})
	if err != nil {
		t.Fatal(err)
	}

	route := httpx.NewRouter(srv, "/v1", PreMiddleware)
	route.GET("/users/{name}", func(c httpx.Context) error {
		u := &User{Name: c.Vars().Get("name")}
		return c.JSON(http.StatusOK, u)
	}, authFilter, CustomMiddleware)

	route.POST("/users", func(c httpx.Context) error {
		u := new(User)
		if err := c.Bind(u); err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, u)
	})

	route.PUT("/users", func(c httpx.Context) error {
		u := new(User)
		if err := c.Bind(u); err != nil {
			return err
		}
		h := c.Middleware(func(ctx context.Context, in interface{}) (interface{}, error) {
			return u, nil
		})
		res, err := h(c, u)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, res)
	})

	if e, err := srv.Endpoint(); err != nil || e == nil {
		t.Fatal(e, err)
	}

	go func() {
		if err := srv.Start(ctx); err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Second)
	testRouteHTTP(t, srv)
	_ = srv.Stop(ctx)
}

func testRouteHTTP(t *testing.T, srv *khttp.Server) {
	t.Helper()
	ep, err := srv.Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := host.ExtractHostPort(ep.Host)
	if err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("http://127.0.0.1:%d/v1", port)

	resp, err := http.Get(base + "/users/foo")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("code: %d", resp.StatusCode)
	}
	u := new(User)
	if err = json.NewDecoder(resp.Body).Decode(u); err != nil {
		t.Fatal(err)
	}
	if u.Name != "foo" {
		t.Fatalf("got %s want foo", u.Name)
	}

	resp, err = http.Post(base+"/users", appJSONStr, strings.NewReader(`{"name":"bar"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("code: %d", resp.StatusCode)
	}
	u = new(User)
	if err = json.NewDecoder(resp.Body).Decode(u); err != nil {
		t.Fatal(err)
	}
	if u.Name != "bar" {
		t.Fatalf("got %s want bar", u.Name)
	}

	req, _ := http.NewRequest(http.MethodPut, base+"/users", strings.NewReader(`{"name":"bar"}`))
	req.Header.Set("Content-Type", appJSONStr)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("code: %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodOptions, base+"/users", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("code: %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Methods") != http.MethodOptions {
		t.Fatal("cors failed")
	}
}

func TestContextPoolReuse(t *testing.T) {
	srv := khttp.NewServer(httpx.ServerOptionsFromRouter()...)
	r := httpx.NewRouter(srv, "/")
	for i := 0; i < 32; i++ {
		n := i
		r.GET(fmt.Sprintf("/p/%d", n), func(c httpx.Context) error {
			return c.String(http.StatusOK, "ok")
		})
	}
}

func TestHandleMethods(t *testing.T) {
	srv := khttp.NewServer()
	r := httpx.NewRouter(srv, "/")
	h := func(c httpx.Context) error { return nil }
	r.GET("/get", h)
	r.HEAD("/head", h)
	r.PATCH("/patch", h)
	r.DELETE("/delete", h)
	r.CONNECT("/connect", h)
	r.OPTIONS("/options", h)
	r.TRACE("/trace", h)
}
