package server

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yaacov/kc-utils/pkg/qemuagent/proto"
)

// newTestConn starts Serve on one end of a pipe and returns the other end plus
// a round-trip helper.
func newTestConn(t *testing.T) func(proto.Request) proto.Response {
	t.Helper()
	c1, c2 := net.Pipe()
	done := make(chan struct{})
	go func() {
		_ = Serve(c2)
		close(done)
	}()
	t.Cleanup(func() {
		c1.Close()
		<-done
	})
	return func(req proto.Request) proto.Response {
		t.Helper()
		if err := proto.WriteMsg(c1, &req); err != nil {
			t.Fatalf("WriteMsg: %v", err)
		}
		var resp proto.Response
		if err := proto.ReadMsg(c1, &resp); err != nil {
			t.Fatalf("ReadMsg: %v", err)
		}
		return resp
	}
}

func TestPing(t *testing.T) {
	call := newTestConn(t)
	if resp := call(proto.Request{Op: proto.OpPing}); resp.Err != "" {
		t.Fatalf("ping err: %s", resp.Err)
	}
}

func TestExecEcho(t *testing.T) {
	call := newTestConn(t)
	resp := call(proto.Request{Op: proto.OpExec, Argv: []string{"echo", "-n", "hi"}})
	if resp.Err != "" {
		t.Fatalf("exec err: %s", resp.Err)
	}
	if resp.ExitCode != 0 || string(resp.Stdout) != "hi" {
		t.Fatalf("unexpected: code=%d out=%q", resp.ExitCode, resp.Stdout)
	}
}

