package distro

import "testing"

func TestName(t *testing.T) {
	if got := Name("debian"); got != "apt" {
		t.Errorf("debian: expected apt, got %q", got)
	}
	if got := Name("sles"); got != "zypper" {
		t.Errorf("sles: expected zypper, got %q", got)
	}
	if got := Name("rhel"); got != "dnf" {
		t.Errorf("rhel: expected dnf, got %q", got)
	}
}
