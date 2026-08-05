package metadata

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// CustomizerOpts builds options for finalize customizers.
func CustomizerOpts(prepare *types.PrepareOutput, convert *types.ConverterOutput) map[string]string {
	opts := map[string]string{"os_type": prepare.Inspect.Type}
	if prepare.Options.Hostname != "" {
		opts["hostname"] = prepare.Options.Hostname
	}
	if prepare.Options.Timezone != "" {
		opts["timezone"] = prepare.Options.Timezone
	}
	scriptsDir := prepare.Options.DynamicScriptsDir
	if scriptsDir == "" {
		scriptsDir = "/mnt/dynamic_scripts"
	}
	opts["scripts_dir"] = scriptsDir
	if convert != nil && convert.SELinuxRelabeled {
		opts["selinux_relabeled"] = "true"
	}
	return opts
}

// WriteTargetMeta merges converter warnings and block errors into TargetMeta
// warnings, then writes the JSON file. Block errors are surfaced as warnings
// so that downstream consumers (e.g. the migration controller) can detect
// partial failures such as a failed initramfs rebuild.
func WriteTargetMeta(path string, meta *types.TargetMeta, convert *types.ConverterOutput) error {
	if len(convert.Warnings) > 0 {
		meta.Warnings = append(meta.Warnings, convert.Warnings...)
	}
	for _, e := range convert.Errors {
		meta.Warnings = append(meta.Warnings, fmt.Sprintf("[%s] %s", e.Block, e.Message))
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}
	return nil
}
