package recovery

import (
	"testing"

	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/recovery"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestOptionsFromConfig(t *testing.T) {
	cfg := &v1.Recovery{StackSize: 4096, DisableStackAll: true}
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
