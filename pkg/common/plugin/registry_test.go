package plugin

import "testing"

type mockPlugin struct{ name string }

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry[string, *mockPlugin]()
	r.Register("test-key", &mockPlugin{name: "test"})
	got, ok := r.Get("test-key")
	if !ok {
		t.Fatal("expected to find registered plugin")
	}
	if got.name != "test" {
		t.Errorf("got name %q, want %q", got.name, "test")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	r := NewRegistry[string, *mockPlugin]()
	_, ok := r.Get("missing")
	if ok {
		t.Fatal("expected not to find unregistered plugin")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry[string, *mockPlugin]()
	r.Register("a", &mockPlugin{name: "alpha"})
	r.Register("b", &mockPlugin{name: "beta"})
	if len(r.All()) != 2 {
		t.Fatalf("got %d plugins, want 2", len(r.All()))
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry[string, *mockPlugin]()
	r.Register("x", &mockPlugin{})
	r.Register("y", &mockPlugin{})
	if len(r.List()) != 2 {
		t.Fatalf("got %d keys, want 2", len(r.List()))
	}
}
