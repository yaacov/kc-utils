package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInputTrimsJSONCredentials(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.json")
	body := `{"host":"h","vm_name":"v","fingerprint":"f","username":"  ","password":"\t"}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	in, err := loadInput(p, "", "", "cli-user", "cli-pass", false, "", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if in.Username != "cli-user" || in.Password != "cli-pass" {
		t.Fatalf("got %q/%q, want CLI credentials for whitespace JSON fields", in.Username, in.Password)
	}
}

func TestLoadInputKeepsJSONCredentials(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.json")
	body := `{"host":"h","vm_name":"v","fingerprint":"f","username":"json-user","password":"json-pass"}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	in, err := loadInput(p, "", "", "cli-user", "cli-pass", false, "", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if in.Username != "json-user" || in.Password != "json-pass" {
		t.Fatalf("got %q/%q, want JSON credentials", in.Username, in.Password)
	}
}

func TestReadPasswordFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "pass")
	if err := os.WriteFile(p, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readPassword(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret" {
		t.Fatalf("got %q, want secret without trailing newline", got)
	}
	empty, err := readPassword("")
	if err != nil || empty != "" {
		t.Fatalf("empty path: got %q err %v", empty, err)
	}
}

func TestResolvePasswordPrefersFlag(t *testing.T) {
	got, err := resolvePassword("flag-pass", "/nope")
	if err != nil {
		t.Fatal(err)
	}
	if got != "flag-pass" {
		t.Fatalf("got %q, want flag-pass", got)
	}
}
