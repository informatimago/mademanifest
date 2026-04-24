package httpservice

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/engine"
)

type Processor func(bodyReader io.Reader, canonPaths canon.Paths) ([]byte, error)

type Handler struct {
	CanonPaths canon.Paths
	Process    Processor
}

func New(canonPaths canon.Paths) Handler {
	return Handler{
		CanonPaths: canonPaths,
		Process: func(bodyReader io.Reader, canonPaths canon.Paths) ([]byte, error) {
			output, err := engine.Run(bodyReader, canonPaths)
			if err != nil {
				return nil, err
			}
			return engine.Render(output, false)
		},
	}
}

func (h Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/version", h.handleVersion)
	mux.HandleFunc("/manifest", h.handleManifest)
}

func (h Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleVersion returns the compiled-in pinned versions as JSON.
// This is the Phase 1 deliverable; later phases embed the same
// VersionInfo into every Trinity response envelope's metadata block.
func (h Handler) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, canon.Versions())
}

func (h Handler) handleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	defer r.Body.Close()

	const maxBodyBytes = 10 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("manifest handler panic: %v", recovered)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal processing error"})
		}
	}()

	output, err := h.Process(r.Body, h.CanonPaths)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(output); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
	}
}
