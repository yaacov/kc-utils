//go:build unix

package backend

import "testing"

func TestValidateName(t *testing.T) {
	if err := ValidateName(NameDirect); err != nil {
		t.Fatalf("direct: %v", err)
	}
	if err := ValidateName(NameGuestfs); err != nil {
		t.Fatalf("guestfs: %v", err)
	}
	if err := ValidateName("bogus"); err == nil {
		t.Fatal("expected error for bogus backend")
	}
}

func TestResolveUnknownBackend(t *testing.T) {
	_, err := Resolve("bogus")
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}
