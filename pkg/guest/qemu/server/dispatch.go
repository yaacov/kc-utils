//go:build linux

package server

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/yaacov/kc-utils/pkg/guest/qemu/protocol"
)

type handler func(*Agent, json.RawMessage, []byte) ([]byte, error)

var handlers = map[string]handler{
	protocol.OpPing:               (*Agent).ping,
	protocol.OpDiscover:           (*Agent).opDiscover,
	protocol.OpSetRoot:            (*Agent).setRoot,
	protocol.OpMount:              (*Agent).opMount,
	protocol.OpUnmountAll:         (*Agent).opUnmountAll,
	protocol.OpUnmountFilesystems: (*Agent).opUnmountAll,
	protocol.OpProbeMount:         (*Agent).probeMount,
	protocol.OpProbeUnmount:       (*Agent).probeUnmount,
	protocol.OpFSType:             (*Agent).opFSType,
	protocol.OpBlkidAttr:          (*Agent).blkidAttr,
	protocol.OpFSCheck:            (*Agent).opFSCheck,
	protocol.OpFSTrim:             (*Agent).opFSTrim,
	protocol.OpDecrypt:            (*Agent).opDecrypt,
	protocol.OpUnlockClevis:       (*Agent).unlockClevis,
	protocol.OpCloseCrypt:         (*Agent).closeCrypt,
	protocol.OpRescanBlock:        (*Agent).opDiscover,
	protocol.OpRunCommand:         (*Agent).runCommand,
	protocol.OpDeviceRead:         (*Agent).deviceRead,
	protocol.OpDeviceWrite:        (*Agent).deviceWrite,
	protocol.OpReadFile:           (*Agent).readFile,
	protocol.OpWriteFile:          (*Agent).writeFile,
	protocol.OpExists:             (*Agent).exists,
	protocol.OpIsDir:              (*Agent).isDir,
	protocol.OpGlob:               (*Agent).glob,
	protocol.OpRemove:             (*Agent).remove,
	protocol.OpRemoveAll:          (*Agent).removeAll,
	protocol.OpRename:             (*Agent).rename,
	protocol.OpSymlink:            (*Agent).symlink,
	protocol.OpReadlink:           (*Agent).readlink,
	protocol.OpChmod:              (*Agent).chmod,
	protocol.OpMkdirAll:           (*Agent).mkdirAll,
	protocol.OpReadDir:            (*Agent).readDir,
	protocol.OpUpload:             (*Agent).upload,
	protocol.OpDownload:           (*Agent).download,
	protocol.OpStatFS:             (*Agent).statFS,
	protocol.OpSync:               (*Agent).sync,
	protocol.OpReleaseDevices:     (*Agent).opReleaseDevices,
	protocol.OpMergeHive:          (*Agent).mergeHive,
}

func (a *Agent) ping(_ json.RawMessage, _ []byte) ([]byte, error) {
	return marshal(protocol.StringResult{Value: "ok"}), nil
}

func (a *Agent) opDiscover(_ json.RawMessage, _ []byte) ([]byte, error) {
	return a.discover(), nil
}

func (a *Agent) setRoot(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	if args.Path != "" {
		a.guestRoot = args.Path
	}
	return nil, nil
}

func (a *Agent) opMount(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.MountArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, a.mount(args)
}

func (a *Agent) opUnmountAll(_ json.RawMessage, _ []byte) ([]byte, error) {
	a.unmountAll()
	return nil, nil
}

func (a *Agent) probeMount(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.ProbeMountArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, a.mount(protocol.MountArgs{
		Device: args.Device, MountPoint: args.MountPoint, FSType: args.FSType, ReadOnly: true,
	})
}

func (a *Agent) probeUnmount(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, a.unmount(args.Path)
}

func (a *Agent) opFSType(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	ft, err := detectFS(args.Path)
	return marshal(protocol.StringResult{Value: ft}), err
}

func (a *Agent) blkidAttr(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.BlkidArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	out, err := exec.Command("blkid", "-o", "value", "-s", args.Attr, args.Device).Output()
	return marshal(protocol.StringResult{Value: strings.TrimSpace(string(out))}), err
}

func (a *Agent) opFSCheck(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.FSCheckArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, fscheck(args.Device, args.FSType)
}

func (a *Agent) opFSTrim(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, exec.Command("fstrim", "-v", args.Path).Run()
}

func (a *Agent) opDecrypt(raw json.RawMessage, blob []byte) ([]byte, error) {
	var args protocol.DecryptArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	mapped, err := a.decrypt(args, blob)
	return marshal(protocol.StringResult{Value: mapped}), err
}

func (a *Agent) unlockClevis(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.UnlockClevisArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	if out, err := exec.Command("clevis", "luks", "unlock", "-d", args.Device, "-n", args.MapperName).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("clevis unlock: %w\n%s", err, out)
	}
	a.cryptMaps = append(a.cryptMaps, args.MapperName)
	return marshal(protocol.StringResult{Value: "/dev/mapper/" + args.MapperName}), nil
}

