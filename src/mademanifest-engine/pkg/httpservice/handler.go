package httpservice

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/trinity/astro"
	"mademanifest-engine/pkg/trinity/hd"
	"mademanifest-engine/pkg/trinity/input"
	"mademanifest-engine/pkg/trinity/output"
)

// Processor consumes the request body and returns the bytes to send
// back, the HTTP status code to attach, and an error.  A non-nil
// error is treated as a server-side execution_failure: the handler
// wraps it in a Trinity error envelope and returns HTTP 500.
//
// When err is nil, body must be a complete JSON document (the Trinity
// envelope) and status the canonical HTTP code chosen by
// output.StatusCodeForErrorType for rejections, or 200 for the
// (Phase 3+) success path.
type Processor func(bodyReader io.Reader, canonPaths canon.Paths) (body []byte, status int, err error)

type Handler struct {
	CanonPaths canon.Paths
	Process    Processor
}

// New wires the default Trinity processor.  Phase 2 implements the
// validator and the rejection path; the success branch returns
// execution_failure with a placeholder message until later phases
// land the calculation pipeline behind the same surface.
func New(canonPaths canon.Paths) Handler {
	return Handler{
		CanonPaths: canonPaths,
		Process:    trinityProcess,
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
// This is the Phase 1 deliverable; later phases embed the matching
// metadata block into every Trinity response envelope.
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
			env := output.NewError(output.ErrorExecutionFailure,
				"internal processing error")
			writeJSON(w, http.StatusInternalServerError, env)
		}
	}()

	body, status, err := h.Process(r.Body, h.CanonPaths)
	if err != nil {
		log.Printf("manifest processor error: %v", err)
		env := output.NewError(output.ErrorExecutionFailure, err.Error())
		writeJSON(w, http.StatusInternalServerError, env)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

// trinityProcess is the default manifest processor.  It reads the
// request body, runs it through the Trinity input validator, and
// returns either:
//
//   * a Trinity error envelope with the validator's classification
//     plus the matching HTTP status code (rejection path), or
//   * a Trinity success envelope with HTTP 200 (success path) –
//     since Phase 3, populated by NewPlaceholderSuccess until
//     Phases 4-8 wire the calculation pipeline behind the same
//     surface.
//
// Phase 3 establishes the envelope shape and the status-code policy.
// Phase 4-8 incrementally fill the placeholder calculation
// sub-fields with real values; each fill is a strict superset of
// the previous behaviour and does not change the wire shape.
func trinityProcess(bodyReader io.Reader, _ canon.Paths) ([]byte, int, error) {
	raw, err := io.ReadAll(bodyReader)
	if err != nil {
		// I/O failures (truncated upload, MaxBytesReader trip)
		// surface as execution_failure: the validator never ran.
		return nil, 0, fmt.Errorf("read request body: %w", err)
	}

	payload, rej := input.Validate(raw)
	if rej != nil {
		env := output.NewError(string(rej.Type), rej.Message)
		body, encErr := json.Marshal(env)
		if encErr != nil {
			return nil, 0, fmt.Errorf("marshal error envelope: %w", encErr)
		}
		return body, output.StatusCodeForErrorType(string(rej.Type)), nil
	}

	// Validation succeeded.  Build the canonical success envelope,
	// then replace each placeholder section with computed values
	// as the calculation phases come online.  Phase 4 wires the
	// astrology section; Phase 5 wires human_design.system.design_time_utc;
	// Phase 6 wires the personality + design activation arrays;
	// Phases 7-8 will fill the remaining placeholders (channels,
	// centers, structural derivations, gene keys).
	env := output.NewPlaceholderSuccess(payload)

	astroSection, err := astro.ComputeAstrology(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("compute astrology: %w", err)
	}
	env.Astrology = astroSection

	designTime, err := hd.ComputeDesignTime(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("compute design time: %w", err)
	}
	env.HumanDesign.System.DesignTimeUTC = output.DesignTime(designTime)

	// Phase 6: personality + design activation arrays.  We re-use
	// the already-computed design moment (designTime) to avoid
	// running the bisection solver a second time.  Personality is
	// the snapshot at birth.
	designJD := hd.DesignJDFromTime(designTime)
	personality, design, err := hd.ComputeActivations(payload, designJD)
	if err != nil {
		return nil, 0, fmt.Errorf("compute activations: %w", err)
	}
	env.HumanDesign.PersonalityActivations = personality
	env.HumanDesign.DesignActivations = design

	body, encErr := json.Marshal(env)
	if encErr != nil {
		return nil, 0, fmt.Errorf("marshal success envelope: %w", encErr)
	}
	return body, http.StatusOK, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
	}
}
