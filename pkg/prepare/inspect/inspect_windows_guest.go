//go:build unix

package inspect

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

var openRegistryHive = func(hivePath string) (registry.Hive, error) {
	editor, ok := registry.Editors.Get("hivex")
	if !ok {
		return nil, fmt.Errorf("registry editor 'hivex' not registered")
	}

	hostPath := hivePath
	var cleanup func()
	if g := guest.Active(); g != nil {
		rel, err := filepath.Rel(g.RootPath(), hivePath)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			guestPath := "/" + filepath.ToSlash(rel)
			checkedOut, c, err := g.CheckoutReadOnly(guestPath)
			if err != nil {
				return nil, err
			}
			hostPath = checkedOut
			cleanup = c
		}
	}

	hive, err := editor.OpenHive(hostPath)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	if cleanup != nil {
		return &checkoutHive{Hive: hive, cleanup: cleanup}, nil
	}
	return hive, nil
}

// checkoutHive wraps a hive opened from a read-only checkout and cleans up on Close.
type checkoutHive struct {
	registry.Hive
	cleanup func()
}

func (h *checkoutHive) Close() error {
	err := h.Hive.Close()
	h.cleanup()
	return err
}

func inspectWindows(root string) (*types.InspectData, error) {
	data := &types.InspectData{
		Type:   "windows",
		Distro: "windows",
	}

	systemRoot := findWindowsDir(root, "Windows")
	if systemRoot == "" {
		systemRoot = "Windows"
	}

	arch := detectWindowsArch(root, systemRoot)
	data.Arch = arch

	if meta, err := inspectWindowsMetadata(root, systemRoot); err == nil {
		if meta.ProductName != "" {
			data.ProductName = meta.ProductName
		}
		if meta.MajorVersion > 0 {
			data.MajorVersion = meta.MajorVersion
		}
		if meta.MinorVersion > 0 {
			data.MinorVersion = meta.MinorVersion
		}
	}

	return data, nil
}

func findWindowsDir(root, name string) string {
	entries, err := guest.FileReadDir(root)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name, name) && e.IsDir {
			return e.Name
		}
	}
	return ""
}

func detectWindowsArch(root, systemRoot string) string {
	sys32 := filepath.Join(root, systemRoot, "System32")
	sysWow := filepath.Join(root, systemRoot, "SysWOW64")

	if guest.FileExists(sysWow) {
		return "x86_64"
	}

	if guest.FileExists(sys32) {
		return "i386"
	}

	return "unknown"
}

// InspectWindowsMetadata returns Windows-specific prepare output fields read
// from registry hives when available.
func InspectWindowsMetadata(root string) (*types.WindowsInspect, error) {
	systemRoot := findWindowsDir(root, "Windows")
	if systemRoot == "" {
		systemRoot = "Windows"
	}

	meta, err := inspectWindowsMetadata(root, systemRoot)
	if err != nil {
		return &types.WindowsInspect{
			SystemRoot:    systemRoot,
			SystemHive:    filepath.Join(systemRoot, "System32", "config", "SYSTEM"),
			SoftwareHive:  filepath.Join(systemRoot, "System32", "config", "SOFTWARE"),
			DriveMappings: map[string]string{},
		}, err
	}

	return &types.WindowsInspect{
		SystemRoot:        systemRoot,
		CurrentControlSet: meta.CurrentControlSet,
		SystemHive:        meta.SystemHive,
		SoftwareHive:      meta.SoftwareHive,
		DriveMappings:     meta.DriveMappings,
	}, nil
}

type windowsMetadata struct {
	ProductName       string
	MajorVersion      int
	MinorVersion      int
	CurrentControlSet int
	SystemHive        string
	SoftwareHive      string
	DriveMappings     map[string]string
}

func inspectWindowsMetadata(root, systemRoot string) (*windowsMetadata, error) {
	systemHiveRel := filepath.Join(systemRoot, "System32", "config", "SYSTEM")
	softwareHiveRel := filepath.Join(systemRoot, "System32", "config", "SOFTWARE")

	meta := &windowsMetadata{
		SystemHive:    systemHiveRel,
		SoftwareHive:  softwareHiveRel,
		DriveMappings: map[string]string{},
	}

	softwareHive, err := openRegistryHive(filepath.Join(root, softwareHiveRel))
	if err == nil {
		defer softwareHive.Close()
		populateWindowsVersion(meta, softwareHive)
	}

	systemHive, sysErr := openRegistryHive(filepath.Join(root, systemHiveRel))
	if sysErr == nil {
		defer systemHive.Close()
		populateCurrentControlSet(meta, systemHive)
	}

	if err != nil {
		return meta, err
	}
	if sysErr != nil {
		return meta, sysErr
	}

	return meta, nil
}

func populateWindowsVersion(meta *windowsMetadata, softwareHive registry.Hive) {
	const currentVersionKey = `Microsoft\Windows NT\CurrentVersion`

	if productName, err := softwareHive.GetString(currentVersionKey, "ProductName"); err == nil {
		meta.ProductName = strings.TrimSpace(productName)
	}

	major, majorErr := softwareHive.GetDWORD(currentVersionKey, "CurrentMajorVersionNumber")
	minor, minorErr := softwareHive.GetDWORD(currentVersionKey, "CurrentMinorVersionNumber")
	if majorErr == nil {
		meta.MajorVersion = int(major)
	}
	if minorErr == nil {
		meta.MinorVersion = int(minor)
	}
	if majorErr == nil {
		return
	}

	version, err := softwareHive.GetString(currentVersionKey, "CurrentVersion")
	if err != nil {
		return
	}
	parts := strings.SplitN(strings.TrimSpace(version), ".", 2)
	if len(parts) >= 1 {
		if major, err := strconv.Atoi(parts[0]); err == nil {
			meta.MajorVersion = major
		}
	}
	if len(parts) == 2 {
		if minor, err := strconv.Atoi(parts[1]); err == nil {
			meta.MinorVersion = minor
		}
	}
}

func populateCurrentControlSet(meta *windowsMetadata, systemHive registry.Hive) {
	if current, err := systemHive.GetDWORD(`Select`, "Current"); err == nil {
		meta.CurrentControlSet = int(current)
	}
}