func TestExecStdinAndExitCode(t *testing.T) {
	call := newTestConn(t)
	resp := call(proto.Request{Op: proto.OpExec, Argv: []string{"cat"}, Stdin: []byte("piped")})
	if resp.Err != "" || string(resp.Stdout) != "piped" {
		t.Fatalf("cat stdin: err=%s out=%q", resp.Err, resp.Stdout)
	}

	resp = call(proto.Request{Op: proto.OpExec, Argv: []string{"false"}})
	if resp.Err != "" {
		t.Fatalf("false should not error at transport level: %s", resp.Err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("false should have nonzero exit code")
	}
}

func TestExecNotFound(t *testing.T) {
	call := newTestConn(t)
	resp := call(proto.Request{Op: proto.OpExec, Argv: []string{"kc-no-such-binary-xyz"}})
	if resp.Err == "" {
		t.Fatalf("expected Err for missing binary")
	}
}

func TestFileOps(t *testing.T) {
	dir := t.TempDir()
	call := newTestConn(t)
	path := filepath.Join(dir, "sub", "file.txt")

	if resp := call(proto.Request{Op: proto.OpMkdir, Path: filepath.Dir(path), Mode: 0o755}); resp.Err != "" {
		t.Fatalf("mkdir: %s", resp.Err)
	}
	if resp := call(proto.Request{Op: proto.OpWriteFile, Path: path, Data: []byte("hello"), Mode: 0o644}); resp.Err != "" {
		t.Fatalf("writefile: %s", resp.Err)
	}
	if resp := call(proto.Request{Op: proto.OpReadFile, Path: path}); resp.Err != "" || string(resp.Data) != "hello" {
		t.Fatalf("readfile: err=%s data=%q", resp.Err, resp.Data)
	}
	if resp := call(proto.Request{Op: proto.OpStat, Path: path}); !resp.Exists || resp.IsDir || resp.Size != 5 {
		t.Fatalf("stat: %+v", resp)
	}
	if resp := call(proto.Request{Op: proto.OpStat, Path: filepath.Join(dir, "missing")}); resp.Err != "" || resp.Exists {
		t.Fatalf("stat missing should report Exists=false without error: %+v", resp)
	}

	entries := call(proto.Request{Op: proto.OpReadDir, Path: filepath.Dir(path)})
	if entries.Err != "" || len(entries.Entries) != 1 || entries.Entries[0].Name != "file.txt" {
		t.Fatalf("readdir: %+v", entries)
	}

	newPath := filepath.Join(dir, "sub", "renamed.txt")
	if resp := call(proto.Request{Op: proto.OpRename, OldPath: path, NewPath: newPath}); resp.Err != "" {
		t.Fatalf("rename: %s", resp.Err)
	}
	if resp := call(proto.Request{Op: proto.OpChmod, Path: newPath, Mode: 0o600}); resp.Err != "" {
		t.Fatalf("chmod: %s", resp.Err)
	}
	if resp := call(proto.Request{Op: proto.OpRemove, Path: newPath}); resp.Err != "" {
		t.Fatalf("remove: %s", resp.Err)
	}
}

func TestSymlinkReadlink(t *testing.T) {
	dir := t.TempDir()
	call := newTestConn(t)
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if resp := call(proto.Request{Op: proto.OpSymlink, Target: target, Link: link}); resp.Err != "" {
		t.Fatalf("symlink: %s", resp.Err)
	}
	if resp := call(proto.Request{Op: proto.OpReadlink, Path: link}); resp.Err != "" || resp.Target != target {
		t.Fatalf("readlink: err=%s target=%q", resp.Err, resp.Target)
	}
}

func TestPReadPWrite(t *testing.T) {
	dir := t.TempDir()
	call := newTestConn(t)
	path := filepath.Join(dir, "raw.img")
	// Pre-size a file so PWrite has room (simulates a partition device).
	if err := os.WriteFile(path, make([]byte, 32), 0o644); err != nil {
		t.Fatal(err)
	}
	if resp := call(proto.Request{Op: proto.OpPWrite, Path: path, Offset: 8, Data: []byte("XYZW")}); resp.Err != "" || resp.Written != 4 {
		t.Fatalf("pwrite: err=%s written=%d", resp.Err, resp.Written)
	}
	resp := call(proto.Request{Op: proto.OpPRead, Path: path, Offset: 8, Length: 4})
	if resp.Err != "" || string(resp.Data) != "XYZW" {
		t.Fatalf("pread: err=%s data=%q", resp.Err, resp.Data)
	}
}

func TestPReadInvalidLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.img")
	if err := os.WriteFile(path, make([]byte, 16), 0o644); err != nil {
		t.Fatal(err)
	}
	call := newTestConn(t)

	// Negative length must not panic make(); it returns an error response.
	if resp := call(proto.Request{Op: proto.OpPRead, Path: path, Offset: 0, Length: -1}); resp.Err == "" {
		t.Fatalf("expected error for negative length")
	}
	// Oversized length is rejected before allocation.
	if resp := call(proto.Request{Op: proto.OpPRead, Path: path, Offset: 0, Length: proto.MaxMessageSize + 1}); resp.Err == "" {
		t.Fatalf("expected error for oversized length")
	}
	// A valid length still works after the guards.
	if resp := call(proto.Request{Op: proto.OpPRead, Path: path, Offset: 0, Length: 4}); resp.Err != "" || len(resp.Data) != 4 {
		t.Fatalf("valid pread: err=%s len=%d", resp.Err, len(resp.Data))
	}
}

func TestStatFS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("statfs is unix-only")
	}
	call := newTestConn(t)
	resp := call(proto.Request{Op: proto.OpStatFS, Path: t.TempDir()})
	if resp.Err != "" {
		t.Fatalf("statfs: %s", resp.Err)
	}
	if resp.FreeBytes <= 0 {
		t.Fatalf("expected positive free bytes, got %d", resp.FreeBytes)
	}
}

func TestUnknownOp(t *testing.T) {
	call := newTestConn(t)
	if resp := call(proto.Request{Op: "bogus"}); resp.Err == "" {
		t.Fatalf("expected error for unknown op")
	}
}
