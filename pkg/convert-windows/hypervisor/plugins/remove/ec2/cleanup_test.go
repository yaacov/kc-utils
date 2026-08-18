//go:build unix

package ec2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresent(t *testing.T) {
	guestRoot := t.TempDir()
	amazonDir := filepath.Join(guestRoot, "Program Files", "Amazon")
	if err := os.MkdirAll(amazonDir, 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if !u.Detect(guestRoot, nil, nil) {
		t.Error("Detect returned false, want true when Amazon directory exists")
	}
}

func TestDetectAbsent(t *testing.T) {
	guestRoot := t.TempDir()

	u := &Cleanup{}
	if u.Detect(guestRoot, nil, nil) {
		t.Error("Detect returned true, want false when Amazon directory is absent")
	}
}

func TestRemove(t *testing.T) {
	guestRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(guestRoot, "Program Files", "Amazon"), 0o755); err != nil {
		t.Fatal(err)
	}

	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		t.Fatal(err)
	}
	xenFiles := []string{"xenvbd.sys", "xennet.sys"}
	for _, f := range xenFiles {
		if err := os.WriteFile(filepath.Join(driversDir, f), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(driversDir, "disk.sys"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	tasksDir := filepath.Join(guestRoot, "Windows", "System32", "Tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, task := range []string{
		"Amazon Ec2 Launch - Instance Integrity",
		"Amazon Ec2 Launch - Sysprep",
	} {
		if err := os.WriteFile(filepath.Join(tasksDir, task), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	h := mock.NewMockHive()
	h.SetDWORD(`Select`, "Current", 2)
	ccs := "ControlSet002"
	for _, svc := range []string{"AmazonSSMAgent", "Xennet", "xenfilt"} {
		h.CreateKey(ccs + `\Services\` + svc)
		h.SetDWORD(ccs+`\Services\`+svc, "Start", 2)
	}
	for _, guid := range []string{systemClassGUID, hdcClassGUID} {
		classPath := ccs + `\Control\Class\` + guid
		h.CreateKey(classPath)
		h.SetMultiString(classPath, "UpperFilters", []string{"PartMgr", "XENFILT"})
	}

	u := &Cleanup{}
	if err := u.Remove(guestRoot, h, nil); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	for _, svc := range []string{"AmazonSSMAgent", "Xennet", "xenfilt"} {
		start, err := h.GetDWORD(ccs+`\Services\`+svc, "Start")
		if err != nil {
			t.Fatalf("GetDWORD for %s: %v", svc, err)
		}
		if start != 4 {
			t.Errorf("%s Start = %d, want 4 (disabled)", svc, start)
		}
	}

	for _, guid := range []string{systemClassGUID, hdcClassGUID} {
		classPath := ccs + `\Control\Class\` + guid
		filters, err := h.GetMultiString(classPath, "UpperFilters")
		if err != nil {
			t.Fatalf("GetMultiString %s: %v", guid, err)
		}
		for _, f := range filters {
			if strings.EqualFold(f, "XENFILT") {
				t.Errorf("XENFILT still in UpperFilters for %s", guid)
			}
		}
	}

	for _, f := range xenFiles {
		if _, err := os.Stat(filepath.Join(driversDir, f)); !os.IsNotExist(err) {
			t.Errorf("xen driver %s still exists after Remove", f)
		}
	}
	if _, err := os.Stat(filepath.Join(driversDir, "disk.sys")); err != nil {
		t.Error("non-xen driver disk.sys was incorrectly removed")
	}

	for _, task := range []string{
		"Amazon Ec2 Launch - Instance Integrity",
		"Amazon Ec2 Launch - Sysprep",
	} {
		if _, err := os.Stat(filepath.Join(tasksDir, task)); !os.IsNotExist(err) {
			t.Errorf("scheduled task %q still exists after Remove", task)
		}
	}
}
