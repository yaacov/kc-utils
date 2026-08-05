//go:build linux

package resolve

import (
	"fmt"
	"os"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest"
)

// Catalog maps device paths, UUIDs, and labels to block devices.
type Catalog struct {
	ByPath  map[string]string
	ByUUID  map[string]string
	ByLabel map[string]string
}

// NewCatalog builds a lookup table from known block devices via the guest wrapper.
func NewCatalog(g *guest.Guest, devices []string) (*Catalog, error) {
	c := &Catalog{
		ByPath:  make(map[string]string),
		ByUUID:  make(map[string]string),
		ByLabel: make(map[string]string),
	}
	for _, dev := range devices {
		c.ByPath[dev] = dev
		if uuid, err := g.BlkidAttr(dev, "UUID"); err == nil && uuid != "" {
			c.ByUUID[strings.ToLower(uuid)] = dev
		}
		if label, err := g.BlkidAttr(dev, "LABEL"); err == nil && label != "" {
			c.ByLabel[label] = dev
		}
	}
	return c, nil
}

// Resolve maps an fstab device specifier to a host block device path.
func (c *Catalog) Resolve(spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", fmt.Errorf("empty device specifier")
	}
	if strings.HasPrefix(spec, "UUID=") {
		uuid := strings.ToLower(strings.TrimPrefix(spec, "UUID="))
		if dev, ok := c.ByUUID[uuid]; ok {
			return dev, nil
		}
		return "", fmt.Errorf("UUID %q not found", uuid)
	}
	if strings.HasPrefix(spec, "LABEL=") {
		label := strings.TrimPrefix(spec, "LABEL=")
		if dev, ok := c.ByLabel[label]; ok {
			return dev, nil
		}
		return "", fmt.Errorf("LABEL %q not found", label)
	}
	if strings.HasPrefix(spec, "/dev/") {
		if dev, ok := c.ByPath[spec]; ok {
			return dev, nil
		}
		if _, err := os.Stat(spec); err == nil {
			return spec, nil
		}
		return "", fmt.Errorf("device %q not found", spec)
	}
	return "", fmt.Errorf("unsupported device specifier %q", spec)
}

// AllDevices returns partition paths plus optional LVM volume paths.
func AllDevices(disks []string, lvPaths []string) []string {
	out := append([]string{}, disks...)
	out = append(out, lvPaths...)
	return out
}
