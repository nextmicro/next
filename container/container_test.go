package container_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/nextmicro/next/container"
)

type fakeService struct{ name string }

type fakeBroker interface {
	Send(msg string) error
}

type kafkaBroker struct{}

func (k *kafkaBroker) Send(msg string) error { return nil }

func TestRegisterAndResolve_ByConcreteType(t *testing.T) {
	c := container.New()
	svc := &fakeService{name: "svc"}

	if err := container.Register(c, svc); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, err := container.Resolve[*fakeService](c)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got != svc {
		t.Fatalf("Resolve returned wrong instance")
	}
}

func TestRegisterAs_ByInterface(t *testing.T) {
	c := container.New()
	b := &kafkaBroker{}

	if err := container.RegisterAs[fakeBroker](c, b); err != nil {
		t.Fatalf("RegisterAs failed: %v", err)
	}

	got, err := container.Resolve[fakeBroker](c)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got != fakeBroker(b) {
		t.Fatalf("Resolve returned wrong instance")
	}
}

func TestRegister_DuplicateReturnsError(t *testing.T) {
	c := container.New()
	if err := container.Register(c, &fakeService{name: "a"}); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	err := container.Register(c, &fakeService{name: "b"})
	if !errors.Is(err, container.ErrAlreadyRegistered) {
		t.Fatalf("expected ErrAlreadyRegistered, got %v", err)
	}
}

func TestResolve_UnregisteredReturnsError(t *testing.T) {
	c := container.New()
	_, err := container.Resolve[*fakeService](c)
	if !errors.Is(err, container.ErrNotRegistered) {
		t.Fatalf("expected ErrNotRegistered, got %v", err)
	}
}

func TestMustResolve_PanicsWhenMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on missing type")
		}
	}()
	c := container.New()
	_ = container.MustResolve[*fakeService](c)
}

func TestRegisterNamed_IsolationFromDefault(t *testing.T) {
	c := container.New()

	a := &fakeService{name: "primary"}
	b := &fakeService{name: "replica"}
	def := &fakeService{name: "default"}

	if err := container.Register(c, def); err != nil {
		t.Fatalf("Register default failed: %v", err)
	}
	if err := container.RegisterNamed[*fakeService](c, "primary", a); err != nil {
		t.Fatalf("RegisterNamed primary failed: %v", err)
	}
	if err := container.RegisterNamed[*fakeService](c, "replica", b); err != nil {
		t.Fatalf("RegisterNamed replica failed: %v", err)
	}

	gotDef, _ := container.Resolve[*fakeService](c)
	gotA, _ := container.ResolveNamed[*fakeService](c, "primary")
	gotB, _ := container.ResolveNamed[*fakeService](c, "replica")

	if gotDef != def {
		t.Fatalf("default mismatch")
	}
	if gotA != a {
		t.Fatalf("primary mismatch")
	}
	if gotB != b {
		t.Fatalf("replica mismatch")
	}
}

func TestRegisterNamed_DuplicateReturnsError(t *testing.T) {
	c := container.New()
	_ = container.RegisterNamed[*fakeService](c, "primary", &fakeService{})
	err := container.RegisterNamed[*fakeService](c, "primary", &fakeService{})
	if !errors.Is(err, container.ErrAlreadyRegistered) {
		t.Fatalf("expected ErrAlreadyRegistered, got %v", err)
	}
}

func TestRegisterNamed_EmptyNameRejected(t *testing.T) {
	c := container.New()
	err := container.RegisterNamed[*fakeService](c, "", &fakeService{})
	if err == nil {
		t.Fatalf("expected error for empty name")
	}
}

func TestSeal_BlocksFurtherRegistration(t *testing.T) {
	c := container.New()
	if c.Sealed() {
		t.Fatalf("new container should not be sealed")
	}
	_ = container.Register(c, &fakeService{name: "a"})
	c.Seal()
	if !c.Sealed() {
		t.Fatalf("Seal did not mark container sealed")
	}

	err := container.Register(c, &kafkaBroker{})
	if !errors.Is(err, container.ErrSealed) {
		t.Fatalf("expected ErrSealed, got %v", err)
	}

	err = container.RegisterNamed[*kafkaBroker](c, "x", &kafkaBroker{})
	if !errors.Is(err, container.ErrSealed) {
		t.Fatalf("expected ErrSealed for named, got %v", err)
	}

	// Resolve 仍然可用
	if _, err := container.Resolve[*fakeService](c); err != nil {
		t.Fatalf("Resolve after Seal should succeed, got %v", err)
	}
}

func TestSeal_Idempotent(t *testing.T) {
	c := container.New()
	c.Seal()
	c.Seal() // should not panic
	if !c.Sealed() {
		t.Fatalf("expected sealed")
	}
}

func TestValidate_AllSatisfied(t *testing.T) {
	c := container.New()
	_ = container.Register(c, &fakeService{})
	_ = container.RegisterAs[fakeBroker](c, &kafkaBroker{})

	deps := map[string][]reflect.Type{
		"componentA": {container.TypeOf[*fakeService](), container.TypeOf[fakeBroker]()},
	}
	if err := c.Validate(deps); err != nil {
		t.Fatalf("expected Validate to pass, got %v", err)
	}
}

func TestValidate_AggregatesMissing(t *testing.T) {
	c := container.New()
	_ = container.Register(c, &fakeService{})

	deps := map[string][]reflect.Type{
		"componentA": {container.TypeOf[fakeBroker]()},
		"componentB": {container.TypeOf[*kafkaBroker]()},
	}
	err := c.Validate(deps)
	if err == nil {
		t.Fatalf("expected Validate to report missing deps")
	}
	msg := err.Error()
	if !contains(msg, "componentA") || !contains(msg, "componentB") {
		t.Fatalf("expected both owners in error, got %s", msg)
	}
}

func TestHas_AndHasNamed(t *testing.T) {
	c := container.New()
	if container.Has[*fakeService](c) {
		t.Fatalf("Has should be false before register")
	}
	_ = container.Register(c, &fakeService{})
	if !container.Has[*fakeService](c) {
		t.Fatalf("Has should be true after register")
	}
	if container.HasNamed[*fakeService](c, "x") {
		t.Fatalf("HasNamed should be false")
	}
	_ = container.RegisterNamed[*fakeService](c, "x", &fakeService{})
	if !container.HasNamed[*fakeService](c, "x") {
		t.Fatalf("HasNamed should be true")
	}
}

func TestConcurrentResolve_NoRace(t *testing.T) {
	c := container.New()
	_ = container.Register(c, &fakeService{name: "svc"})
	c.Seal()

	const n = 1000
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := container.Resolve[*fakeService](c); err != nil {
				t.Errorf("concurrent Resolve failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
