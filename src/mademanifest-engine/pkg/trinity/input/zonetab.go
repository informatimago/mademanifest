package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// canonicalZones holds the union of parsed zone.tab + zone1970.tab,
// lazily populated on first use from <ZONEINFO>.  When the load
// fails (ZONEINFO unset, both files missing, parse error) the map
// stays nil and validateTimezone falls back to the legacy
// prefix-list + LoadLocation rejection path documented inline below.
//
// A6 (RESOLVED, Document 12 D24) requires that only canonical IANA
// Area/Location identifiers be accepted — no aliases, no link
// names.  Both zone.tab (per-country, ~418 zones) and zone1970.tab
// (post-1970 equivalence classes, ~312 zones) are IANA-authoritative
// for that set: every row is a Zone (Link entries live exclusively
// in `backward`).  We union both so a per-country name such as
// Europe/Amsterdam (present in zone.tab, consolidated under
// Europe/Brussels in zone1970.tab) is accepted while link names
// such as US/Eastern remain rejected.
//
// In production the Docker builder copies both files from the
// extracted IANA source into ZONEINFO so this loader always finds
// them.  Local non-Docker `go test` runs leave ZONEINFO unset and
// therefore exercise the legacy fallback, which is sufficient for
// CI but is *not* the canon-authoritative behaviour — production
// runs must boot with ZONEINFO pointing at the vendored zoneinfo.
var (
	canonicalZonesOnce sync.Once
	canonicalZones     map[string]struct{}
)

// canonicalZoneFiles lists the IANA tab files we consult, in the
// order they are loaded.  zone.tab carries the per-country names
// users typically request; zone1970.tab is unioned for
// completeness (its post-1970 consolidations include zones that do
// not appear in zone.tab, e.g. some Antarctic stations).
var canonicalZoneFiles = []string{"zone.tab", "zone1970.tab"}

// loadCanonicalZones returns the canonical-zone whitelist.  It is
// safe to call concurrently; the load runs at most once per
// process.  A nil map indicates that no whitelist is available and
// callers should use the legacy fallback.
func loadCanonicalZones() map[string]struct{} {
	canonicalZonesOnce.Do(func() {
		zoneInfo := os.Getenv("ZONEINFO")
		if zoneInfo == "" {
			return
		}
		merged := make(map[string]struct{})
		loaded := false
		for _, name := range canonicalZoneFiles {
			f, err := os.Open(filepath.Join(zoneInfo, name))
			if err != nil {
				continue
			}
			set, err := ParseZoneTab(f)
			f.Close()
			if err != nil {
				continue
			}
			loaded = true
			for z := range set {
				merged[z] = struct{}{}
			}
		}
		if loaded {
			canonicalZones = merged
		}
	})
	return canonicalZones
}

// ParseZoneTab parses an IANA zone1970.tab (or zone.tab) stream and
// returns the set of canonical timezone identifiers it contains.
// The format is:
//
//	# comment lines start with '#'
//	<countries>\t<coords>\t<TZ>[\t<comments>]
//
// Empty lines and comment lines are skipped.  Any non-comment line
// without at least three tab-separated columns is reported as a
// parse error so a malformed input file fails loud rather than
// degrading to a silently incomplete whitelist.
//
// Exported for the unit tests; the loader above uses it on the
// production-image zone1970.tab.
func ParseZoneTab(r io.Reader) (map[string]struct{}, error) {
	set := make(map[string]struct{})
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			return nil, fmt.Errorf("zone1970.tab: line %d: "+
				"expected >= 3 tab-separated columns, got %d",
				lineNo, len(fields))
		}
		zone := strings.TrimSpace(fields[2])
		if zone == "" {
			return nil, fmt.Errorf("zone1970.tab: line %d: empty TZ column", lineNo)
		}
		set[zone] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("zone1970.tab: scan error: %w", err)
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("zone1970.tab: no canonical zones parsed")
	}
	return set, nil
}
