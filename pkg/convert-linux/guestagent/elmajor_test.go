//go:build unix

package guestagent

import "testing"

func TestLocalPackageMajorVersionAmazonLinux(t *testing.T) {
	if got := localPackageMajorVersion("amzn", 2023); got != 9 {
		t.Errorf("amzn 2023 = %d, want 9", got)
	}
	if got := localPackageMajorVersion("amzn", 2); got != 7 {
		t.Errorf("amzn 2 = %d, want 7", got)
	}
}

func TestLocalPackageMajorVersionFedora(t *testing.T) {
	if got := localPackageMajorVersion("fedora", 39); got != 0 {
		t.Errorf("fedora 39 = %d, want 0", got)
	}
}

func TestLocalPackageMajorVersionRHEL(t *testing.T) {
	if got := localPackageMajorVersion("rhel", 9); got != 9 {
		t.Errorf("rhel 9 = %d, want 9", got)
	}
}
