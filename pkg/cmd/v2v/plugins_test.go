//go:build unix

package v2v

// Register backend plugins so capability checks (SupportsSharedListener) resolve
// the same set of backends the production mains blank-import.
import (
	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/guestfs"
	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/qemu"
)
