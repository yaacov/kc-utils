//go:build unix

package defaulttrimmer

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/finalize/fstrim"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type DefaultTrimmer struct{}

func init() {
	fstrim.Trimmers.Register("default", &DefaultTrimmer{})
}

func (t *DefaultTrimmer) Trim(mountpoint string) error {
	g := guest.Active()
	if g == nil {
		return fmt.Errorf("no active guest handle")
	}
	return g.FSTrim(mountpoint)
}
