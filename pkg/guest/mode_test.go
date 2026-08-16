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
		{Mode(""), "direct"},
	}
	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("Mode(%q).String() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}
