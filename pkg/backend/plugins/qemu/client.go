//go:build unix

package qemu

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/yaacov/kc-utils/pkg/qemuagent/proto"
)

// agentClient speaks the primitive protocol to the in-guest agent over a unix
// socket exposed by QEMU. Calls are serialized: the agent handles one request
// per response on a single connection.
type agentClient struct {
	mu   sync.Mutex
	conn net.Conn
}

// pingReady sends a single Ping with an I/O deadline. It is used only for
// boot-readiness probing on a fresh connection; on timeout the connection is
// left in an undefined framing state and the caller must discard it.
func (c *agentClient) pingReady(timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("agent connection closed")
	}
	_ = c.conn.SetDeadline(time.Now().Add(timeout))
	defer func() { _ = c.conn.SetDeadline(time.Time{}) }()
	if err := proto.WriteMsg(c.conn, &proto.Request{Op: proto.OpPing}); err != nil {
		return err
	}
	var resp proto.Response
	return proto.ReadMsg(c.conn, &resp)
}

func (c *agentClient) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// call sends one request and returns the response. A non-empty Response.Err is
// surfaced as a Go error; Exec failures (non-zero exit) are not errors here and
// are inspected by the caller.
func (c *agentClient) call(req *proto.Request) (*proto.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, fmt.Errorf("agent connection closed")
	}
	if err := proto.WriteMsg(c.conn, req); err != nil {
		return nil, fmt.Errorf("agent write %s: %w", req.Op, err)
	}
	var resp proto.Response
	if err := proto.ReadMsg(c.conn, &resp); err != nil {
		return nil, fmt.Errorf("agent read %s: %w", req.Op, err)
	}
	if resp.Err != "" {
		return &resp, fmt.Errorf("agent %s: %s", req.Op, resp.Err)
	}
	return &resp, nil
}

// ping checks that the agent is responsive.
func (c *agentClient) ping() error {
	_, err := c.call(&proto.Request{Op: proto.OpPing})
	return err
}

// execResult is the outcome of a guest command.
type execResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// exec runs argv in the appliance and returns its output and exit code. err is
// non-nil only when the program could not be started at all.
func (c *agentClient) exec(argv []string, stdin []byte, env []string) (execResult, error) {
	resp, err := c.call(&proto.Request{
		Op:    proto.OpExec,
		Argv:  argv,
		Stdin: stdin,
		Env:   env,
	})
	if err != nil {
		return execResult{}, err
	}
	return execResult{Stdout: resp.Stdout, Stderr: resp.Stderr, ExitCode: resp.ExitCode}, nil
}

// run is exec plus a non-zero-exit-is-error convenience used by most callers.
// It merges stderr into the error for diagnostics, mirroring CombinedOutput.
func (c *agentClient) run(argv ...string) ([]byte, error) {
	res, err := c.exec(argv, nil, nil)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, fmt.Errorf("%v: exit %d: %s", argv, res.ExitCode, string(res.Stderr))
	}
	return res.Stdout, nil
}

func (c *agentClient) readFile(path string) ([]byte, error) {
	resp, err := c.call(&proto.Request{Op: proto.OpReadFile, Path: path})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *agentClient) writeFile(path string, data []byte, mode os.FileMode) error {
	_, err := c.call(&proto.Request{Op: proto.OpWriteFile, Path: path, Data: data, Mode: uint32(mode.Perm())})
	return err
}

// statResult mirrors the stat fields of a proto.Response.
type statResult struct {
	Exists bool
	IsDir  bool
	Mode   os.FileMode
	Size   int64
}

func (c *agentClient) stat(path string) (statResult, error) {
	resp, err := c.call(&proto.Request{Op: proto.OpStat, Path: path})
	if err != nil {
		return statResult{}, err
	}
	return statResult{
		Exists: resp.Exists,
		IsDir:  resp.IsDir,
		Mode:   os.FileMode(resp.Mode),
		Size:   resp.Size,
	}, nil
}

func (c *agentClient) readDir(path string) ([]proto.DirEntry, error) {
	resp, err := c.call(&proto.Request{Op: proto.OpReadDir, Path: path})
	if err != nil {
		return nil, err
	}
	return resp.Entries, nil
}

func (c *agentClient) mkdirAll(path string, mode os.FileMode) error {
	_, err := c.call(&proto.Request{Op: proto.OpMkdir, Path: path, Mode: uint32(mode.Perm())})
	return err
}

func (c *agentClient) remove(path string, recursive bool) error {
	_, err := c.call(&proto.Request{Op: proto.OpRemove, Path: path, Recursive: recursive})
	return err
}

func (c *agentClient) rename(oldPath, newPath string) error {
	_, err := c.call(&proto.Request{Op: proto.OpRename, OldPath: oldPath, NewPath: newPath})
	return err
}

func (c *agentClient) symlink(target, link string) error {
	_, err := c.call(&proto.Request{Op: proto.OpSymlink, Target: target, Link: link})
	return err
}

func (c *agentClient) readlink(path string) (string, error) {
	resp, err := c.call(&proto.Request{Op: proto.OpReadlink, Path: path})
	if err != nil {
		return "", err
	}
	return resp.Target, nil
}

func (c *agentClient) chmod(path string, mode os.FileMode) error {
	_, err := c.call(&proto.Request{Op: proto.OpChmod, Path: path, Mode: uint32(mode.Perm())})
	return err
}

func (c *agentClient) pread(path string, offset int64, length int) ([]byte, error) {
	resp, err := c.call(&proto.Request{Op: proto.OpPRead, Path: path, Offset: offset, Length: length})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *agentClient) pwrite(path string, offset int64, data []byte) (int, error) {
	resp, err := c.call(&proto.Request{Op: proto.OpPWrite, Path: path, Offset: offset, Data: data})
	if err != nil {
		return 0, err
	}
	return resp.Written, nil
}

func (c *agentClient) statFS(path string) (freeBytes, freeInodes int64, err error) {
	resp, err := c.call(&proto.Request{Op: proto.OpStatFS, Path: path})
	if err != nil {
		return 0, 0, err
	}
	return resp.FreeBytes, resp.FreeInodes, nil
}
