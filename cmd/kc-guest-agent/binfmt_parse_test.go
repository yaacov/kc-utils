package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureBinfmtFFlag(t *testing.T) {
	cases := []struct {
		in      string
		wantOK  bool
		wantHas string
	}{
		{"", false, ""},
		{"# comment", false, ""},
		{":qemu-x86_64:M::\\x7fELF:\\xff:/usr/bin/qemu-x86_64-static:OC", true, "F"},
		{":qemu-x86_64:M::\\x7fELF:\\xff:/usr/bin/qemu-x86_64-static:CF", true, "F"},
		{":qemu-x86_64:M::\\x7fELF:\\xff:/usr/bin/qemu-x86_64-static:", true, "F"},
		{":qemu-x86_64:M::\\x7fELF:\\xff:/usr/bin/qemu-x86_64-static", true, "F"},
		{"not-a-binfmt-line", false, ""},
	}
	for _, c := range cases {
		got, ok := ensureBinfmtFFlag(c.in)
		if ok != c.wantOK {
			t.Errorf("ensureBinfmtFFlag(%q) ok=%v, want %v", c.in, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if !strings.Contains(got, c.wantHas) {
			t.Errorf("ensureBinfmtFFlag(%q) = %q, want flag %s", c.in, got, c.wantHas)
		}
		if !strings.Contains(binfmtInterpreter(got), "qemu-x86_64-static") {
			t.Errorf("interpreter = %q", binfmtInterpreter(got))
		}
	}
}

func TestBinfmtTargetsNative(t *testing.T) {
	cases := []struct {
		interp, goarch string
		want           bool
	}{
		{"/usr/bin/qemu-x86_64-static", "arm64", false},
		{"/usr/bin/qemu-aarch64-static", "arm64", true},
		{"/usr/bin/qemu-aarch64-static", "amd64", false},
		{"/usr/bin/qemu-x86_64-static", "amd64", true},
		{"/usr/bin/qemu-i386-static", "amd64", true},
		{"/usr/bin/qemu-i386-static", "arm64", false},
	}
	for _, c := range cases {
		if got := binfmtTargetsNative(c.interp, c.goarch); got != c.want {
			t.Errorf("binfmtTargetsNative(%q, %q) = %v, want %v", c.interp, c.goarch, got, c.want)
		}
	}
}

func TestCollectBinfmtRegisterLinesFromConf(t *testing.T) {
	dir := t.TempDir()
	conf := `:qemu-x86_64:M::\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00:\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:/usr/bin/qemu-x86_64-static:OC
`
	if err := os.WriteFile(filepath.Join(dir, "qemu-x86_64.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := collectBinfmtRegisterLines(
		[]string{dir},
		[]string{"/usr/bin/qemu-x86_64-static"},
		"arm64",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("got %d lines: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "F") {
		t.Errorf("missing F flag: %s", lines[0])
	}
	if binfmtName(lines[0]) != "qemu-x86_64" {
		t.Errorf("name = %q", binfmtName(lines[0]))
	}
}

func TestCollectBinfmtRegisterLinesFallbackAndSkipNative(t *testing.T) {
	lines, err := collectBinfmtRegisterLines(
		nil,
		[]string{"/usr/bin/qemu-x86_64-static", "/usr/bin/qemu-aarch64-static"},
		"arm64",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1 (skip native aarch64): %v", len(lines), lines)
	}
	if binfmtName(lines[0]) != "qemu-x86_64" {
		t.Errorf("name = %q", binfmtName(lines[0]))
	}
	if !strings.HasSuffix(binfmtInterpreter(lines[0]), "qemu-x86_64-static") {
		t.Errorf("interpreter = %q", binfmtInterpreter(lines[0]))
	}
}
