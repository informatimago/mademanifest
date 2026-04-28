package canon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAssertTZDBVersionUnsetIsSkip pins the documented behaviour:
// when ZONEINFO is not set the assertion is a no-op so local dev
// and CI runs that fall back to the host's system zoneinfo still
// boot.
func TestAssertTZDBVersionUnsetIsSkip(t *testing.T) {
	t.Setenv("ZONEINFO", "")
	if err := AssertTZDBVersion(); err != nil {
		t.Fatalf("AssertTZDBVersion with empty ZONEINFO returned %v; want nil", err)
	}
}

// TestAssertTZDBVersionMatchAccepts pins that a +VERSION marker whose
// trimmed contents equal TZDBVersion is accepted.
func TestAssertTZDBVersionMatchAccepts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "+VERSION"),
		[]byte(TZDBVersion+"\n"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	t.Setenv("ZONEINFO", dir)
	if err := AssertTZDBVersion(); err != nil {
		t.Fatalf("AssertTZDBVersion with matching marker returned %v; want nil", err)
	}
}

// TestAssertTZDBVersionMissingMarkerFails pins that a custom ZONEINFO
// without a +VERSION marker is rejected: the canon pinning intent
// would otherwise be defeated by an unmarked operator-provided dir.
func TestAssertTZDBVersionMissingMarkerFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZONEINFO", dir)
	err := AssertTZDBVersion()
	if err == nil {
		t.Fatal("AssertTZDBVersion with missing marker returned nil; want error")
	}
	if !strings.Contains(err.Error(), "+VERSION") {
		t.Errorf("error message %q should mention +VERSION", err)
	}
}

// TestAssertTZDBVersionMismatchFails pins that any release other than
// the canon-pinned TZDBVersion is rejected.
func TestAssertTZDBVersionMismatchFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "+VERSION"),
		[]byte("1999z\n"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	t.Setenv("ZONEINFO", dir)
	err := AssertTZDBVersion()
	if err == nil {
		t.Fatal("AssertTZDBVersion with mismatched marker returned nil; want error")
	}
	if !strings.Contains(err.Error(), TZDBVersion) {
		t.Errorf("error message %q should mention canon TZDBVersion %q",
			err, TZDBVersion)
	}
}
