package astro

import "testing"

// TestSignForBoundariesAndWraps drives the every-30°-boundary
// invariant in the canon: each sign is exactly 30° wide,
// start-inclusive / end-exclusive.  The pattern catches both
// floating-point rounding mistakes and accidental off-by-one in
// the integer cast.
func TestSignForBoundariesAndWraps(t *testing.T) {
	cases := []struct {
		longitude float64
		want      string
	}{
		{0.0, "aries"},
		{29.999999, "aries"},
		{30.0, "taurus"},
		{59.999999, "taurus"},
		{60.0, "gemini"},
		{89.999999, "gemini"},
		{90.0, "cancer"},
		{120.0, "leo"},
		{150.0, "virgo"},
		{179.999999, "virgo"},
		{180.0, "libra"},
		{210.0, "scorpio"},
		{240.0, "sagittarius"},
		{270.0, "capricorn"},
		{300.0, "aquarius"},
		{329.999999, "aquarius"},
		{330.0, "pisces"},
		{359.999999, "pisces"},
	}
	for _, tc := range cases {
		if got := SignFor(tc.longitude); got != tc.want {
			t.Errorf("SignFor(%v) = %q, want %q", tc.longitude, got, tc.want)
		}
	}
}

// TestSignForRejectsOutOfRange ensures the helper signals a bug
// instead of wrapping silently.  Callers must normalise to
// [0, 360) before calling SignFor.
func TestSignForRejectsOutOfRange(t *testing.T) {
	cases := []float64{-0.0001, -1.0, -360.0, 360.0, 720.0}
	for _, lon := range cases {
		if got := SignFor(lon); got != "" {
			t.Errorf("SignFor(%v) = %q, want empty string for out-of-range", lon, got)
		}
	}
}
