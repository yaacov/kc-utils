package bls

import (
	"strings"
	"testing"
)

const testBLS = `title CentOS Linux (5.4.0) 8
version 5.4.0
linux /vmlinuz-5.4.0
initrd /initramfs-5.4.0.img
options rd.lvm.lv=centos/root quiet
`

func TestParse(t *testing.T) {
	e := Parse(testBLS)
	if e.Get("title") != "CentOS Linux (5.4.0) 8" {
		t.Errorf("title = %q, want CentOS Linux (5.4.0) 8", e.Get("title"))
	}
	if e.Get("version") != "5.4.0" {
		t.Errorf("version = %q, want 5.4.0", e.Get("version"))
	}
	if e.Get("linux") != "/vmlinuz-5.4.0" {
		t.Errorf("linux = %q, want /vmlinuz-5.4.0", e.Get("linux"))
	}
}

func TestSet(t *testing.T) {
	e := Parse(testBLS)
	e.Set("version", "5.14.0")
	if e.Get("version") != "5.14.0" {
		t.Errorf("version = %q, want 5.14.0", e.Get("version"))
	}
	e.Set("grub_arg", "--unrestricted")
	if e.Get("grub_arg") != "--unrestricted" {
		t.Errorf("grub_arg = %q, want --unrestricted", e.Get("grub_arg"))
	}
}

func TestString(t *testing.T) {
	e := Parse(testBLS)
	out := e.String()
	if !strings.Contains(out, "title CentOS Linux (5.4.0) 8") {
		t.Error("output should contain title line")
	}
	if !strings.Contains(out, "linux /vmlinuz-5.4.0") {
		t.Error("output should contain linux line")
	}
	if !strings.Contains(out, "options rd.lvm.lv=centos/root quiet") {
		t.Error("output should contain options line")
	}
}
