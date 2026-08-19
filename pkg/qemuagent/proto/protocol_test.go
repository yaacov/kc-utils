package proto

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestMessageRoundTrip(t *testing.T) {
	req := Request{
		Op:    OpExec,
		Argv:  []string{"echo", "hello"},
		Stdin: []byte("input\x00binary"),
		Env:   []string{"FOO=bar"},
	}
	var buf bytes.Buffer
	if err := WriteMsg(&buf, &req); err != nil {
		t.Fatalf("WriteMsg: %v", err)
	}
	var got Request
	if err := ReadMsg(&buf, &got); err != nil {
		t.Fatalf("ReadMsg: %v", err)
	}
	if got.Op != req.Op || len(got.Argv) != 2 || got.Argv[1] != "hello" {
		t.Fatalf("argv mismatch: %+v", got)
	}
	if !bytes.Equal(got.Stdin, req.Stdin) {
		t.Fatalf("stdin mismatch: %q", got.Stdin)
	}
	if len(got.Env) != 1 || got.Env[0] != "FOO=bar" {
		t.Fatalf("env mismatch: %+v", got.Env)
	}
}

func TestBinaryPayloadPreserved(t *testing.T) {
	// Base64 encoding must survive arbitrary bytes (e.g. a boot sector).
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i)
	}
	var buf bytes.Buffer
	if err := WriteMsg(&buf, &Response{Data: data}); err != nil {
		t.Fatalf("WriteMsg: %v", err)
	}
	var got Response
	if err := ReadMsg(&buf, &got); err != nil {
		t.Fatalf("ReadMsg: %v", err)
	}
	if !bytes.Equal(got.Data, data) {
		t.Fatalf("data corrupted")
	}
}

func TestReadMsgEOFAtBoundary(t *testing.T) {
	var buf bytes.Buffer
	var got Request
	if err := ReadMsg(&buf, &got); err != io.EOF {
		t.Fatalf("want io.EOF at empty boundary, got %v", err)
	}
}

func TestTwoMessagesSequential(t *testing.T) {
	var buf bytes.Buffer
	_ = WriteMsg(&buf, &Request{Op: OpPing})
	_ = WriteMsg(&buf, &Request{Op: OpStat, Path: "/etc/fstab"})

	var a, b Request
	if err := ReadMsg(&buf, &a); err != nil || a.Op != OpPing {
		t.Fatalf("first: %+v err=%v", a, err)
	}
	if err := ReadMsg(&buf, &b); err != nil || b.Op != OpStat || b.Path != "/etc/fstab" {
		t.Fatalf("second: %+v err=%v", b, err)
	}
}

func TestMessageTooLarge(t *testing.T) {
	// A header claiming a huge frame must be rejected, not allocated.
	hdr := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	var got Request
	if err := ReadMsg(bytes.NewReader(hdr), &got); err == nil {
		t.Fatalf("expected error for oversized frame")
	}
}

func TestOmitemptyKeepsFramesSmall(t *testing.T) {
	// A ping should not serialize op-specific fields.
	payload, _ := json.Marshal(&Request{Op: OpPing})
	if bytes.Contains(payload, []byte("argv")) || bytes.Contains(payload, []byte("offset")) {
		t.Fatalf("unexpected fields in ping: %s", payload)
	}
}
