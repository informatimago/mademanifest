// Package input implements the Trinity Engine v1 input contract:
// the canonical payload DTO (Payload) and a strict boundary
// validator (Validate) that rejects any input outside the canon's
// in-scope shape with one of three classifications:
//
//   incomplete_input   – a required field is missing.
//   invalid_input      – a present field has the wrong type, format,
//                        range, or shape (numeric-as-string included).
//   unsupported_input  – the input is structurally valid but outside
//                        Trinity v1 supported scope (per A5).
//
// Canon source: specifications/trinity/trinity.org §"Input Contract"
// lines 155-220, plus Documents 04 and 09.  The "no silent repair"
// rule from line 212 means the validator must never coerce, infer,
// truncate, default, or alias-resolve.
package input

// Payload is the canonical Trinity v1 input.  All five fields are
// required; v1 has no optional canonical input fields.
//
// JSON tag values are part of the input contract and must not change
// without an InputSchemaVersion bump.
type Payload struct {
	BirthDate string  `json:"birth_date"` // YYYY-MM-DD (Gregorian)
	BirthTime string  `json:"birth_time"` // HH:MM (24-hour, minute precision)
	Timezone  string  `json:"timezone"`   // IANA Area/Location identifier
	Latitude  float64 `json:"latitude"`   // decimal degrees, [-90.0, 90.0]
	Longitude float64 `json:"longitude"`  // decimal degrees, [-180.0, 180.0]
}
