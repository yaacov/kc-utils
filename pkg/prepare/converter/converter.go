package converter

import (
	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// ConverterSelector selects which converter binary to run based on
// OS inspection results.
type ConverterSelector interface {
	Select(inspect *types.InspectData) (converterName string, err error)
}

// Selectors is the global registry of ConverterSelector implementations.
var Selectors = plugin.NewRegistry[string, ConverterSelector]()