func (a *Agent) closeCrypt(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, exec.Command("cryptsetup", "close", args.Path).Run()
}

func (a *Agent) runCommand(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.RunCommandArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	root := args.GuestRoot
	if root == "" {
		root = a.guestRoot
	}
	cmdArgs := append([]string{root}, args.Cmd...)
	out, err := exec.Command("chroot", cmdArgs...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("chroot %v: %w\n%s", args.Cmd, err, out)
	}
	return out, nil
}

func (a *Agent) deviceRead(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.DeviceRWArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	f, err := os.Open(args.Device)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, args.Size)
	_, err = f.ReadAt(buf, args.Offset)
	return buf, err
}

func (a *Agent) deviceWrite(raw json.RawMessage, blob []byte) ([]byte, error) {
	var args protocol.DeviceRWArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(args.Device, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = f.WriteAt(blob, args.Offset)
	return nil, err
}

func (a *Agent) readFile(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return os.ReadFile(a.host(args.Path))
}

func (a *Agent) writeFile(raw json.RawMessage, blob []byte) ([]byte, error) {
	var args protocol.WriteFileArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	p := a.host(args.Path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, err
	}
	perm := args.Perm
	if perm == 0 {
		perm = 0o644
	}
	return nil, os.WriteFile(p, blob, perm)
}

func (a *Agent) exists(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	_, err := os.Stat(a.host(args.Path))
	return marshal(protocol.BoolResult{Value: err == nil}), nil
}

func (a *Agent) isDir(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	info, err := os.Stat(a.host(args.Path))
	return marshal(protocol.BoolResult{Value: err == nil && info.IsDir()}), nil
}

func (a *Agent) glob(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(a.host(args.Path))
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, m := range matches {
		rel, err := filepath.Rel(a.guestRoot, m)
		if err != nil {
			continue
		}
		paths = append(paths, "/"+filepath.ToSlash(rel))
	}
	return marshal(protocol.PathsResult{Paths: paths}), nil
}

func (a *Agent) remove(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Remove(a.host(args.Path))
}

func (a *Agent) removeAll(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.RemoveAll(a.host(args.Path))
}

func (a *Agent) rename(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.RenameArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Rename(a.host(args.Old), a.host(args.New))
}

func (a *Agent) symlink(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.SymlinkArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Symlink(args.Target, a.host(args.Link))
}

func (a *Agent) readlink(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	v, err := os.Readlink(a.host(args.Path))
	return marshal(protocol.StringResult{Value: v}), err
}

func (a *Agent) chmod(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.ChmodArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Chmod(a.host(args.Path), args.Mode)
}

func (a *Agent) mkdirAll(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.MkdirArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	perm := args.Perm
	if perm == 0 {
		perm = 0o755
	}
	return nil, os.MkdirAll(a.host(args.Path), perm)
}

func (a *Agent) readDir(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(a.host(args.Path))
	if err != nil {
		return nil, err
	}
	var out []protocol.DirEntry
	for _, e := range entries {
		mode := os.FileMode(0)
		if info, err := e.Info(); err == nil {
			mode = info.Mode()
		}
		out = append(out, protocol.DirEntry{Name: e.Name(), IsDir: e.IsDir(), Mode: mode})
	}
	return marshal(protocol.DirResult{Entries: out}), nil
}

func (a *Agent) upload(raw json.RawMessage, blob []byte) ([]byte, error) {
	var args protocol.UploadArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	p := a.host(args.GuestPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, err
	}
	return nil, os.WriteFile(p, blob, 0o644)
}

func (a *Agent) download(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return os.ReadFile(a.host(args.Path))
}

func (a *Agent) statFS(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(a.host(args.Path), &st); err != nil {
		return nil, err
	}
	ffree := int64(st.Ffree)
	if st.Files == 0 && st.Ffree == 0 {
		ffree = -1
	}
	return marshal(protocol.StatFSResult{
		FreeBytes:  int64(st.Bavail) * int64(st.Bsize),
		FreeInodes: ffree,
	}), nil
}

func (a *Agent) sync(_ json.RawMessage, _ []byte) ([]byte, error) {
	syscall.Sync()
	return nil, nil
}

func (a *Agent) opReleaseDevices(_ json.RawMessage, _ []byte) ([]byte, error) {
	a.releaseDevices()
	return nil, nil
}

func (a *Agent) mergeHive(raw json.RawMessage, blob []byte) ([]byte, error) {
	var args protocol.MergeHiveArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp("", "kc-reg-*.reg")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(blob); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()
	hive := a.host(args.Path)
	out, err := exec.Command("hivexregedit", "--merge", hive, tmpPath).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("hivexregedit: %w\n%s", err, out)
	}
	return nil, nil
}
