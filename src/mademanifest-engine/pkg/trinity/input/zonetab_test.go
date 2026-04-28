package input

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestParseZoneTabAcceptsCanonicalLines pins the happy path: the
// parser extracts column 3 from each non-comment line, ignores
// comments and blank lines, and tolerates an optional 4th column.
func TestParseZoneTabAcceptsCanonicalLines(t *testing.T) {
	const sample = `# tzdb timezone descriptions
# (comment)
AD	+4230+00131	Europe/Andorra
AE,OM,RE,SC,TF	+2518+05518	Asia/Dubai
AF	+3431+06912	Asia/Kabul
AQ	-6617+11031	Antarctica/Casey	Casey
AU	-3133+15905	Australia/Lord_Howe	Lord Howe Island

XX	+0000+00000	Etc/UTC
`
	got, err := ParseZoneTab(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("ParseZoneTab: %v", err)
	}
	want := []string{
		"Europe/Andorra",
		"Asia/Dubai",
		"Asia/Kabul",
		"Antarctica/Casey",
		"Australia/Lord_Howe",
		"Etc/UTC",
	}
	for _, z := range want {
		if _, ok := got[z]; !ok {
			t.Errorf("ParseZoneTab missing zone %q", z)
		}
	}
	if len(got) != len(want) {
		t.Errorf("ParseZoneTab returned %d zones; want %d", len(got), len(want))
	}
}

// TestParseZoneTabRejectsMalformedLine pins fail-loud behaviour: a
// non-comment line without enough columns must error rather than
// silently produce an incomplete whitelist.
func TestParseZoneTabRejectsMalformedLine(t *testing.T) {
	const sample = "AD\t+4230+00131\tEurope/Andorra\nbogus_line_no_tabs\n"
	_, err := ParseZoneTab(strings.NewReader(sample))
	if err == nil {
		t.Fatal("ParseZoneTab accepted malformed line; want error")
	}
	if !strings.Contains(err.Error(), "tab-separated") {
		t.Errorf("error %q should explain the malformed line", err)
	}
}

// TestParseZoneTabRejectsEmptyResult guards against silently
// accepting a file with only comments or blanks.
func TestParseZoneTabRejectsEmptyResult(t *testing.T) {
	const sample = "# only comments\n\n# another\n"
	_, err := ParseZoneTab(strings.NewReader(sample))
	if err == nil {
		t.Fatal("ParseZoneTab accepted comment-only input; want error")
	}
}

// TestParseZoneTabRejectsEmptyTZColumn pins that whitespace-only TZ
// fields are an error, not a silent skip.
func TestParseZoneTabRejectsEmptyTZColumn(t *testing.T) {
	const sample = "AD\t+4230+00131\t\n"
	_, err := ParseZoneTab(strings.NewReader(sample))
	if err == nil {
		t.Fatal("ParseZoneTab accepted empty TZ column; want error")
	}
	if !strings.Contains(err.Error(), "empty TZ column") {
		t.Errorf("error %q should mention the empty TZ column", err)
	}
}

// resetCanonicalZones clears the sync.Once + cached map so a test
// can re-trigger the lazy load with a different ZONEINFO.
func resetCanonicalZones() {
	canonicalZonesOnce = sync.Once{}
	canonicalZones = nil
}

// TestValidateTimezoneUsesWhitelistWhenAvailable proves the
// production path: when ZONEINFO points at a directory containing
// zone1970.tab, validateTimezone accepts only zones in that file
// — even aliases the legacy prefix list would miss.
//
// Asia/Calcutta is a *link* to Asia/Kolkata; it resolves under
// time.LoadLocation and is *not* caught by ianaLinkPrefixes, so the
// legacy fallback would erroneously accept it.  The whitelist path
// must reject it because zone1970.tab lists only Asia/Kolkata.
func TestValidateTimezoneUsesWhitelistWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	tab := "AD\t+4230+00131\tEurope/Andorra\nIN\t+2232+08822\tAsia/Kolkata\n"
	if err := os.WriteFile(filepath.Join(dir, "zone1970.tab"), []byte(tab), 0o644); err != nil {
		t.Fatalf("write tab: %v", err)
	}
	t.Setenv("ZONEINFO", dir)
	resetCanonicalZones()
	t.Cleanup(resetCanonicalZones)

	if r := validateTimezone("Europe/Andorra"); r != nil {
		t.Errorf("Europe/Andorra: %v; want accept", r)
	}
	if r := validateTimezone("Asia/Calcutta"); r == nil {
		t.Error("Asia/Calcutta accepted under whitelist; want rejection " +
			"(it is a link to Asia/Kolkata, not a canonical zone)")
	} else if !strings.Contains(r.Message, "canonical") {
		t.Errorf("rejection message %q should mention canonical IANA form", r.Message)
	}
}

// TestValidateTimezoneFallsBackWhenZoneInfoUnset proves the dev /
// CI fallback path: with ZONEINFO unset, validateTimezone uses the
// legacy prefix list + LoadLocation against Go's embedded tzdb.
func TestValidateTimezoneFallsBackWhenZoneInfoUnset(t *testing.T) {
	t.Setenv("ZONEINFO", "")
	resetCanonicalZones()
	t.Cleanup(resetCanonicalZones)

	// Europe/Amsterdam is in Go's embedded tzdb, so the fallback
	// LoadLocation succeeds.
	if r := validateTimezone("Europe/Amsterdam"); r != nil {
		t.Errorf("Europe/Amsterdam (fallback): %v; want accept", r)
	}
	// US/Eastern is on the prefix-list — fallback rejects it.
	if r := validateTimezone("US/Eastern"); r == nil {
		t.Error("US/Eastern (fallback): accepted; want rejection")
	}
}
