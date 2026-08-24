//go:build unix

package guestfs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/backend/windowsvol"
)

func (b *Backend) checkUnsupportedWindowsVolumes() error {
	var issues []windowsvol.Issue
	for _, diskPath := range b.diskPaths {
		diskDev, err := b.diskDeviceName(diskPath)
		if err != nil {
			continue
		}
		issues = append(issues, b.scanDiskWindowsVolumes(diskDev)...)
	}
	if issue := windowsvol.FirstUnsupported(issues, backend.NameGuestfs); issue != nil {
		return windowsvol.UnsupportedError(issue.Kind, issue.Device, backend.NameGuestfs)
	}
	return nil
}

func (b *Backend) scanDiskWindowsVolumes(diskDev string) []windowsvol.Issue {
	out, err := b.session.remote("list-partitions")
	if err != nil {
		return nil
	}
	var issues []windowsvol.Issue
	for _, partDev := range partitionsForDisk(diskDev, parseListPartitions(string(out))) {
		fsType := ""
		if ftOut, ftErr := b.session.remote("vfs-type " + quoteGuestfish(partDev)); ftErr == nil {
			fsType = strings.TrimSpace(string(ftOut))
		}
		partNum := b.partitionNumber(partDev)
		if partNum <= 0 {
			continue
		}
		partType := b.partitionType(diskDev, partNum)
		if kind, ok := windowsvol.Classify(partType, fsType); ok {
			issues = append(issues, windowsvol.Issue{Kind: kind, Device: partDev})
		}
	}
	return issues
}

func (b *Backend) partitionNumber(partDev string) int {
	out, err := b.session.remote("part-to-partnum " + quoteGuestfish(partDev))
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func (b *Backend) partitionType(diskDev string, partNum int) string {
	if out, err := b.session.remoteScript(fmt.Sprintf("-part-get-gpt-type %s %d\n", quoteGuestfish(diskDev), partNum)); err == nil {
		if pt := strings.TrimSpace(string(out)); pt != "" {
			return pt
		}
	}
	if out, err := b.session.remoteScript(fmt.Sprintf("-part-get-mbr-id %s %d\n", quoteGuestfish(diskDev), partNum)); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// parseListPartitions extracts partition device paths from guestfish list-partitions output.
func parseListPartitions(out string) []string {
	var devs []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/") {
			devs = append(devs, line)
		}
	}
	return devs
}

// partitionsForDisk returns partition devices belonging to diskDev.
func partitionsForDisk(diskDev string, all []string) []string {
	var out []string
	for _, partDev := range all {
		if deviceBelongsToDisk(partDev, diskDev) {
			out = append(out, partDev)
		}
	}
	return out
}
