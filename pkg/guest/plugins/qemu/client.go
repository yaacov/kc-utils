//go:build unix

package qemu

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
)

type client struct {
	mu   sync.Mutex
	conn net.Conn
	next uint64
}

func dialAgent(sock string) (*client, error) {
	c, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("dial kc-agent %s: %w", sock, err)
	}
	cl := &client{conn: c}
	if err := cl.call(protocol.OpPing, nil, nil); err != nil {
		c.Close()
		return nil, fmt.Errorf("kc-agent ping: %w", err)
	}
	return cl, nil
}

func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *client) call(op string, args any, result any) error {
	_, err := c.callBlob(op, args, nil, result)
	return err
}

func (c *client) callBlob(op string, args any, blob []byte, result any) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, fmt.Errorf("kc-agent connection closed")
	}
	c.next++
	req := protocol.Request{ID: c.next, Op: op, Size: int64(len(blob))}
	if args != nil {
		raw, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		req.Args = raw
	}
	if err := protocol.WriteFrame(c.conn, req); err != nil {
		return nil, err
	}
	if len(blob) > 0 {
		if err := protocol.WriteBlob(c.conn, blob); err != nil {
			return nil, err
		}
	}
	var resp protocol.Response
	if err := protocol.ReadFrame(c.conn, &resp); err != nil {
		return nil, err
	}
	if resp.ID != req.ID {
		return nil, fmt.Errorf("rpc id mismatch: got %d want %d", resp.ID, req.ID)
	}
	if !resp.OK {
		if resp.Error == "" {
			resp.Error = "agent error"
		}
		return nil, fmt.Errorf("%s: %s", op, resp.Error)
	}
	var out []byte
	if resp.Size > 0 {
		b, err := protocol.ReadBlob(c.conn)
		if err != nil {
			return nil, err
		}
		out = b
	}
	if result != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return out, err
		}
	}
	return out, nil
}
