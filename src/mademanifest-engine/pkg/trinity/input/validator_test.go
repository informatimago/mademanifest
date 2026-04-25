package input

import (
	"testing"

	// time/tzdata is imported for its side effect: it registers an
	// embedded tzdb so time.LoadLocation works in any environment
	// that the test runs in (tests can run before the production
	// binary explicitly imports it).
	_ "time/tzdata"
)

// canonicalBaseline is the Trinity payload from
// specifications/trinity/trinity.org §"Canonical Payload" – every
// rejection test mutates one field so the test name plus the
// mutation completely describes the case.
const canonicalBaseline = `{
  "birth_date": "1990-04-09",
  "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167,
  "longitude": 4.4
}`

// TestValidateAcceptsCanonicalBaseline locks in the round-trip:
// every later rejection test must change exactly one field, so the
// baseline must remain accepted forever.  A regression here means
// the validator accidentally rejects something it should not.
func TestValidateAcceptsCanonicalBaseline(t *testing.T) {
	got, rej := Validate([]byte(canonicalBaseline))
	if rej != nil {
		t.Fatalf("baseline rejected unexpectedly: %v", rej)
	}
	want := Payload{
		BirthDate: "1990-04-09",
		BirthTime: "18:04",
		Timezone:  "Europe/Amsterdam",
		Latitude:  51.9167,
		Longitude: 4.4,
	}
	if got != want {
		t.Errorf("payload = %+v, want %+v", got, want)
	}
}

// TestValidateRejections drives the validator with one row per rule
// in trinity.org §"Validation Rules" + §"Rejection Rules".  Every
// row pins the rejection Type and the offending Field; per A4 the
// Message text is informational and not asserted.
func TestValidateRejections(t *testing.T) {
	cases := []struct {
		name      string
		payload   string
		wantType  RejectionType
		wantField string
	}{
		// --- incomplete_input: each required field missing.
		{
			name: "missing birth_date",
			payload: `{
  "birth_time": "18:04", "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectIncomplete, wantField: "birth_date",
		},
		{
			name: "missing birth_time",
			payload: `{
  "birth_date": "1990-04-09", "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectIncomplete, wantField: "birth_time",
		},
		{
			name: "missing timezone",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectIncomplete, wantField: "timezone",
		},
		{
			name: "missing latitude",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam", "longitude": 4.4
}`,
			wantType: RejectIncomplete, wantField: "latitude",
		},
		{
			name: "missing longitude",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam", "latitude": 51.9167
}`,
			wantType: RejectIncomplete, wantField: "longitude",
		},

		// --- invalid_input: numeric fields supplied as strings.
		{
			name: "latitude as string",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": "51.9167", "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "latitude",
		},
		{
			name: "longitude as string",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": "4.4"
}`,
			wantType: RejectInvalid, wantField: "longitude",
		},

		// --- invalid_input: malformed dates and times.
		{
			name: "non-existent date 1990-02-30",
			payload: `{
  "birth_date": "1990-02-30", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "birth_date",
		},
		{
			name: "two-digit year 90-04-09",
			payload: `{
  "birth_date": "90-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "birth_date",
		},
		{
			name: "month 13",
			payload: `{
  "birth_date": "1990-13-01", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "birth_date",
		},
		{
			name: "garbage time 9pm",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "9pm",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "birth_time",
		},

		// --- unsupported_input: A5 working assumption.
		{
			name: "seconds present 18:04:00",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04:00",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectUnsupported, wantField: "birth_time",
		},
		{
			name: "fractional seconds 18:04:00.5",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04:00.5",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectUnsupported, wantField: "birth_time",
		},

		// --- invalid_input: timezone shape and aliases (A6).
		{
			name: "timezone abbreviation CET",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "CET",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "timezone",
		},
		{
			name: "timezone link name US/Eastern (A6)",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "US/Eastern",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "timezone",
		},
		{
			name: "timezone garbage Mars/Olympus_Mons",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Mars/Olympus_Mons",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "timezone",
		},

		// --- invalid_input: numeric range.
		{
			name: "latitude > 90",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 91.0, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "latitude",
		},
		{
			name: "latitude < -90",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": -91.0, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "latitude",
		},
		{
			name: "longitude > 180",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 181.0
}`,
			wantType: RejectInvalid, wantField: "longitude",
		},

		// --- invalid_input: shape and unknown fields.
		{
			name: "unknown field place_name",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4,
  "place_name": "Schiedam"
}`,
			wantType: RejectInvalid, wantField: "place_name",
		},
		{
			name: "payload is a JSON array",
			payload: `[{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}]`,
			wantType: RejectInvalid, wantField: "",
		},
		{
			name:      "payload is malformed JSON",
			payload:   `{"birth_date": "1990-04-09",`,
			wantType:  RejectInvalid,
			wantField: "",
		},
		{
			name:      "payload is the literal null",
			payload:   `null`,
			wantType:  RejectInvalid,
			wantField: "",
		},
		{
			name: "trailing data after object",
			payload: `{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
} {"another": "object"}`,
			wantType: RejectInvalid, wantField: "",
		},

		// --- invalid_input: wrong type on a string field.
		{
			name: "birth_date as number",
			payload: `{
  "birth_date": 19900409, "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`,
			wantType: RejectInvalid, wantField: "birth_date",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, r := Validate([]byte(tc.payload))
			if r == nil {
				t.Fatalf("expected rejection, got payload %+v", payload)
			}
			if r.Type != tc.wantType {
				t.Errorf("rejection type = %q, want %q (msg: %s)",
					r.Type, tc.wantType, r.Message)
			}
			if r.Field != tc.wantField {
				t.Errorf("rejection field = %q, want %q (msg: %s)",
					r.Field, tc.wantField, r.Message)
			}
			if r.Message == "" {
				t.Errorf("rejection message is empty (Type=%s Field=%s)",
					r.Type, r.Field)
			}
		})
	}
}

// TestRejectionImplementsErrorInterface guards the convenience that
// callers can return *Rejection wherever an error is expected.
func TestRejectionImplementsErrorInterface(t *testing.T) {
	var _ error = (*Rejection)(nil)
	r := rej(RejectInvalid, "latitude", "out of range")
	if r.Error() == "" {
		t.Errorf("Rejection.Error() returned empty string")
	}
}

// TestRejectionTypeStringsMatchCanonicalErrorTypes pins the equality
// between RejectionType values and the canonical error_type literals
// in the output package, so callers may pass the value directly into
// output.NewError without translation.
func TestRejectionTypeStringsMatchCanonicalErrorTypes(t *testing.T) {
	cases := map[RejectionType]string{
		RejectIncomplete:  "incomplete_input",
		RejectInvalid:     "invalid_input",
		RejectUnsupported: "unsupported_input",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("RejectionType %q != canonical error_type %q", got, want)
		}
	}
}
