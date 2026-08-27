package vsphere

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCredentialsPrefersExplicit(t *testing.T) {
	dir := t.TempDir()
	userFile := filepath.Join(dir, "accessKeyId")
	passFile := filepath.Join(dir, "secretKey")
	if err := os.WriteFile(userFile, []byte("file-user\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passFile, []byte("file-pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	user, pass, err := readCredentials("vpx://url-user@vc/dc", "flag-user", "flag-pass", userFile, passFile)
	if err != nil {
		t.Fatal(err)
	}
	if user != "flag-user" || pass != "flag-pass" {
		t.Fatalf("got %q/%q, want flag credentials", user, pass)
	}
}

func TestReadCredentialsFallsBackToFiles(t *testing.T) {
	dir := t.TempDir()
	userFile := filepath.Join(dir, "accessKeyId")
	passFile := filepath.Join(dir, "secretKey")
	if err := os.WriteFile(userFile, []byte("file-user\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passFile, []byte("file-pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	user, pass, err := readCredentials("vpx://url-user@vc/dc", "", "", userFile, passFile)
	if err != nil {
		t.Fatal(err)
	}
	if user != "file-user" || pass != "file-pass" {
		t.Fatalf("got %q/%q, want file credentials", user, pass)
	}
}

func TestReadCredentialsFallsBackToURLUser(t *testing.T) {
	dir := t.TempDir()
	passFile := filepath.Join(dir, "secretKey")
	if err := os.WriteFile(passFile, []byte("file-pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	user, pass, err := readCredentials("vpx://url-user@vc/dc", "", "", filepath.Join(dir, "missing-user"), passFile)
	if err != nil {
		t.Fatal(err)
	}
	if user != "url-user" || pass != "file-pass" {
		t.Fatalf("got %q/%q, want URL user and file password", user, pass)
	}
}

func TestReadCredentialsEmpty(t *testing.T) {
	dir := t.TempDir()
	_, _, err := readCredentials("", "", "", filepath.Join(dir, "u"), filepath.Join(dir, "p"))
	if err == nil {
		t.Fatal("expected empty credentials error")
	}
}

func TestReadCredentialsPreservesExplicitSpaces(t *testing.T) {
	dir := t.TempDir()
	userFile := filepath.Join(dir, "accessKeyId")
	passFile := filepath.Join(dir, "secretKey")
	if err := os.WriteFile(userFile, []byte("file-user\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passFile, []byte("file-pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	user, pass, err := readCredentials("", " flag-user ", " flag-pass ", userFile, passFile)
	if err != nil {
		t.Fatal(err)
	}
	if user != " flag-user " || pass != " flag-pass " {
		t.Fatalf("got %q/%q, want explicit credentials with spaces", user, pass)
	}
}

func TestReadCredentialsWhitespaceFallsBackToFiles(t *testing.T) {
	dir := t.TempDir()
	userFile := filepath.Join(dir, "accessKeyId")
	passFile := filepath.Join(dir, "secretKey")
	if err := os.WriteFile(userFile, []byte("file-user\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passFile, []byte("file-pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	user, pass, err := readCredentials("vpx://url-user@vc/dc", "  ", "\t", userFile, passFile)
	if err != nil {
		t.Fatal(err)
	}
	if user != "file-user" || pass != "file-pass" {
		t.Fatalf("got %q/%q, want file credentials for whitespace-only flags", user, pass)
	}
}
