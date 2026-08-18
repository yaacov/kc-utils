//go:build linux

package guestfs

import (
	"errors"
	"strings"
	"testing"
)

func TestTruncateLog(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"hello world", 5, "hello…"},
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"", 5, ""},
		{"abc", 0, "abc"},
	}
	for _, tc := range cases {
		if got := truncateLog(tc.input, tc.max); got != tc.want {
			t.Errorf("truncateLog(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
		}
	}
}

func TestPathError(t *testing.T) {
	if pathError("read", "/etc/fstab", nil) != nil {
		t.Fatal("expected nil for nil error")
	}
	err := pathError("write", "/etc/hostname", errors.New("permission denied"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	want := "write /etc/hostname: permission denied"
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestClearDirNonexistent(t *testing.T) {
	if err := clearDir("/nonexistent/path/that/does/not/exist"); err != nil {
		t.Fatalf("clearDir on nonexistent dir should return nil, got %v", err)
	}
}

func TestGlobSplit(t *testing.T) {
	cases := []struct {
		pattern    string
		wantRoot   string
		wantSuffix string
	}{
		{"/lib/modules/5.14/kernel/drivers/*/*.ko*", "/lib/modules/5.14/kernel/drivers", "*/*.ko*"},
		{"/etc/rc*.d/[SK]*kudzu", "/etc", "rc*.d/[SK]*kudzu"},
		{"/netplan/*.yaml", "/netplan", "*.yaml"},
		{"/a/b/c", "/a/b/c", ""},
		{"*", "/", "*"},
	}
	for _, tc := range cases {
		root, suffix := globSplit(tc.pattern)
		if root != tc.wantRoot || suffix != tc.wantSuffix {
			t.Errorf("globSplit(%q) = (%q, %q), want (%q, %q)", tc.pattern, root, suffix, tc.wantRoot, tc.wantSuffix)
		}
	}
}

func TestPrefixDash(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			"adds dash to plain commands",
			"mount /dev/sda1 /\nls /\n",
			"-mount /dev/sda1 /\n-ls /\n",
		},
		{
			"preserves existing dash prefix",
			"-mount /dev/sda1 /\n-ls /\n",
			"-mount /dev/sda1 /\n-ls /\n",
		},
		{
			"skips empty lines",
			"mount /dev/sda1 /\n\n\nls /\n",
			"-mount /dev/sda1 /\n-ls /\n",
		},
		{
			"handles mixed prefixed and unprefixed",
			"-mount /dev/sda1 /\nls /\n-cat /etc/fstab\n",
			"-mount /dev/sda1 /\n-ls /\n-cat /etc/fstab\n",
		},
		{
			"empty script",
			"",
			"",
		},
		{
			"whitespace-only lines are skipped",
			"  \n\t\nmount /\n",
			"-mount /\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := prefixDash(tc.input)
			if got != tc.want {
				t.Errorf("prefixDash(%q) =\n%q\nwant:\n%q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExtractGuestfsError(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
	}{
		{
			"no error",
			"/dev/sda1\n/dev/sda2\n",
			"",
		},
		{
			"single error line",
			"libguestfs: error: download: /etc/cloud/cloud.cfg: No such file or directory",
			"libguestfs: error: download: /etc/cloud/cloud.cfg: No such file or directory",
		},
		{
			"error among normal output",
			"some output\nlibguestfs: error: mount failed\nmore output\n",
			"libguestfs: error: mount failed",
		},
		{
			"empty output",
			"",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractGuestfsError(tc.output)
			if got != tc.want {
				t.Errorf("extractGuestfsError(%q) = %q, want %q", tc.output, got, tc.want)
			}
		})
	}
}

func TestExtractAllGuestfsErrors(t *testing.T) {
	out := "ok\nlibguestfs: error: mount: /dev/sda1: failed\nlibguestfs: error: copy_out: missing\n"
	got := extractAllGuestfsErrors(out)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %v", len(got), got)
	}
	if !strings.Contains(got[0], "mount:") || !strings.Contains(got[1], "copy_out:") {
		t.Fatalf("unexpected: %v", got)
	}
}

func TestShellQuote(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "''"},
		{"simple", "simple"},
		{"--force", "--force"},
		{"/boot/initramfs.img", "/boot/initramfs.img"},
		{"virtio virtio_blk", "'virtio virtio_blk'"},
		{"it's", `'it'\''s'`},
		{"hello world", "'hello world'"},
		{"$HOME", "'$HOME'"},
		{"a`b", "'a`b'"},
		{"a;b", "'a;b'"},
	}
	for _, tc := range cases {
		got := shellQuote(tc.input)
		if got != tc.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGlobMatchSegments(t *testing.T) {
	cases := []struct {
		relPath string
		pattern string
		want    bool
	}{
		{"virtio/virtio_blk.ko.xz", "*/*.ko*", true},
		{"block/virtio_blk.ko", "*/*.ko*", true},
		{"virtio/virtio_blk.so", "*/*.ko*", false},
		{"deep/nested/file.ko", "*/*.ko*", false},
		{"file.ko", "*/*.ko*", false},
		{"rc5.d/S10kudzu", "rc*.d/[SK]*kudzu", true},
		{"rc5.d/X10kudzu", "rc*.d/[SK]*kudzu", false},
		{"foo.yaml", "*.yaml", true},
		{"foo.yml", "*.yaml", false},
	}
	for _, tc := range cases {
		got := globMatchSegments(tc.relPath, tc.pattern)
		if got != tc.want {
			t.Errorf("globMatchSegments(%q, %q) = %v, want %v", tc.relPath, tc.pattern, got, tc.want)
		}
	}
}
