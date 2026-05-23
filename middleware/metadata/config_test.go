package metadata

import (
	"testing"

	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/metadata/v1"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestOptionsFromConfigServer(t *testing.T) {
	cfg := &v1.Metadata{Prefix: []string{"x-custom-"}}
	anyCfg, err := anypb.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	opts, err := optionsFromConfigServer(&configv1.Middleware{Options: anyCfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 {
		t.Fatalf("want 1 option, got %d", len(opts))
	}
}

func TestOptionsFromConfigServer_DefaultPrefix(t *testing.T) {
	opts, err := optionsFromConfigServer(&configv1.Middleware{})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 {
		t.Fatalf("want default prefix option, got %d", len(opts))
	}
}
