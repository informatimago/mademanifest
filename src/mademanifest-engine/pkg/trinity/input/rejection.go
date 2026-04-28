package input

import "fmt"

// RejectionType is the classification carried by a Rejection.  The
// string values match the canonical error_type literals from
// trinity.org §"Error Output" – callers can hand the value directly
// to output.NewError() without translation.
type RejectionType string

const (
	RejectIncomplete  RejectionType = "incomplete_input"
	RejectInvalid     RejectionType = "invalid_input"
	RejectUnsupported RejectionType = "unsupported_input"
)

// Rejection is the structured failure value returned by Validate.
//
// Field names the offending JSON key (or the empty string when the
// rejection applies to the payload as a whole, e.g. malformed JSON
// or non-object root).  Message is a developer-facing diagnostic
// string; callers may pass it through to the response envelope's
// "message" field.  A4 (RESOLVED, Document 12 D23): the Message
// text is informational – fixtures match Type (and Field) only.
type Rejection struct {
	Type    RejectionType
	Field   string
	Message string
}

// Error makes Rejection a Go error so it composes with idiomatic
// `if err := Validate(...); err != nil` patterns.
func (r *Rejection) Error() string {
	if r.Field == "" {
		return fmt.Sprintf("%s: %s", r.Type, r.Message)
	}
	return fmt.Sprintf("%s in field %q: %s", r.Type, r.Field, r.Message)
}

// rej is a small constructor used internally so the validator stays
// readable.
func rej(t RejectionType, field, msg string) *Rejection {
	return &Rejection{Type: t, Field: field, Message: msg}
}
