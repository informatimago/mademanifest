package output

import "time"

// DesignTime is the JSON marshaling type for the Human Design
// design_time_utc field.  Trinity.org §"Formatting Rules" lines
// 582-584 pins the format as RFC 3339 UTC with whole-second
// precision and a trailing "Z".  Per the A3 working assumption
// (canon owner has not yet pinned the rounding rule for the
// design-time bisection's whole-second emission), the marshaler
// applies a *floor* truncation: any sub-second component is dropped.
//
// The zero value of DesignTime serialises as "0001-01-01T00:00:00Z",
// which is the placeholder value Phase 3 emits before Phase 5 lands
// the real bisection solver.
type DesignTime time.Time

// MarshalJSON formats the time as `"YYYY-MM-DDTHH:MM:SSZ"`.  The
// Truncate(time.Second) call is the A3-floor rule – it drops the
// nanosecond component without rounding.
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
