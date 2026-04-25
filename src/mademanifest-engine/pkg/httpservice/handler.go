package httpservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/hd/structure"
	"mademanifest-engine/pkg/trinity/astro"
	"mademanifest-engine/pkg/trinity/genekeys"
	"mademanifest-engine/pkg/trinity/hd"
	"mademanifest-engine/pkg/trinity/input"
	"mademanifest-engine/pkg/trinity/output"
)

// MaxRequestBodyBytes caps the size of the request body that
// /manifest accepts.  Phase 10 pins this at 1 MiB, well above any
// realistic Trinity input (a canonical payload is ~150 bytes), but
// far below sizes that would burden the engine's JSON decoder or
// expose us to memory-pressure DoS via large bodies.
//
// Bodies that exceed this cap are rejected with HTTP 413 and an
// unsupported_input envelope per the Phase 10 plan deliverable.
const MaxRequestBodyBytes = 1 << 20

// Processor consumes the request body and returns the bytes to send
// back, the HTTP status code to attach, and an error.  A non-nil
// error is treated as a server-side execution_failure: the handler
// wraps it in a Trinity error envelope and returns HTTP 500.
//
// When err is nil, body must be a complete JSON document (the Trinity
// envelope) and status the canonical HTTP code chosen by
// output.StatusCodeForErrorType for rejections, or 200 for the
// success path.
//
// Phase 12 retired the canonPaths argument: the trinity processor
// reads its canonical constants directly from pkg/canon, never
// from a JSON file on disk.
type Processor func(bodyReader io.Reader) (body []byte, status int, err error)

// Handler is the Trinity HTTP service.  Phase 12 retired the
// CanonPaths field that originally piped legacy canon JSON paths
// through the handler; the trinity processor now reads canon
// directly from pkg/canon.
//
// DevCORS is OFF by default and must remain so in production.
// See the docstring on withCORS for the threat model and the dev
// workflow that enables it.
type Handler struct {
	Process Processor
	DevCORS bool
}

// New wires the default Trinity processor.  CORS is OFF; flip
// Handler.DevCORS = true (or pass --dev-cors / TRINITY_DEV_CORS=1
// to cmd/httpserver) to enable it for the local browser test
// client.
func New() Handler {
	return Handler{
		Process: trinityProcess,
	}
}

func (h Handler) Register(mux *http.ServeMux) {
	healthz := http.HandlerFunc(h.handleHealth)
	version := http.HandlerFunc(h.handleVersion)
	manifest := http.HandlerFunc(h.handleManifest)
	if h.DevCORS {
		healthz = withCORS(healthz)
		version = withCORS(version)
		manifest = withCORS(manifest)
	}
	mux.Handle("/healthz", healthz)
	mux.Handle("/version", version)
	mux.Handle("/manifest", manifest)
}

// withCORS wraps an http.HandlerFunc with permissive CORS headers
// and short-circuits OPTIONS preflight requests with HTTP 204.
//
// CORS is OFF by default and is only enabled when
// Handler.DevCORS is true (toggled by the cmd/httpserver
// `--dev-cors` flag or the TRINITY_DEV_CORS=1 environment
// variable).  Production deployments must leave it OFF: even
// though the engine is stateless and has no PII or auth, a
// wildcard CORS posture would let any web origin call the engine
// on a user's behalf, adding gratuitous attack surface (e.g.
// CSRF-style amplification of computational load) for zero
// benefit in production where requests arrive through an
// authenticated ingress controller.
//
// When ON the middleware emits Access-Control-Allow-Origin: *,
// answers OPTIONS preflight with HTTP 204, and is enabled only
// for the local-dev workflow described in
// src/scripts/k8s-local-up.sh and src/scripts/client.html.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (h Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// VersionResponse is the wire shape of GET /version.  It embeds the
// canon.VersionInfo block (engine_version, canon_version, etc.) and
// adds a deployment-resolved diagnostic field for the Swiss
// Ephemeris data path.  Phase 9 surfaces ephe_path_resolved here so
// operators can confirm at runtime exactly which ephemeris bundle
// the engine has loaded.  The field never appears in the trinity
// success/error response metadata block – that is reserved for
// canon version pins (trinity.org §"Metadata" lines 451-462).
type VersionResponse struct {
	canon.VersionInfo
	EphePathResolved string `json:"ephe_path_resolved"`
}

