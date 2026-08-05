package distro

import "testing"

func TestFormat(t *testing.T) {
	if got := Format("debian"); got != "deb" {
		t.Errorf("debian: expected deb, got %q", got)
	}
	if got := Format("rhel"); got != "rpm" {
		t.Errorf("rhel: expected rpm, got %q", got)
	}
}
