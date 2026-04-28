package canon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AssertTZDBVersion verifies that the runtime IANA tzdata release
// matches the canon-pinned TZDBVersion (D20 / A1 RESOLVED).  It is
// designed for the Docker / Kubernetes deployment where the builder
// stage has compiled IANA tzdata 2026a into ZONEINFO and dropped the
// upstream "version" file as <ZONEINFO>/+VERSION.
//
// Behaviour:
//
//   * If the environment variable ZONEINFO is set and points at a
//     directory containing a "+VERSION" file, AssertTZDBVersion reads
//     that file and compares its trimmed contents against TZDBVersion.
//     A mismatch is a fatal canon violation and is returned as an
//     error.  This is the canonical production path.
//
//   * If ZONEINFO is unset, AssertTZDBVersion returns nil.  Local
//     development and CI runs that fall back to Go's embedded
//     time/tzdata or the host system zoneinfo skip the assertion;
//     they remain responsible for matching 2026a out of band.  The
//     production Dockerfile sets ZONEINFO so the assertion always
//     fires inside the container.
//
//   * If ZONEINFO is set but the +VERSION marker is missing, the
//     assertion fails fast.  Operating against an unmarked custom
//     ZONEINFO would defeat the canon's pinning intent; the operator
//     must either restore the marker or unset ZONEINFO.
//
// The assertion is intentionally narrow: it does not try to parse
// individual zones or DST rules, because tzdb release identifiers are
// the canonical signal IANA itself uses to reference a release.
func AssertTZDBVersion() error {
	zoneInfo := os.Getenv("ZONEINFO")
	if zoneInfo == "" {
		return nil
	}
	markerPath := filepath.Join(zoneInfo, "+VERSION")
	raw, err := os.ReadFile(markerPath)
	if err != nil {
		return fmt.Errorf("tzdb version marker missing at %s "+
			"(production builds must compile IANA %s into ZONEINFO and "+
			"copy its 'version' file as +VERSION): %w",
			markerPath, TZDBVersion, err)
	}
	got := strings.TrimSpace(string(raw))
	if got != TZDBVersion {
		return fmt.Errorf("tzdb version mismatch under ZONEINFO=%s: "+
			"marker reports %q, canon pins %q (D20 / A1 RESOLVED — "+
			"runtime tzdata must match the canon release exactly)",
			zoneInfo, got, TZDBVersion)
	}
	return nil
}
