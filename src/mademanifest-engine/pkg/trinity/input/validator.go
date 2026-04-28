package input

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// requiredFields lists the canonical input fields in the order the
// validator reports them when several are missing.  Order matches
// trinity.org §"Canonical Payload" lines 158-165.
var requiredFields = []string{
	"birth_date",
	"birth_time",
	"timezone",
	"latitude",
	"longitude",
}

var (
	// dateRE strictly accepts YYYY-MM-DD with four-digit year.  Rules
	// like "1990-13-01" still pass the regex but get rejected by the
	// time.Parse Gregorian check below.
	dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	// timeMinuteRE is the canonical HH:MM, 24-hour, minute precision
	// (trinity.org line 161).  HH=00..23, MM=00..59.
	timeMinuteRE = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

	// timeWithSecondsRE catches HH:MM:SS or HH:MM:SS.fraction.
	// A5 (RESOLVED, Document 12 D24 + Document 04): structurally
	// well-formed but outside v1 supported scope ⇒ unsupported_input.
	// Sub-minute precision falls in this bucket.
	timeWithSecondsRE = regexp.MustCompile(`^[0-2]\d:[0-5]\d:\d`)

	// ianaCanonicalShapeRE is a coarse shape check requiring at
	// least one slash.  It rules out abbreviations like "CET", "EST"
	// (per trinity.org line 110-111: "rejects abbreviations").
	ianaCanonicalShapeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(?:/[A-Za-z0-9_+\-]+)+$`)
)

// Known IANA *link* prefixes rejected per A6 (RESOLVED, Document 12
// D24): canonical IANA Area/Location identifiers only.  Aliases and
// link names are not accepted unless explicitly introduced by a
// later canon revision.  This conservative prefix list will be
// replaced with a curated zone.tab in a follow-up.
var ianaLinkPrefixes = []string{
	"US/",       // US/Eastern, US/Pacific, ... -> link to America/*
	"SystemV/",  // legacy compatibility links
	"Brazil/",   // link to America/Bahia / Sao_Paulo / etc.
	"Canada/",   // link to America/Toronto / Vancouver / etc.
	"Chile/",    // link to America/Santiago / Punta_Arenas
	"Mexico/",   // link to America/Mexico_City etc.
	"Etc/GMT+",  // numbered GMT offsets
	"Etc/GMT-",  // numbered GMT offsets
}

// Validate parses raw JSON, applies every Trinity v1 input rule
// (presence, strict numeric typing, format, range, IANA canonical
// zone, no unknown fields), and returns either a fully populated
// Payload or a non-nil Rejection.  The two return values are
// mutually exclusive: an error implies the Payload is the zero
// value, and vice versa.
func Validate(raw []byte) (Payload, *Rejection) {
	// First pass: decode into a map of raw messages so we can
	// distinguish "missing" (key absent) from "wrong type" (key
	// present, value wrong shape) and from "unknown field" (key
	// present, not in our schema).  json.Decoder.DisallowUnknownFields
	// only catches unknown fields when decoding into a struct; we
	// implement the same check manually because our second pass
	// reads the values out one by one.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // preserve numeric precision for range checks
	var m map[string]json.RawMessage
	if err := dec.Decode(&m); err != nil {
		return Payload{}, classifyDecodeError(err)
	}
	// The decoder allows trailing data after the first JSON value,
	// which Trinity does not – the canonical payload is exactly one
	// object.
	if dec.More() {
		return Payload{}, rej(RejectInvalid, "",
			"payload must be a single JSON object, found trailing data")
	}
	if m == nil {
		return Payload{}, rej(RejectInvalid, "",
			"payload must be a JSON object, got null")
	}

	// Reject any unknown fields up front – the canon forbids extra
	// fields silently affecting results, so we surface them as
	// invalid_input rather than swallowing them.
	known := make(map[string]bool, len(requiredFields))
	for _, f := range requiredFields {
		known[f] = true
	}
	for k := range m {
		if !known[k] {
			return Payload{}, rej(RejectInvalid, k,
				"unknown field; canonical payload has exactly "+
					strings.Join(requiredFields, ", "))
		}
	}

	// Required-presence check.  We report the first missing field in
	// canonical order so the client sees a deterministic message.
	for _, f := range requiredFields {
		if _, ok := m[f]; !ok {
			return Payload{}, rej(RejectIncomplete, f, "required field is missing")
		}
	}

	// Type + content checks per field.
	var p Payload
	if r := decodeString(m["birth_date"], "birth_date", &p.BirthDate); r != nil {
		return Payload{}, r
	}
	if r := validateBirthDate(p.BirthDate); r != nil {
		return Payload{}, r
	}
	if r := decodeString(m["birth_time"], "birth_time", &p.BirthTime); r != nil {
		return Payload{}, r
	}
	if r := validateBirthTime(p.BirthTime); r != nil {
		return Payload{}, r
	}
	if r := decodeString(m["timezone"], "timezone", &p.Timezone); r != nil {
		return Payload{}, r
	}
	if r := validateTimezone(p.Timezone); r != nil {
		return Payload{}, r
	}
	if r := decodeNumber(m["latitude"], "latitude", -90.0, 90.0, &p.Latitude); r != nil {
		return Payload{}, r
	}
	if r := decodeNumber(m["longitude"], "longitude", -180.0, 180.0, &p.Longitude); r != nil {
		return Payload{}, r
	}
	return p, nil
}

// classifyDecodeError maps a json.Decode failure on the outer object
// to a Rejection.  It does not try to be exhaustive – the goal is to
// avoid leaking the standard library's error text directly.
func classifyDecodeError(err error) *Rejection {
	if err == nil {
		return nil
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return rej(RejectInvalid, "",
			fmt.Sprintf("malformed JSON at byte %d", syntaxErr.Offset))
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return rej(RejectInvalid, "",
			"payload must be a JSON object")
	}
	return rej(RejectInvalid, "", "JSON decode error: "+err.Error())
}

// decodeString reads a JSON string value into dst.  Anything that
// is not a JSON string – number, boolean, null, object, array – is
// invalid_input.
func decodeString(raw json.RawMessage, field string, dst *string) *Rejection {
	if !looksLikeJSONString(raw) {
		return rej(RejectInvalid, field,
			"must be a JSON string")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return rej(RejectInvalid, field, "invalid JSON string: "+err.Error())
	}
	return nil
}

// decodeNumber reads a JSON number into dst with strict typing.
// Strings that happen to contain digits are rejected per
// trinity.org line 187-189 ("strict numeric typing").
func decodeNumber(raw json.RawMessage, field string, lo, hi float64, dst *float64) *Rejection {
	if looksLikeJSONString(raw) {
		return rej(RejectInvalid, field,
			"numeric field supplied as string; strict numeric typing required")
	}
	// json.Number preserves precision; we convert with strconv via
	// json.Number.Float64.
	var num json.Number
	if err := json.Unmarshal(raw, &num); err != nil {
		return rej(RejectInvalid, field, "must be a numeric JSON value")
	}
	v, err := num.Float64()
	if err != nil {
		return rej(RejectInvalid, field, "value is not a finite number")
	}
	if v < lo || v > hi {
		return rej(RejectInvalid, field,
			fmt.Sprintf("out of range; expected [%g, %g]", lo, hi))
	}
	*dst = v
	return nil
}

// looksLikeJSONString returns true when the raw token starts with a
// double quote.  json.RawMessage preserves the original bytes,
// stripped of leading whitespace by the decoder, so a single byte
// suffices.
func looksLikeJSONString(raw json.RawMessage) bool {
	for _, b := range raw {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '"':
			return true
		default:
			return false
		}
	}
	return false
}

// validateBirthDate enforces strict YYYY-MM-DD plus a Gregorian
// validity round-trip.  Round-trip parse-then-format catches all
// "real-looking" but impossible dates (1990-02-30, 2025-13-01) which
// time.Parse otherwise normalises silently.
func validateBirthDate(s string) *Rejection {
	if !dateRE.MatchString(s) {
		return rej(RejectInvalid, "birth_date",
			"must match YYYY-MM-DD (e.g. 1990-04-09)")
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		return rej(RejectInvalid, "birth_date",
			"not a valid Gregorian date: "+err.Error())
	}
	if parsed.Format("2006-01-02") != s {
		return rej(RejectInvalid, "birth_date",
			"date components are not a valid Gregorian calendar date")
	}
	return nil
}

// validateBirthTime distinguishes HH:MM (accept), HH:MM:SS or
// finer-grained values (unsupported_input – A5 RESOLVED), and
// everything else (invalid_input).
func validateBirthTime(s string) *Rejection {
	if timeMinuteRE.MatchString(s) {
		return nil
	}
	if timeWithSecondsRE.MatchString(s) {
		return rej(RejectUnsupported, "birth_time",
			"sub-minute precision is outside Trinity v1 scope; canonical format is HH:MM")
	}
	return rej(RejectInvalid, "birth_time",
		"must match HH:MM, 24-hour, minute precision (e.g. 18:04)")
}

// validateTimezone rejects abbreviations (no slash), known link-name
// prefixes (A6 RESOLVED – canonical IANA Area/Location only), and
// identifiers that time.LoadLocation cannot resolve against the
// embedded tzdb.
func validateTimezone(s string) *Rejection {
	if s == "" {
		return rej(RejectInvalid, "timezone", "must be a non-empty IANA identifier")
	}
	if !ianaCanonicalShapeRE.MatchString(s) {
		return rej(RejectInvalid, "timezone",
			"must be an IANA Area/Location identifier (abbreviations like CET are rejected)")
	}
	for _, p := range ianaLinkPrefixes {
		if strings.HasPrefix(s, p) {
			return rej(RejectInvalid, "timezone",
				"IANA link-name aliases are rejected; use the canonical Area/Location form")
		}
	}
	if _, err := time.LoadLocation(s); err != nil {
		return rej(RejectInvalid, "timezone",
			"unknown IANA identifier: "+err.Error())
	}
	return nil
}
