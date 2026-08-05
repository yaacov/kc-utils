//go:build linux

package guest

import "testing"

func TestModeString(t *testing.T) {
	cases := []struct {
		mode Mode
		want string
	}{
		{ModeDirect, "direct"},
		{ModeGuestfs, "guestfs"},
		{Mode(99), "direct"},
	}
	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("Mode(%d).String() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestModeFromBool(t *testing.T) {
	if ModeFromBool(false) != ModeDirect {
		t.Fatal("ModeFromBool(false) should be ModeDirect")
	}
	if ModeFromBool(true) != ModeGuestfs {
		t.Fatal("ModeFromBool(true) should be ModeGuestfs")
	}
}
