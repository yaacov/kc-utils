// Package server implements the in-appliance kc-guest-agent: it serves the
// primitive operations defined in pkg/qemuagent/proto over a connection.
//
// The agent is intentionally "dumb": each handler maps one primitive to a
// single os / os/exec call on appliance paths. All orchestration (discovery,
// LVM, LUKS, mounting, fs-checks, chroot) lives host-side in the qemu backend,
// which composes these primitives. The agent runs inside a trusted, throwaway
// appliance VM, so Exec is unrestricted.
package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/yaacov/kc-utils/pkg/qemuagent/proto"
)

// Serve reads requests from rw and writes one response per request until the
// connection closes (io.EOF) or a transport error occurs. Requests are handled
// one at a time; the host serializes calls on a single connection.
func Serve(rw io.ReadWriter) error {
	for {
		var req proto.Request
		if err := proto.ReadMsg(rw, &req); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}
		if err := proto.WriteMsg(rw, handle(&req)); err != nil {
			return err
		}
	}
}

func handle(req *proto.Request) *proto.Response {
	switch req.Op {
	case proto.OpPing:
		return &proto.Response{}
	case proto.OpExec:
		return handleExec(req)
	case proto.OpReadFile:
		return handleReadFile(req)
	case proto.OpWriteFile:
		return handleWriteFile(req)
	case proto.OpStat:
		return handleStat(req)
	case proto.OpReadDir:
		return handleReadDir(req)
	case proto.OpMkdir:
		return handleMkdir(req)
	case proto.OpRemove:
		return handleRemove(req)
	case proto.OpRename:
		return handleRename(req)
	case proto.OpSymlink:
		return handleSymlink(req)
	case proto.OpReadlink:
		return handleReadlink(req)
	case proto.OpChmod:
		return handleChmod(req)
	case proto.OpPRead:
		return handlePRead(req)
	case proto.OpPWrite:
		return handlePWrite(req)
	case proto.OpStatFS:
		return handleStatFS(req)
	default:
		return errResp(fmt.Errorf("unknown op %q", req.Op))
	}
}

func errResp(err error) *proto.Response {
	if err == nil {
		return &proto.Response{}
	}
	return &proto.Response{Err: err.Error()}
}

func handleExec(req *proto.Request) *proto.Response {
	if len(req.Argv) == 0 {
		return errResp(errors.New("exec: empty argv"))
	}
	cmd := exec.Command(req.Argv[0], req.Argv[1:]...)
	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	}
	if len(req.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	resp := &proto.Response{}
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			resp.ExitCode = ee.ExitCode()
		} else {
			// Could not start the program at all (not found, permission).
			resp.Err = fmt.Sprintf("exec %s: %v", req.Argv[0], err)
		}
	}
	resp.Stdout = stdout.Bytes()
	resp.Stderr = stderr.Bytes()
	return resp
}

func handleReadFile(req *proto.Request) *proto.Response {
	data, err := os.ReadFile(req.Path)
	if err != nil {
		return errResp(err)
	}
	return &proto.Response{Data: data}
}

func handleWriteFile(req *proto.Request) *proto.Response {
	return errResp(os.WriteFile(req.Path, req.Data, os.FileMode(req.Mode)))
}

func handleStat(req *proto.Request) *proto.Response {
	fi, err := os.Lstat(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &proto.Response{Exists: false}
		}
		return errResp(err)
	}
	return &proto.Response{
		Exists: true,
		IsDir:  fi.IsDir(),
		Mode:   uint32(fi.Mode()),
		Size:   fi.Size(),
	}
}

func handleReadDir(req *proto.Request) *proto.Response {
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		return errResp(err)
	}
	out := make([]proto.DirEntry, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			// Entry vanished between listing and stat; skip it.
			continue
		}
		out = append(out, proto.DirEntry{
			Name:  e.Name(),
			IsDir: fi.IsDir(),
			Mode:  uint32(fi.Mode()),
			Size:  fi.Size(),
		})
	}
	return &proto.Response{Entries: out}
}

func handleMkdir(req *proto.Request) *proto.Response {
	return errResp(os.MkdirAll(req.Path, os.FileMode(req.Mode)))
}

func handleRemove(req *proto.Request) *proto.Response {
	if req.Recursive {
		return errResp(os.RemoveAll(req.Path))
	}
	return errResp(os.Remove(req.Path))
}

func handleRename(req *proto.Request) *proto.Response {
	return errResp(os.Rename(req.OldPath, req.NewPath))
}

func handleSymlink(req *proto.Request) *proto.Response {
	return errResp(os.Symlink(req.Target, req.Link))
}

func handleReadlink(req *proto.Request) *proto.Response {
	target, err := os.Readlink(req.Path)
	if err != nil {
		return errResp(err)
	}
	return &proto.Response{Target: target}
}

func handleChmod(req *proto.Request) *proto.Response {
	return errResp(os.Chmod(req.Path, os.FileMode(req.Mode)))
}

func handlePRead(req *proto.Request) *proto.Response {
	// Guard the allocation: a negative length would panic make(), and an
	// oversized one cannot fit in a response frame anyway.
	if req.Length < 0 {
		return errResp(fmt.Errorf("pread: negative length %d", req.Length))
	}
	if req.Length > proto.MaxMessageSize {
		return errResp(fmt.Errorf("pread: length %d exceeds max %d", req.Length, proto.MaxMessageSize))
	}
	f, err := os.Open(req.Path)
	if err != nil {
		return errResp(err)
	}
	defer f.Close()
	buf := make([]byte, req.Length)
	n, err := f.ReadAt(buf, req.Offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return errResp(err)
	}
	return &proto.Response{Data: buf[:n]}
}

func handlePWrite(req *proto.Request) *proto.Response {
	f, err := os.OpenFile(req.Path, os.O_WRONLY, 0)
	if err != nil {
		return errResp(err)
	}
	defer f.Close()
	n, err := f.WriteAt(req.Data, req.Offset)
	if err != nil {
		return errResp(err)
	}
	return &proto.Response{Written: n}
}
