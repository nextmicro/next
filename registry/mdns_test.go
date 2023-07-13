package registry

import (
	"context"
	"github.com/go-kratos/kratos/v2/registry"
	"os"
	"testing"
)

func TestMDNS(t *testing.T) {
	// skip test in travis because of sendto: operation not permitted error
	if travis := os.Getenv("TRAVIS"); travis == "true" {
		t.Skip()
	}

	testData := []*registry.ServiceInstance{
		{
			ID:      "test2-1",
			Name:    "test2",
			Version: "1.0.1",
			Metadata: map[string]string{
				"foo2": "bar2",
			},
			Endpoints: []string{
				"10.0.0.2:10002",
			},
		},
		{
			ID:      "test3-1",
			Name:    "test3",
			Version: "1.0.3",
			Metadata: map[string]string{
				"foo3": "bar3",
			},
			Endpoints: []string{
				"10.0.0.2:10003",
			},
		},
		{
			ID:   "test4-1",
			Name: "test4",
			Metadata: map[string]string{
				"foo4": "bar4",
			},
			Endpoints: []string{
				"[::]:10004",
			},
		},
	}

	// new registry
	r := NewRegistry()

	ctx := context.TODO()
	for _, service := range testData {
		// register service
		if err := r.Register(ctx, service); err != nil {
			t.Fatal(err)
		}

		// get registered service
		s, err := r.GetService(ctx, service.Name)
		if err != nil {
			t.Fatal(err)
		}

		if len(s) != 1 {
			t.Fatalf("Expected one result for %s got %d", service.Name, len(s))
		}

		if s[0].Name != service.Name {
			t.Fatalf("Expected name %s got %s", service.Name, s[0].Name)
		}

		if s[0].Version != service.Version {
			t.Fatalf("Expected version %s got %s", service.Version, s[0].Version)
		}
	}
}

func TestEncoding(t *testing.T) {
	testData := []*mdnsTxt{
		{
			Version: "1.0.0",
			Metadata: map[string]string{
				"foo": "bar",
			},
			Endpoints: []string{
				"10.0.0.2:10003",
				"10.0.0.2:10002",
				"10.0.0.2:10001",
			},
		},
	}

	for _, d := range testData {
		encoded, err := encode(d)
		if err != nil {
			t.Fatal(err)
		}

		for _, txt := range encoded {
			if len(txt) > 255 {
				t.Fatalf("One of parts for txt is %d characters", len(txt))
			}
		}

		decoded, err := decode(encoded)
		if err != nil {
			t.Fatal(err)
		}

		if decoded.Version != d.Version {
			t.Fatalf("Expected version %s got %s", d.Version, decoded.Version)
		}

		if len(decoded.Endpoints) != len(d.Endpoints) {
			t.Fatalf("Expected %d endpoints, got %d", len(d.Endpoints), len(decoded.Endpoints))
		}

		for k, v := range d.Metadata {
			if val := decoded.Metadata[k]; val != v {
				t.Fatalf("Expected %s=%s got %s=%s", k, v, k, val)
			}
		}
	}
}
