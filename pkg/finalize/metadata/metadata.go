package metadata

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// CustomizerOpts builds options for finalize customizers.
func CustomizerOpts(pipeline *types.PipelineData) map[string]string {
	opts := map[string]string{"os_type": pipeline.Prepare.Inspect.Type}
	if pipeline.Prepare.Options.Hostname != "" {
		opts["hostname"] = pipeline.Prepare.Options.Hostname
	}
	if pipeline.Prepare.Options.Timezone != "" {
		opts["timezone"] = pipeline.Prepare.Options.Timezone
	}
	scriptsDir := pipeline.Prepare.Options.DynamicScriptsDir
	if scriptsDir == "" {
		scriptsDir = "/mnt/dynamic_scripts"
	}
	opts["scripts_dir"] = scriptsDir
	if pipeline.Convert != nil && pipeline.Convert.SELinuxRelabeled {
		opts["selinux_relabeled"] = "true"
	}
	return opts
}

// WriteTargetMeta merges converter warnings and block errors into TargetMeta
// warnings, then writes the full PipelineData JSON. Block errors are surfaced
// as warnings so that downstream consumers (e.g. the migration controller) can
// detect partial failures such as a failed initramfs rebuild.
func WriteTargetMeta(path string, pipeline *types.PipelineData) error {
	convert := pipeline.Convert
	if convert != nil && (len(convert.Warnings) > 0 || len(convert.Errors) > 0) {
		// Target may be nil (e.g. an early-failing convert). Allocate it so the
		// warnings are still recorded and marshaled, rather than dereferencing a
		// nil pointer.
		if pipeline.Target == nil {
			pipeline.Target = &types.TargetMeta{}
		}
		meta := pipeline.Target
		if len(convert.Warnings) > 0 {
			meta.Warnings = append(meta.Warnings, convert.Warnings...)
		}
		for _, e := range convert.Errors {
			meta.Warnings = append(meta.Warnings, fmt.Sprintf("[%s] %s", e.Block, e.Message))
		}
	}
	data, err := json.MarshalIndent(pipeline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}
	return nil
}