// handleVersion returns the compiled-in pinned versions as JSON,
// plus the resolved ephemeris path (Phase 9 diagnostic).
func (h Handler) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, VersionResponse{
		VersionInfo:      canon.Versions(),
		EphePathResolved: ephemeris.ResolvedEphePath(),
	})
}

func (h Handler) handleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	defer r.Body.Close()

	// Phase 10: Content-Type enforcement.  A POST /manifest with the
	// wrong (or missing) Content-Type is rejected with HTTP 415 and
	// a Trinity error envelope of type invalid_input, before the
	// body is even read.  This protects clients from accidentally
	// posting form-encoded or text/plain payloads and from us
	// silently parsing them as JSON.
	if msg, ok := requireJSONContentType(r); !ok {
		env := output.NewError(output.ErrorInvalidInput, msg)
		writeJSON(w, http.StatusUnsupportedMediaType, env)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)

	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("manifest handler panic: %v", recovered)
			env := output.NewError(output.ErrorExecutionFailure,
				"internal processing error")
			writeJSON(w, http.StatusInternalServerError, env)
		}
	}()

	body, status, err := h.Process(r.Body)
	if err != nil {
		// Phase 10: distinguish oversize-body errors from generic
		// execution failures.  http.MaxBytesReader returns
		// *http.MaxBytesError once the cap is hit; we surface that
		// as the canonical 413 + unsupported_input envelope before
		// falling through to the generic 500 path.
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			env := output.NewError(output.ErrorUnsupportedInput,
				fmt.Sprintf("request body exceeds %d-byte limit",
					MaxRequestBodyBytes))
			writeJSON(w, http.StatusRequestEntityTooLarge, env)
			return
		}
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

// requireJSONContentType inspects the request's Content-Type header
// and returns ("", true) when it is application/json (with optional
// parameters such as charset=utf-8), or (msg, false) when it is
// missing, malformed, or not application/json.
//
// The string return is suitable for the error envelope's message
// field; the bool is the gate for the caller's branching.
func requireJSONContentType(r *http.Request) (string, bool) {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return "Content-Type header is required; expected application/json", false
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return fmt.Sprintf("Content-Type %q is not parseable: %s", ct, err.Error()), false
	}
	if mediaType != "application/json" {
		return fmt.Sprintf(
			"Content-Type %q is not supported; expected application/json",
			mediaType), false
	}
	return "", true
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
func trinityProcess(bodyReader io.Reader) ([]byte, int, error) {
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

	// Phase 7: structural derivations from the combined activation
	// set – channels, centers, definition, type, authority, profile,
	// incarnation_cross.  Replaces the placeholder reflector / lunar /
	// "1/1" values seeded by NewPlaceholderSuccess.
	struc, err := structure.Compute(personality, design)
	if err != nil {
		return nil, 0, fmt.Errorf("compute structure: %w", err)
	}
	env.HumanDesign.Channels = struc.Channels
	env.HumanDesign.Centers = struc.Centers
	env.HumanDesign.Definition = struc.Definition
	env.HumanDesign.Type = struc.Type
	env.HumanDesign.Authority = struc.Authority
	env.HumanDesign.Profile = struc.Profile
	env.HumanDesign.IncarnationCross = struc.IncarnationCross

	// Phase 8: Gene Keys block.  Derived directly from the four
	// HD pillar activations (personality sun + earth, design sun +
	// earth) — no astronomical computation, no node policy.
	gk, err := genekeys.Compute(personality, design)
	if err != nil {
		return nil, 0, fmt.Errorf("compute gene keys: %w", err)
	}
	env.GeneKeys = gk

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
