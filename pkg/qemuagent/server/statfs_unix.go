//go:build unix

package server

import (
	"syscall"

	"github.com/yaacov/kc-utils/pkg/qemuagent/proto"
)

// handleStatFS returns free bytes and free inodes for the filesystem containing
// the requested path. syscall.Statfs is available on both linux (appliance) and
// darwin (local tests); field types differ, so values are widened to int64.
func handleStatFS(req *proto.Request) *proto.Response {
	var st syscall.Statfs_t
	if err := syscall.Statfs(req.Path, &st); err != nil {
		return errResp(err)
	}
	return &proto.Response{
		FreeBytes:  int64(st.Bavail) * int64(st.Bsize),
		FreeInodes: int64(st.Ffree),
	}
}
