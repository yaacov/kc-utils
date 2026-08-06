package v2v

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/v2v/env"
)

func TestInspectionHandler(t *testing.T) {
	dir := t.TempDir()
	inspection := filepath.Join(dir, "inspection.xml")
	if err := os.WriteFile(inspection, []byte("<v2v/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &env.Config{InspectionOutputFile: inspection, IsInPlace: true}
	s := &serverHandler{cfg: cfg}

	req := httptest.NewRequest(http.MethodGet, "/inspection", nil)
	rec := httptest.NewRecorder()
	s.inspectionHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestVMHandlerInPlace(t *testing.T) {
	cfg := &env.Config{IsInPlace: true}
	s := &serverHandler{cfg: cfg}
	rec := httptest.NewRecorder()
	s.vmHandler(rec, httptest.NewRequest(http.MethodGet, "/vm", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestWarningsHandlerEmpty(t *testing.T) {
	warnings = nil
	cfg := &env.Config{}
	s := &serverHandler{cfg: cfg}
	rec := httptest.NewRecorder()
	s.warningsHandler(rec, httptest.NewRequest(http.MethodGet, "/warnings", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}
