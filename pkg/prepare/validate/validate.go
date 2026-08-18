//go:build unix

package validate

import (
	"fmt"
	"os"
)

// Input checks required prepare inputs and creates the mount root.
func Input(disks int, mountRoot string) error {
	if disks == 0 {
		return fmt.Errorf("no disks specified")
	}
	if mountRoot == "" {
		return fmt.Errorf("mount root not specified")
	}
	if err := os.MkdirAll(mountRoot, 0o755); err != nil {
		return fmt.Errorf("creating mount root: %w", err)
	}
	return nil
}
