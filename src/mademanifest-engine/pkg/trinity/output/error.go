package output

import "net/http"

// Canonical response status strings.  Trinity pins "success" and
// "error" exactly; any other value is a canon violation.
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

// Canonical error_type values from trinity.org §"Error Output" lines
// 569-575.  These are the only allowed values for Error.Type.
const (
	ErrorInvalidInput     = "invalid_input"
	ErrorIncompleteInput  = "incomplete_input"
	ErrorUnsupportedInput = "unsupported_input"
	ErrorCanonConflict    = "canon_conflict"
	ErrorExecutionFailure = "execution_failure"
)

// Error is the nested object inside an ErrorEnvelope.  Type is one
// of the canonical error_type constants; Message is a developer-
// facing string.  A4 (error-message canon not yet pinned) means
// fixtures compare Type only, not Message, until the canon owner
// decides otherwise.
type Error struct {
	Type    string `json:"error_type"`
	Message string `json:"message"`
}

// ErrorEnvelope is the top-level Trinity error response.  Key order
// (status, metadata, error) matches trinity.org §"Error Output"
// lines 560-569.  Phase 3 will add a serializer that pins the order
// irrespective of the json encoder implementation.
type ErrorEnvelope struct {
	Status   string   `json:"status"`
	Metadata Metadata `json:"metadata"`
	Error    Error    `json:"error"`
}

// NewError constructs an ErrorEnvelope with the compiled-in
// metadata and the given error_type and message.  Callers should
// use the exported Error* constants for errType so that typos are
// caught at compile time.
func NewError(errType, message string) ErrorEnvelope {
	return ErrorEnvelope{
		Status:   StatusError,
		Metadata: CurrentMetadata(),
		Error: Error{
			Type:    errType,
			Message: message,
		},
	}
}

// StatusCodeForErrorType returns the HTTP status code associated
// with a given canonical error_type.  The mapping follows the
// Phase 3 policy pre-landed here because the Phase 2 HTTP handler
// needs it to return rejections:
//
//   invalid_input, incomplete_input  -> 400 Bad Request
//   unsupported_input                -> 422 Unprocessable Entity
//   canon_conflict, execution_failure -> 500 Internal Server Error
//
// Unknown error_type values return 500 as a safe default, which
// guarantees the engine never silently ships an unmapped rejection
// at success-status codes.
func StatusCodeForErrorType(errType string) int {
	switch errType {
	case ErrorIncompleteInput, ErrorInvalidInput:
		return http.StatusBadRequest
	case ErrorUnsupportedInput:
		return http.StatusUnprocessableEntity
	case ErrorCanonConflict, ErrorExecutionFailure:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
