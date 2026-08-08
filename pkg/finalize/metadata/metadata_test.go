package metadata

import (
	"os"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestCustomizerOptsIncludesTimezone(t *testing.T) {
	pipeline := &types.PipelineData{
		Prepare: &types.PrepareOutput{
			Inspect: types.InspectData{Type: "linux"},
			Options: types.PrepareOptions{Timezone: "America/New_York"},
		},
	}
	opts := CustomizerOpts(pipeline)
	if opts["timezone"] != "America/New_York" {
		t.Errorf("timezone = %q, want America/New_York", opts["timezone"])
	}
}

func TestCustomizerOptsOmitsEmptyTimezone(t *testing.T) {
	pipeline := &types.PipelineData{
		Prepare: &types.PrepareOutput{
			Inspect: types.InspectData{Type: "linux"},
			Options: types.PrepareOptions{},
		},
	}
	opts := CustomizerOpts(pipeline)
	if _, ok := opts["timezone"]; ok {
		t.Error("timezone should not be set when empty")
	}
}

func TestCustomizerOptsIncludesHostname(t *testing.T) {
	pipeline := &types.PipelineData{
		Prepare: &types.PrepareOutput{
			Inspect: types.InspectData{Type: "windows"},
			Options: types.PrepareOptions{Hostname: "myhost"},
		},
	}
	opts := CustomizerOpts(pipeline)
	if opts["hostname"] != "myhost" {
		t.Errorf("hostname = %q, want myhost", opts["hostname"])
	}
}

func TestCustomizerOptsDefaultScriptsDir(t *testing.T) {
	pipeline := &types.PipelineData{
		Prepare: &types.PrepareOutput{
			Inspect: types.InspectData{Type: "linux"},
			Options: types.PrepareOptions{},
		},
	}
	opts := CustomizerOpts(pipeline)
	if opts["scripts_dir"] != "/mnt/dynamic_scripts" {
		t.Errorf("scripts_dir = %q, want /mnt/dynamic_scripts", opts["scripts_dir"])
	}
}

func TestCustomizerOptsCustomScriptsDir(t *testing.T) {
	pipeline := &types.PipelineData{
		Prepare: &types.PrepareOutput{
			Inspect: types.InspectData{Type: "linux"},
			Options: types.PrepareOptions{DynamicScriptsDir: "/custom/scripts"},
		},
	}
	opts := CustomizerOpts(pipeline)
	if opts["scripts_dir"] != "/custom/scripts" {
		t.Errorf("scripts_dir = %q, want /custom/scripts", opts["scripts_dir"])
	}
}

func TestCustomizerOptsSELinuxRelabeled(t *testing.T) {
	pipeline := &types.PipelineData{
		Prepare: &types.PrepareOutput{
			Inspect: types.InspectData{Type: "linux"},
		},
		Convert: &types.ConverterOutput{SELinuxRelabeled: true},
	}
	opts := CustomizerOpts(pipeline)
	if opts["selinux_relabeled"] != "true" {
		t.Errorf("selinux_relabeled = %q, want true", opts["selinux_relabeled"])
	}
}

func TestCustomizerOptsSELinuxNotRelabeled(t *testing.T) {
	pipeline := &types.PipelineData{
		Prepare: &types.PrepareOutput{
			Inspect: types.InspectData{Type: "linux"},
		},
	}
	opts := CustomizerOpts(pipeline)
	if _, ok := opts["selinux_relabeled"]; ok {
		t.Error("selinux_relabeled should not be set when convert is nil")
	}
}

func TestWriteTargetMeta(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/meta.json"
	pipeline := &types.PipelineData{
		Convert: &types.ConverterOutput{
			Warnings: []string{"test warning"},
		},
		Target: &types.TargetMeta{
			GuestCaps: types.GuestCaps{BlockBus: "virtio"},
		},
	}
	if err := WriteTargetMeta(path, pipeline); err != nil {
		t.Fatalf("WriteTargetMeta error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "virtio") {
		t.Error("output should contain block_bus")
	}
	if !strings.Contains(content, "test warning") {
		t.Error("output should contain merged warning")
	}
}

func TestWriteTargetMetaNoWarnings(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/meta.json"
	pipeline := &types.PipelineData{
		Convert: &types.ConverterOutput{},
		Target:  &types.TargetMeta{},
	}
	if err := WriteTargetMeta(path, pipeline); err != nil {
		t.Fatalf("WriteTargetMeta error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if strings.Contains(string(data), "warnings") {
		t.Error("should not include warnings key when empty")
	}
}

func TestWriteTargetMetaBlockErrors(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/meta.json"
	pipeline := &types.PipelineData{
		Convert: &types.ConverterOutput{
			Errors: []types.BlockError{
				{Block: "initramfs", Message: "all initramfs rebuild methods failed for kernel 5.14.0"},
				{Block: "uefi/grub-fallback", Message: "no EFI bootloader found"},
			},
		},
		Target: &types.TargetMeta{},
	}
	if err := WriteTargetMeta(path, pipeline); err != nil {
		t.Fatalf("WriteTargetMeta error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[initramfs]") {
		t.Error("output should contain initramfs block error as warning")
	}
	if !strings.Contains(content, "[uefi/grub-fallback]") {
		t.Error("output should contain uefi block error as warning")
	}
	if !strings.Contains(content, "warnings") {
		t.Error("output should include warnings key when block errors exist")
	}
}

func TestWriteTargetMetaBadPath(t *testing.T) {
	pipeline := &types.PipelineData{
		Convert: &types.ConverterOutput{},
		Target:  &types.TargetMeta{},
	}
	err := WriteTargetMeta("/nonexistent/dir/meta.json", pipeline)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}
