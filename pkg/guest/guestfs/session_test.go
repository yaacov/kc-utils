//go:build linux

package guestfs

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseListenOutput(t *testing.T) {
	pid, err := parseListenOutput("GUESTFISH_PID=12345; export GUESTFISH_PID;\n")
	if err != nil {
		t.Fatal(err)
	}
	if pid != 12345 {
		t.Fatalf("pid=%d", pid)
	}

	if _, err := parseListenOutput("no pid here\n"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDrainGuestfishListenStdout(t *testing.T) {
	r, w := io.Pipe()
	pidCh := make(chan int, 1)
	done := make(chan error, 1)
	go func() {
		_, err := drainGuestfishListenStdout(r, pidCh)
		done <- err
	}()

	_, _ = w.Write([]byte("GUESTFISH_PID="))
	_, _ = w.Write([]byte("4242; export GUESTFISH_PID;\n"))

	select {
	case pid := <-pidCh:
		if pid != 4242 {
			t.Fatalf("pid=%d", pid)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PID (stdout drain must not require EOF)")
	}

	_, _ = w.Write([]byte("more noise\n"))
	_ = w.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestGuestfishSocketPath(t *testing.T) {
	p := guestfishSocketPath(100)
	if !strings.Contains(p, "/socket-100") {
		t.Fatalf("path=%q", p)
	}
	if !strings.Contains(p, ".guestfish-") {
		t.Fatalf("path=%q", p)
	}
}

func TestEnvGuestfishPIDPrefersKC(t *testing.T) {
	t.Setenv(EnvGuestfishPID, "111")
	t.Setenv(EnvKCGuestfishPID, "222")
	pid, ok := envGuestfishPID()
	if !ok || pid != 222 {
		t.Fatalf("got pid=%d ok=%v", pid, ok)
	}

	t.Setenv(EnvKCGuestfishPID, "")
	pid, ok = envGuestfishPID()
	if !ok || pid != 111 {
		t.Fatalf("fallback pid=%d ok=%v", pid, ok)
	}

	t.Setenv(EnvGuestfishPID, "")
	if _, ok := envGuestfishPID(); ok {
		t.Fatal("expected no pid")
	}
}

func TestSharedListenerEnv(t *testing.T) {
	t.Setenv("V2V_memSize", "")
	t.Setenv("V2V_smp", "")
	t.Setenv("LIBGUESTFS_MEMSIZE", "2048")
	t.Setenv("LIBGUESTFS_SMP", "4")
	l := &SharedListener{PID: 99}
	env := l.Env()
	want := map[string]bool{
		fmt.Sprintf("%s=99", EnvGuestfishPID):   true,
		fmt.Sprintf("%s=99", EnvKCGuestfishPID): true,
		"LIBGUESTFS_BACKEND=direct":             true,
		"LIBGUESTFS_MEMSIZE=2048":               true,
		"LIBGUESTFS_SMP=4":                      true,
		"HOME=" + guestfishHome:                 true,
	}
	if len(env) != len(want) {
		t.Fatalf("env=%v (want %d entries)", env, len(want))
	}
	for _, e := range env {
		if !want[e] {
			t.Fatalf("unexpected %q in %v", e, env)
		}
	}
	if (&SharedListener{}).Env() != nil {
		t.Fatal("zero listener should return nil env")
	}
}

func TestSharedListenerEnvV2VPriority(t *testing.T) {
	t.Setenv("V2V_memSize", "3072")
	t.Setenv("V2V_smp", "8")
	t.Setenv("LIBGUESTFS_MEMSIZE", "2048")
	t.Setenv("LIBGUESTFS_SMP", "4")

	l := &SharedListener{PID: 42}
	env := l.Env()
	want := map[string]bool{
		fmt.Sprintf("%s=42", EnvGuestfishPID):   true,
		fmt.Sprintf("%s=42", EnvKCGuestfishPID): true,
		"LIBGUESTFS_BACKEND=direct":             true,
		"LIBGUESTFS_MEMSIZE=3072":               true,
		"LIBGUESTFS_SMP=8":                      true,
		"HOME=" + guestfishHome:                 true,
	}
	if len(env) != len(want) {
		t.Fatalf("env=%v (want %d entries)", env, len(want))
	}
	for _, e := range env {
		if !want[e] {
			t.Fatalf("unexpected %q in %v", e, env)
		}
	}
}

func TestEnsureGuestfishEnvSetsDefaults(t *testing.T) {
	t.Setenv("V2V_memSize", "")
	t.Setenv("V2V_smp", "")
	t.Setenv("LIBGUESTFS_BACKEND", "")
	t.Setenv("LIBGUESTFS_MEMSIZE", "")
	t.Setenv("LIBGUESTFS_SMP", "")

	if err := ensureGuestfishEnv(); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("LIBGUESTFS_BACKEND"); got != "direct" {
		t.Errorf("LIBGUESTFS_BACKEND=%q, want direct", got)
	}
	if got := os.Getenv("LIBGUESTFS_MEMSIZE"); got != "2048" {
		t.Errorf("LIBGUESTFS_MEMSIZE=%q, want 2048", got)
	}
	smp := os.Getenv("LIBGUESTFS_SMP")
	if smp == "" || smp == "0" {
		t.Errorf("LIBGUESTFS_SMP=%q, want >0", smp)
	}
}

func TestEnsureGuestfishEnvV2VWins(t *testing.T) {
	t.Setenv("V2V_memSize", "2048")
	t.Setenv("V2V_smp", "6")
	t.Setenv("LIBGUESTFS_MEMSIZE", "4096")
	t.Setenv("LIBGUESTFS_SMP", "2")

	if err := ensureGuestfishEnv(); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("LIBGUESTFS_MEMSIZE"); got != "2048" {
		t.Errorf("LIBGUESTFS_MEMSIZE=%q, want 2048 (V2V_memSize wins)", got)
	}
	if got := os.Getenv("LIBGUESTFS_SMP"); got != "6" {
		t.Errorf("LIBGUESTFS_SMP=%q, want 6 (V2V_smp wins)", got)
	}
}

func TestEnsureGuestfishEnvLibguestfsFallback(t *testing.T) {
	t.Setenv("V2V_memSize", "")
	t.Setenv("V2V_smp", "")
	t.Setenv("LIBGUESTFS_MEMSIZE", "4096")
	t.Setenv("LIBGUESTFS_SMP", "2")

	if err := ensureGuestfishEnv(); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("LIBGUESTFS_MEMSIZE"); got != "4096" {
		t.Errorf("LIBGUESTFS_MEMSIZE=%q, want 4096 (LIBGUESTFS_ fallback)", got)
	}
	if got := os.Getenv("LIBGUESTFS_SMP"); got != "2" {
		t.Errorf("LIBGUESTFS_SMP=%q, want 2 (LIBGUESTFS_ fallback)", got)
	}
}

func TestEnsureGuestfishEnvInvalidIgnored(t *testing.T) {
	t.Setenv("V2V_memSize", "notanumber")
	t.Setenv("V2V_smp", "0")
	t.Setenv("LIBGUESTFS_MEMSIZE", "")
	t.Setenv("LIBGUESTFS_SMP", "")

	if err := ensureGuestfishEnv(); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("LIBGUESTFS_MEMSIZE"); got != "2048" {
		t.Errorf("LIBGUESTFS_MEMSIZE=%q, want 2048 (invalid V2V_ falls to default)", got)
	}
	smp := os.Getenv("LIBGUESTFS_SMP")
	if smp == "" || smp == "0" {
		t.Errorf("LIBGUESTFS_SMP=%q, want >0 (V2V_smp=0 falls to default)", smp)
	}
}

func TestSessionCloseSkipsExternal(t *testing.T) {
	s := &guestfishSession{pid: 1, ownedExternally: true}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	if s.pid != 0 {
		t.Fatalf("pid should be cleared, got %d", s.pid)
	}
}

func TestAttachSessionAdoptsEnv(t *testing.T) {
	t.Setenv(EnvGuestfishPID, "")
	t.Setenv(EnvKCGuestfishPID, "")
	b := &Backend{diskPaths: []string{"/tmp/disk.img"}}
	if err := b.attachSession(); err != nil {
		t.Fatal(err)
	}
	if b.session != nil {
		t.Fatal("expected no session without env PID")
	}
}

func TestAttachSessionFailsDeadSharedPID(t *testing.T) {
	t.Setenv(EnvKCGuestfishPID, "1")
	t.Setenv(EnvGuestfishPID, "")
	b := &Backend{diskPaths: []string{"/tmp/disk.img"}}
	err := b.attachSession()
	if err == nil {
		t.Fatal("expected error for dead shared PID")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("error=%v", err)
	}
	if b.session != nil {
		t.Fatal("session must stay nil after failed attach")
	}
}

func TestOpenGuestfishSessionFailsDeadSharedPID(t *testing.T) {
	t.Setenv(EnvKCGuestfishPID, "1")
	t.Setenv(EnvGuestfishPID, "")
	_, err := openGuestfishSession()
	if err == nil {
		t.Fatal("expected error for dead shared PID")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("error=%v", err)
	}
}

func TestReleaseExitsLocalSession(t *testing.T) {
	b := &Backend{
		session: &guestfishSession{pid: 0, ownedExternally: false},
	}
	if err := b.Release(); err != nil {
		t.Fatal(err)
	}
	if b.session != nil {
		t.Fatal("Release should clear session")
	}
}

func TestCheckAliveNilSession(t *testing.T) {
	var s *guestfishSession
	if err := s.checkAlive(); err == nil {
		t.Fatal("expected error for nil session")
	}
}

func TestCheckAliveZeroPID(t *testing.T) {
	s := &guestfishSession{pid: 0}
	if err := s.checkAlive(); err == nil {
		t.Fatal("expected error for zero PID")
	}
}

func TestCheckAliveDeadPID(t *testing.T) {
	s := &guestfishSession{pid: 999999}
	err := s.checkAlive()
	if err == nil {
		t.Fatal("expected error for dead PID")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("error should mention 'not running', got: %v", err)
	}
}

func TestReleaseKeepsExternalSemantics(t *testing.T) {
	b := &Backend{
		session: &guestfishSession{pid: 4242, ownedExternally: true},
	}
	if err := b.Release(); err != nil {
		t.Fatal(err)
	}
	if b.session != nil {
		t.Fatal("Release should clear backend session pointer")
	}
}
