package lifecycle_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nextmicro/next/lifecycle"
)

func TestFuncComponent_NoCallbacksAllNil(t *testing.T) {
	c := lifecycle.NewFuncComponent("noop")
	if c.Name() != "noop" {
		t.Fatalf("Name = %q, want %q", c.Name(), "noop")
	}
	ctx := context.Background()
	if err := c.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := c.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := c.Watch(ctx); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if err := c.Health(ctx); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if c.Deps() != nil {
		t.Fatalf("Deps should be nil when none declared")
	}
}

func TestFuncComponent_CallbacksInvoked(t *testing.T) {
	var initN, startN, stopN, watchN, healthN int

	c := lifecycle.NewFuncComponent("biz").
		WithInit(func(context.Context) error { initN++; return nil }).
		WithStart(func(context.Context) error { startN++; return nil }).
		WithStop(func(context.Context) error { stopN++; return nil }).
		WithWatch(func(context.Context) error { watchN++; return nil }).
		WithHealth(func(context.Context) error { healthN++; return nil })

	lc := lifecycle.New()
	lc.Add(c)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("lc.Start: %v", err)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("lc.Stop: %v", err)
	}
	report := lc.Health(context.Background())
	if report.Status != lifecycle.HealthOK {
		t.Fatalf("Health = %s, want OK", report.Status)
	}
	if initN != 1 || startN != 1 || stopN != 1 || watchN != 1 || healthN != 1 {
		t.Fatalf("unexpected call counts: init=%d start=%d stop=%d watch=%d health=%d",
			initN, startN, stopN, watchN, healthN)
	}
}

func TestFuncComponent_StartErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	c := lifecycle.NewFuncComponent("fail").
		WithStart(func(context.Context) error { return boom })

	lc := lifecycle.New()
	lc.Add(c)
	err := lc.Start(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
}
