package config_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/nextmicro/next/config"
)

// fakeValue 实现 kconfig.Value 的最小子集，用于单元测试。
type fakeValue struct {
	raw any
}

func (f *fakeValue) Bool() (bool, error)             { return false, nil }
func (f *fakeValue) Int() (int64, error)             { return 0, nil }
func (f *fakeValue) Float() (float64, error)         { return 0, nil }
func (f *fakeValue) String() (string, error)         { return "", nil }
func (f *fakeValue) Duration() (time.Duration, error) { return 0, nil }
func (f *fakeValue) Slice() ([]kconfig.Value, error) { return nil, nil }
func (f *fakeValue) Map() (map[string]kconfig.Value, error) {
	return nil, nil
}
func (f *fakeValue) Load() any { return f.raw }
func (f *fakeValue) Store(v any) {
	f.raw = v
}
func (f *fakeValue) Scan(out any) error {
	if scanErr, ok := f.raw.(error); ok {
		return scanErr
	}
	switch v := out.(type) {
	case *testCfg:
		if c, ok := f.raw.(testCfg); ok {
			*v = c
			return nil
		}
	}
	return errors.New("fakeValue.Scan: unsupported type")
}

// fakeConfig 实现 kconfig.Config 的最小子集。
type fakeConfig struct {
	mu       sync.Mutex
	values   map[string]*fakeValue
	watchers map[string][]kconfig.Observer
}

func newFakeConfig() *fakeConfig {
	return &fakeConfig{
		values:   make(map[string]*fakeValue),
		watchers: make(map[string][]kconfig.Observer),
	}
}

func (f *fakeConfig) set(path string, v any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[path] = &fakeValue{raw: v}
}

func (f *fakeConfig) trigger(path string, v any) {
	f.mu.Lock()
	val := &fakeValue{raw: v}
	f.values[path] = val
	obs := append([]kconfig.Observer(nil), f.watchers[path]...)
	f.mu.Unlock()
	for _, o := range obs {
		o(path, val)
	}
}

func (f *fakeConfig) Load() error  { return nil }
func (f *fakeConfig) Close() error { return nil }
func (f *fakeConfig) Scan(any) error {
	return errors.New("not used")
}
func (f *fakeConfig) Value(key string) kconfig.Value {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.values[key]; ok {
		return v
	}
	return nil
}
func (f *fakeConfig) Watch(key string, ob kconfig.Observer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.watchers[key] = append(f.watchers[key], ob)
	return nil
}

type testCfg struct {
	Datasource string
	Pool       int
}

func TestBind_Success(t *testing.T) {
	c := newFakeConfig()
	c.set("database", testCfg{Datasource: "user:pwd@/db", Pool: 8})

	got, err := config.Bind[testCfg](c, "database")
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if got.Datasource != "user:pwd@/db" || got.Pool != 8 {
		t.Fatalf("Bind returned wrong values: %+v", got)
	}
}

func TestBind_PathNotFound(t *testing.T) {
	c := newFakeConfig()
	_, err := config.Bind[testCfg](c, "missing")
	if !errors.Is(err, config.ErrPathNotFound) {
		t.Fatalf("expected ErrPathNotFound, got %v", err)
	}
}

func TestBind_ScanError(t *testing.T) {
	c := newFakeConfig()
	c.set("bad", errors.New("oops"))
	_, err := config.Bind[testCfg](c, "bad")
	if err == nil || errors.Is(err, config.ErrPathNotFound) {
		t.Fatalf("expected scan error, got %v", err)
	}
}

func TestBind_NilConfigReturnsError(t *testing.T) {
	_, err := config.Bind[testCfg](nil, "x")
	if err == nil {
		t.Fatal("expected error for nil Config")
	}
}

func TestMustBind_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	c := newFakeConfig()
	_ = config.MustBind[testCfg](c, "missing")
}

func TestBindWatch_TriggersCallback(t *testing.T) {
	c := newFakeConfig()
	c.set("database", testCfg{Datasource: "init", Pool: 1})

	var called int32
	var seen atomic.Pointer[testCfg]
	stop, err := config.BindWatch[testCfg](c, "database", func(cfg *testCfg) {
		atomic.AddInt32(&called, 1)
		seen.Store(cfg)
	})
	if err != nil {
		t.Fatalf("BindWatch: %v", err)
	}
	defer stop()

	c.trigger("database", testCfg{Datasource: "new-dsn", Pool: 9})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&called) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected callback once, got %d", called)
	}
	got := seen.Load()
	if got == nil || got.Datasource != "new-dsn" || got.Pool != 9 {
		t.Fatalf("callback received wrong value: %+v", got)
	}
}

func TestBindWatch_NilArgsReturnError(t *testing.T) {
	if _, err := config.BindWatch[testCfg](nil, "x", func(*testCfg) {}); err == nil {
		t.Fatal("expected error for nil Config")
	}
	if _, err := config.BindWatch[testCfg](newFakeConfig(), "x", nil); err == nil {
		t.Fatal("expected error for nil fn")
	}
}
