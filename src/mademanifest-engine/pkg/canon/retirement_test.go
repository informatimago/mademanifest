package canon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoFileImportsRetiredPoCPackages is the Phase 12 compile-time
// guard that the implementation plan calls for ("no Go file imports
// emit_golden").  It walks every .go file under the engine module
// and fails if any of the retired PoC packages appear in an import
// path:
//
//   * pkg/emit_golden  – removed: GoldenCase data model retired.
//   * pkg/process_input – removed: PoC input decoder retired.
//   * pkg/engine        – removed: file-based orchestrator retired.
//   * pkg/human_design  – removed: PoC HD pipeline retired
//                         (Trinity HD lives at pkg/trinity/hd
//                         with primitives in pkg/hd/calc and
//                         pkg/hd/structure).
//   * pkg/gene_keys     – removed: PoC Gene Keys retired
//                         (Trinity Gene Keys at pkg/trinity/genekeys).
//   * pkg/astrology     – removed: PoC astrology retired
//                         (Trinity astrology at pkg/trinity/astro).
//   * pkg/geolocation   – removed: PoC helper retired.
//
// The walk roots at the engine module directory so we cover both
// production code and tests.  We skip vendored / generated trees
// (none today) and dot-prefixed directories.
func TestNoFileImportsRetiredPoCPackages(t *testing.T) {
	root := engineModuleRoot(t)
	retired := []string{
		"mademanifest-engine/pkg/emit_golden",
		"mademanifest-engine/pkg/process_input",
		"mademanifest-engine/pkg/engine",
		"mademanifest-engine/pkg/human_design",
		"mademanifest-engine/pkg/gene_keys",
		"mademanifest-engine/pkg/astrology",
		"mademanifest-engine/pkg/geolocation",
	}

	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(base, ".go") {
			return nil
		}
		// This file is itself the test that names the retired
		// packages — skip its own contents to avoid a self-match.
		if base == "retirement_test.go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for _, r := range retired {
			// Match in import-path form (quoted) or anywhere in the
			// file.  We use the unquoted check because "go/format"
			// always quotes import paths, but tests that build
			// strings dynamically (e.g. fuzz-style asserts) would
			// also flag.
			needle := `"` + r + `"`
			if strings.Contains(text, needle) {
				offenders = append(offenders,
					relPath(t, root, path)+": imports "+r)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(offenders) > 0 {
		t.Errorf("retired PoC packages still imported:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

func engineModuleRoot(t *testing.T) string {
	t.Helper()
	// We are at <root>/pkg/canon/retirement_test.go; the engine
	// module root is two directories up.
	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs cwd: %v", err)
	}
	return filepath.Clean(filepath.Join(abs, "..", ".."))
}

func relPath(t *testing.T, root, path string) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
