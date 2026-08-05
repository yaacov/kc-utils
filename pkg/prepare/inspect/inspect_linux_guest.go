//go:build linux

package inspect

import (
	"bufio"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

func inspectLinux(root string) (*types.InspectData, error) {
	data := &types.InspectData{
		Type: "linux",
	}

	osRelease := filepath.Join(root, "etc", "os-release")
	if guest.FileExists(osRelease) {
		if err := parseOSRelease(osRelease, data); err == nil {
			return data, nil
		}
	}

	redhatRelease := filepath.Join(root, "etc", "redhat-release")
	if content, err := guest.FileRead(redhatRelease); err == nil {
		data.ProductName = strings.TrimSpace(string(content))
		data.Distro = "rhel"
		return data, nil
	}

	debianVersion := filepath.Join(root, "etc", "debian_version")
	if content, err := guest.FileRead(debianVersion); err == nil {
		data.ProductName = "Debian " + strings.TrimSpace(string(content))
		data.Distro = "debian"
		return data, nil
	}

	return data, nil
}

func parseOSRelease(path string, data *types.InspectData) error {
	raw, err := guest.FileRead(path)
	if err != nil {
		return err
	}

	kv := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := strings.Trim(line[idx+1:], "\"'")
		kv[key] = val
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	data.Distro = kv["ID"]
	data.ProductName = kv["PRETTY_NAME"]

	if ver := kv["VERSION_ID"]; ver != "" {
		parts := strings.SplitN(ver, ".", 2)
		if major, err := strconv.Atoi(parts[0]); err == nil {
			data.MajorVersion = major
		}
		if len(parts) > 1 {
			if minor, err := strconv.Atoi(parts[1]); err == nil {
				data.MinorVersion = minor
			}
		}
	}

	return nil
}
