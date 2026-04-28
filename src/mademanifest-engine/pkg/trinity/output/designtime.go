package output

import "time"

// DesignTime is the JSON marshaling type for the Human Design
// design_time_utc field.  Document 07 §"Formatting Rules" pins the
// format as RFC 3339 UTC with whole-second precision and a trailing
// "Z".  A3 (RESOLVED, Document 12 D22): emission uses *truncation*
// to whole seconds – not rounding, not ceiling – applied to the
// solver's lower-bound timestamp.  Truncate(time.Second) below
// implements that rule.
//
// The zero value of DesignTime serialises as "0001-01-01T00:00:00Z",
// which is the placeholder value Phase 3 emits before Phase 5 lands
// the real bisection solver.
type DesignTime time.Time

// MarshalJSON formats the time as `"YYYY-MM-DDTHH:MM:SSZ"`.  The
// Truncate(time.Second) call is the A3 / D22 truncation rule – it
// drops any sub-second remainder without rounding.
func (d DesignTime) MarshalJSON() ([]byte, error) {
	t := time.Time(d).UTC().Truncate(time.Second)
	const layout = `"2006-01-02T15:04:05Z"`
	return []byte(t.Format(layout)), nil
}

// UnmarshalJSON accepts the canonical layout so a SuccessEnvelope
// can round-trip through encoding/json without losing the wrapped
// Time value (used by the round-trip test in success_test.go).
func (d *DesignTime) UnmarshalJSON(raw []byte) error {
	const layout = `"2006-01-02T15:04:05Z"`
	t, err := time.Parse(layout, string(raw))
	if err != nil {
		return err
	}
	*d = DesignTime(t)
	return nil
}
