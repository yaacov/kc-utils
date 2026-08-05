package v2v

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/yaacov/kc-utils/pkg/v2v/env"
)

// Warning represents a non-fatal migration issue exposed via HTTP.
type Warning struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

var (
	server   *http.Server
	warnings []Warning
	mu       sync.Mutex
)

// SetWarnings stores finalize warnings for the /warnings endpoint.
func SetWarnings(msgs []string) {
	mu.Lock()
	defer mu.Unlock()
	for _, msg := range msgs {
		warnings = append(warnings, Warning{Reason: "ConversionWarning", Message: msg})
	}
}

// AddWarning appends a warning message.
func AddWarning(reason, message string) {
	mu.Lock()
	defer mu.Unlock()
	warnings = append(warnings, Warning{Reason: reason, Message: message})
}

// Start serves Forklift-compatible HTTP endpoints on :8080.
func Start(cfg *env.Config) error {
	mux := http.NewServeMux()
	s := &serverHandler{cfg: cfg}
	mux.HandleFunc("/vm", s.vmHandler)
	mux.HandleFunc("/inspection", s.inspectionHandler)
	mux.HandleFunc("/warnings", s.warningsHandler)
	mux.HandleFunc("/shutdown", s.shutdownHandler)
	server = &http.Server{Addr: ":8080", Handler: mux}
	fmt.Println("Starting server on :8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

type serverHandler struct {
	cfg *env.Config
}

func (s *serverHandler) vmHandler(w http.ResponseWriter, r *http.Request) {
	if s.cfg.IsInPlace {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "VM YAML not supported", http.StatusInternalServerError)
}

func (s *serverHandler) inspectionHandler(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(s.cfg.InspectionOutputFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *serverHandler) warningsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if len(warnings) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	data, err := json.Marshal(warnings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *serverHandler) shutdownHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Shutdown request received. Shutting down server.")
	w.WriteHeader(http.StatusNoContent)
	if server != nil {
		_ = server.Shutdown(context.Background())
	}
}
