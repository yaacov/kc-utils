//go:build unix

package vmware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresent(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\VMware Tools`)

	u := &Remove{}
	if !u.Detect("/fake/root", nil, h) {
		t.Error("Detect returned false, want true when VMware Tools key exists")
	}
}

func TestDetectViaDirectory(t *testing.T) {
	guestRoot := t.TempDir()
	toolsDir := filepath.Join(guestRoot, "Program Files", "VMware", "VMware Tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()

	u := &Remove{}
	if !u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned false, want true when VMware Tools directory exists (no registry key)")
	}
}

func TestDetectAbsent(t *testing.T) {
	guestRoot := t.TempDir()
	h := mock.NewMockHive()

	u := &Remove{}
	if u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned true, want false when neither directory nor registry key exists")
	}
}

func TestRemove(t *testing.T) {
	guestRoot := t.TempDir()
	toolsDir := filepath.Join(guestRoot, "Program Files", "VMware", "VMware Tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatalf("failed to create tools dir: %v", err)
	}
	markerFile := filepath.Join(toolsDir, "vmtoolsd.exe")
	if err := os.WriteFile(markerFile, []byte("fake"), 0o644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	h := mock.NewMockHive()
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\VMware Tools`)

	u := &Remove{}
	err := u.Remove(guestRoot, nil, h)
	if err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if _, err := os.Stat(toolsDir); !os.IsNotExist(err) {
		t.Errorf("VMware Tools directory still exists after Remove")
	}
	if h.KeyExists(`Microsoft\Windows\CurrentVersion\Uninstall\VMware Tools`) {
		t.Error("VMware Tools registry key still exists after Remove")
	}
}

func TestRemoveMSIProducts(t *testing.T) {
	h := mock.NewMockHive()

	// Simulate an MSI product entry for VMware Tools (encoded GUID).
	encodedGUID := "87654321432143214321CBA987654321"
	productKey := installerProducts + `\` + encodedGUID
	h.CreateKey(productKey)
	h.SetString(productKey, "ProductName", "VMware Tools")

	featureKey := installerFeatures + `\` + encodedGUID
	h.CreateKey(featureKey)

	userDataKey := userDataProducts + `\` + encodedGUID + `\InstallProperties`
	h.CreateKey(userDataKey)
	h.SetString(userDataKey, "DisplayName", "VMware Tools")

	guids := removeMSIProducts(h)

	if len(guids) != 1 || guids[0] != encodedGUID {
		t.Fatalf("expected [%s], got %v", encodedGUID, guids)
	}
	if h.KeyExists(productKey) {
		t.Error("MSI product key still exists after removal")
	}
	if h.KeyExists(featureKey) {
		t.Error("MSI feature key still exists after removal")
	}
	if h.KeyExists(userDataProducts + `\` + encodedGUID) {
		t.Error("MSI user-data key still exists after removal")
	}
}

func TestRemoveScheduledTasks(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(taskCacheTree)

	removeScheduledTasks(h)

	if h.KeyExists(taskCacheTree) {
		t.Error("VMware scheduled task key still exists after removal")
	}
}

func TestDecodeMSIGUID(t *testing.T) {
	cases := []struct {
		encoded string
		want    string
	}{
		{"87654321432143214321CBA987654321", "{12345678-1234-1234-3412-BC9A78563412}"},
		{"", ""},
		{"short", ""},
	}
	for _, tc := range cases {
		got := decodeMSIGUID(tc.encoded)
		if got != tc.want {
			t.Errorf("decodeMSIGUID(%q) = %q, want %q", tc.encoded, got, tc.want)
		}
	}
}

func TestFirstbootScriptWritten(t *testing.T) {
	guestRoot := t.TempDir()

	writeMSIUninstallFirstboot(guestRoot, []string{"87654321432143214321CBA987654321"})

	scriptPath := filepath.Join(guestRoot, "Program Files", "Guestfs", "Firstboot", "scripts", "0010-vmware-msi-uninstall.ps1")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("firstboot script not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("firstboot script is empty")
	}
}
