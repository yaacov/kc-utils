package xml

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestWriteInspectionXML(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/inspection.xml"
	meta := &types.TargetMeta{
		Inspect: types.InspectData{
			Type:         "linux",
			Distro:       "rhel",
			MajorVersion: 9,
			Arch:         "x86_64",
			ProductName:  "Red Hat Enterprise Linux 9",
		},
	}
	if err := WriteInspectionXML(meta, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc inspectionV2V
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.OS.Osinfo != "rhel9" {
		t.Errorf("osinfo = %q, want rhel9", doc.OS.Osinfo)
	}
}
