package def

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/converter"
)

type DefaultSelector struct{}

func init() {
	converter.Selectors.Register("default", &DefaultSelector{})
}

func (d *DefaultSelector) Select(inspect *types.InspectData) (string, error) {
	switch inspect.Type {
	case "linux":
		return "kc-convert-linux", nil
	case "windows":
		return "kc-convert-windows", nil
	default:
		return "", fmt.Errorf("unsupported OS type: %s", inspect.Type)
	}
}
