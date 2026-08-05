package keyvalue

import "testing"

const testConfig = `FOO="bar"
BAZ="qux"
# a comment
EMPTY=""
`

func TestGet(t *testing.T) {
	f := Parse(testConfig)
	if got := f.Get("FOO"); got != "bar" {
		t.Errorf("got %q, want bar", got)
	}
	if got := f.Get("MISSING"); got != "" {
		t.Errorf("got %q for missing key, want empty", got)
	}
}

func TestSet(t *testing.T) {
	f := Parse(testConfig)
	f.Set("FOO", "new")
	if got := f.Get("FOO"); got != "new" {
		t.Errorf("got %q, want new", got)
	}
	f.Set("NEW_KEY", "value")
	if got := f.Get("NEW_KEY"); got != "value" {
		t.Errorf("got %q, want value", got)
	}
}

func TestDelete(t *testing.T) {
	f := Parse(testConfig)
	f.Delete("BAZ")
	if got := f.Get("BAZ"); got != "" {
		t.Errorf("got %q after delete, want empty", got)
	}
}
