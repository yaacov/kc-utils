//go:build linux

package guest

import "testing"

func TestSanitizeDevice(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"/dev/sda1", "_dev_sda1"},
		{"nvme0n1p3", "nvme0n1p3"},
		{"", ""},
		{"/dev/mapper/luks-abc", "_dev_mapper_luks_abc"},
	}
	for _, tc := range cases {
		if got := sanitizeDevice(tc.input); got != tc.want {
			t.Errorf("sanitizeDevice(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStringsTrimGuest(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"/etc/fstab", "etc/fstab"},
		{"etc/fstab", "etc/fstab"},
		{"/", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := stringsTrimGuest(tc.input); got != tc.want {
			t.Errorf("stringsTrimGuest(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGuestPathFromHostNoActive(t *testing.T) {
	ClearActive()
	_, _, ok := guestPathFromHost("/some/path")
	if ok {
		t.Fatal("expected ok=false with no active guest")
	}
}

func TestGuestPathFromHostOutsideRoot(t *testing.T) {
	g := &Guest{rootPath: "/mnt/guest", mode: ModeDirect}
	SetActive(g)
	defer ClearActive()

	_, _, ok := guestPathFromHost("/tmp/outside")
	if ok {
		t.Fatal("expected ok=false for path outside guest root")
	}
}

func TestGuestPathFromHostInsideRoot(t *testing.T) {
	root := t.TempDir()
	g := &Guest{rootPath: root, mode: ModeDirect}
	SetActive(g)
	defer ClearActive()

	gp, guest, ok := guestPathFromHost(root + "/etc/fstab")
	if !ok {
		t.Fatal("expected ok=true for path inside guest root")
	}
	if guest != g {
		t.Fatal("expected same guest handle")
	}
	if gp != "/etc/fstab" {
		t.Fatalf("got guest path %q, want /etc/fstab", gp)
	}
}

func TestGuestPathFromHostAtRoot(t *testing.T) {
	root := t.TempDir()
	g := &Guest{rootPath: root, mode: ModeDirect}
	SetActive(g)
	defer ClearActive()

	gp, _, ok := guestPathFromHost(root)
	if !ok {
		t.Fatal("expected ok=true for root path")
	}
	if gp != "/" {
		t.Fatalf("got guest path %q, want /", gp)
	}
}
