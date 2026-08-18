//go:build unix

package selinux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSELinuxConfigTargeted(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "etc", "selinux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "SELINUX=enforcing\nSELINUXTYPE=targeted\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	policy, disabled, err := readSELinuxConfig(root)
	if err != nil {
		t.Fatalf("readSELinuxConfig: %v", err)
	}
	if policy != "targeted" {
		t.Errorf("policy = %q, want targeted", policy)
	}
	if disabled {
		t.Error("expected disabled=false")
	}
}

func TestReadSELinuxConfigDisabled(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "etc", "selinux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "SELINUX=disabled\nSELINUXTYPE=targeted\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	_, disabled, err := readSELinuxConfig(root)
	if err != nil {
		t.Fatalf("readSELinuxConfig: %v", err)
	}
	if !disabled {
		t.Error("expected disabled=true")
	}
}

func TestReadSELinuxConfigDefaultPolicy(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "etc", "selinux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "SELINUX=enforcing\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	policy, _, err := readSELinuxConfig(root)
	if err != nil {
		t.Fatalf("readSELinuxConfig: %v", err)
	}
	if policy != "targeted" {
		t.Errorf("policy = %q, want targeted (default)", policy)
	}
}

func TestReadSELinuxConfigMLS(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "etc", "selinux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "SELINUX=enforcing\nSELINUXTYPE=mls\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	policy, _, err := readSELinuxConfig(root)
	if err != nil {
		t.Fatalf("readSELinuxConfig: %v", err)
	}
	if policy != "mls" {
		t.Errorf("policy = %q, want mls", policy)
	}
}

func TestFindSetfiles(t *testing.T) {
	root := t.TempDir()

	if p := findSetfiles(root); p != "" {
		t.Errorf("expected empty, got %q", p)
	}

	usrSbin := filepath.Join(root, "usr", "sbin")
	if err := os.MkdirAll(usrSbin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrSbin, "setfiles"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if p := findSetfiles(root); p != "/usr/sbin/setfiles" {
		t.Errorf("expected /usr/sbin/setfiles, got %q", p)
	}
}

func TestFindSetfilesSbin(t *testing.T) {
	root := t.TempDir()
	sbin := filepath.Join(root, "sbin")
	if err := os.MkdirAll(sbin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sbin, "setfiles"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if p := findSetfiles(root); p != "/sbin/setfiles" {
		t.Errorf("expected /sbin/setfiles, got %q", p)
	}
}

func TestMountPointsForSetfiles(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect []string
	}{
		{"nil", nil, []string{"/"}},
		{"empty", []string{}, []string{"/"}},
		{"root only", []string{"/"}, []string{"/"}},
		{"with boot", []string{"/", "/boot"}, []string{"/", "/boot"}},
		{"missing root", []string{"/boot", "/home"}, []string{"/", "/boot", "/home"}},
		{"dedup", []string{"/", "/boot", "/boot"}, []string{"/", "/boot"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mountPointsForSetfiles(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("len = %d, want %d: %v", len(got), len(tt.expect), got)
			}
			for i, v := range got {
				if v != tt.expect[i] {
					t.Errorf("[%d] = %q, want %q", i, v, tt.expect[i])
				}
			}
		})
	}
}

func TestRelabelNoSELinux(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}

	relabeled, err := Relabel(root, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if relabeled {
		t.Error("expected relabeled=false when no /etc/selinux")
	}
}

func TestRelabelSELinuxDisabled(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "etc", "selinux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "SELINUX=disabled\nSELINUXTYPE=targeted\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	relabeled, err := Relabel(root, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if relabeled {
		t.Error("expected relabeled=false when SELinux disabled")
	}
}

func TestRelabelNoSpecFile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "etc", "selinux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "SELINUX=enforcing\nSELINUXTYPE=targeted\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Relabel(root, nil)
	if err == nil {
		t.Fatal("expected error for missing spec file")
	}
}

func TestRelabelNoSetfiles(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "etc", "selinux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "SELINUX=enforcing\nSELINUXTYPE=targeted\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	specDir := filepath.Join(root, "etc", "selinux", "targeted", "contexts", "files")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "file_contexts"), []byte("/.*\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Relabel(root, nil)
	if err == nil {
		t.Fatal("expected error for missing setfiles binary")
	}
}
