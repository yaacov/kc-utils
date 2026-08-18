//go:build linux

package guest

import "testing"

func TestSetActiveAndClear(t *testing.T) {
	if Active() != nil {
		t.Fatal("expected nil before SetActive")
	}
	g := &Guest{rootPath: "/tmp/test", backendName: BackendDirect}
	SetActive(g)
	if Active() != g {
		t.Fatal("Active() should return the set guest")
	}
	ClearActive()
	if Active() != nil {
		t.Fatal("expected nil after ClearActive")
	}
}

func TestRunInGuestNoActive(t *testing.T) {
	ClearActive()
	_, err := RunInGuest("/", []string{"echo", "hi"})
	if err == nil {
		t.Fatal("expected error with no active guest")
	}
}

func TestBlkidUUIDNoActive(t *testing.T) {
	ClearActive()
	if BlkidUUID("/dev/sda1") != "" {
		t.Fatal("expected empty string with no active guest")
	}
}
