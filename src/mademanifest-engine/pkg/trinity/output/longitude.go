package output

import "strconv"

// Longitude is the JSON marshaling type for any field that
// trinity.org §"Formatting Rules" line 581 requires to be a "numeric
// JSON value, rounded to 6 decimal places".  In Go we keep the value
// as float64 so callers can do arithmetic, and customise MarshalJSON
// to emit exactly six fractional digits.
//
// The same type is used for ecliptic longitudes (astrology objects,
// house cusps, ascendant, midheaven) and for geographic coordinates
// inside InputEcho, on the assumption that the canon's "longitude"
// rule applies to both.  The canon does not distinguish them; if a
// canon revision pins different precisions for geographic vs
// ecliptic longitudes, only this file changes.
type Longitude float64

// MarshalJSON emits the value as a JSON number with exactly six
// fractional digits.  strconv.FormatFloat('f', 6, 64) round-trips
// cleanly through encoding/json (a JSON number is parsed back to a
// float64).
func (l Longitude) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(l), 'f', 6, 64)), nil
}
