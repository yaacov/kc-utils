package grub

import (
	"strings"
	"testing"
)

const testGrub = `# GRUB defaults
GRUB_TIMEOUT=5
GRUB_DISTRIBUTOR="CentOS"
GRUB_DEFAULT=saved
GRUB_CMDLINE_LINUX="rd.lvm.lv=centos/root rhgb quiet vga=791"
`

func TestParse(t *testing.T) {
	c := Parse(testGrub)
	if c.Get("GRUB_TIMEOUT") != "5" {
		t.Errorf("GRUB_TIMEOUT = %q, want 5", c.Get("GRUB_TIMEOUT"))
	}
	if c.Get("GRUB_DISTRIBUTOR") != "CentOS" {
		t.Errorf("GRUB_DISTRIBUTOR = %q, want CentOS", c.Get("GRUB_DISTRIBUTOR"))
	}
	if c.Get("GRUB_DEFAULT") != "saved" {
		t.Errorf("GRUB_DEFAULT = %q, want saved", c.Get("GRUB_DEFAULT"))
	}
}

func TestSet(t *testing.T) {
	c := Parse(testGrub)
	c.Set("GRUB_TIMEOUT", "10")
	if c.Get("GRUB_TIMEOUT") != "10" {
		t.Errorf("GRUB_TIMEOUT = %q, want 10", c.Get("GRUB_TIMEOUT"))
	}
}

func TestGetKernelArgs(t *testing.T) {
	c := Parse(testGrub)
	args := c.GetKernelArgs()
	if len(args) != 4 {
		t.Fatalf("got %d kernel args, want 4", len(args))
	}
	if args[0] != "rd.lvm.lv=centos/root" {
		t.Errorf("first arg = %q, want rd.lvm.lv=centos/root", args[0])
	}
	if args[1] != "rhgb" {
		t.Errorf("second arg = %q, want rhgb", args[1])
	}
}

func TestAddKernelArg(t *testing.T) {
	c := Parse(testGrub)
	c.AddKernelArg("console=ttyS0")
	args := c.GetKernelArgs()
	found := false
	for _, a := range args {
		if a == "console=ttyS0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("console=ttyS0 should be in kernel args")
	}
}

func TestAddKernelArgExisting(t *testing.T) {
	c := Parse(testGrub)
	c.AddKernelArg("rhgb")
	args := c.GetKernelArgs()
	count := 0
	for _, a := range args {
		if a == "rhgb" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("rhgb appears %d times, want 1", count)
	}
}

func TestRemoveKernelArg(t *testing.T) {
	c := Parse(testGrub)
	c.RemoveKernelArg("rhgb")
	args := c.GetKernelArgs()
	for _, a := range args {
		if a == "rhgb" {
			t.Error("rhgb should have been removed")
		}
	}
	if len(args) != 3 {
		t.Errorf("got %d args after remove, want 3", len(args))
	}
}

func TestRemoveKernelArgPrefix(t *testing.T) {
	c := Parse(testGrub)
	c.RemoveKernelArg("vga")
	args := c.GetKernelArgs()
	for _, a := range args {
		if strings.HasPrefix(a, "vga=") {
			t.Error("vga= arg should have been removed")
		}
	}
	if len(args) != 3 {
		t.Errorf("got %d args after prefix remove, want 3", len(args))
	}
}

func TestString(t *testing.T) {
	c := Parse(testGrub)
	out := c.String()
	if !strings.Contains(out, "GRUB_TIMEOUT=5") {
		t.Error("output should contain GRUB_TIMEOUT")
	}
	if !strings.Contains(out, "# GRUB defaults") {
		t.Error("output should preserve comments")
	}
}
