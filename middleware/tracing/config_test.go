package tracing

import (
	"testing"

	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/tracing/v1"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestOptionsFromConfig(t *testing.T) {
	cfg := &v1.Tracing{Disabled: true}
	anyCfg, err := anypb.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	opts, err := optionsFromConfig(&configv1.Middleware{Options: anyCfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 {
		t.Fatalf("want 1 option, got %d", len(opts))
	}
}

func TestDefaultPropagator(t *testing.T) {
	p := DefaultPropagator()
	if p == nil {
		t.Fatal("propagator is nil")
	}
	var _ propagation.TextMapPropagator = p
}
