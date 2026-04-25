// Package golden iterates the Trinity golden test pack rooted at
// src/golden/trinity/ and compares engine responses against frozen
// expected.json fixtures.
//
// trinity.org §"Golden Test Pack Requirements" pins the minimums per
// category and the comparison rules:
//
//   * valid_baseline:       >= 3 fixtures
//   * valid_edge:           >= 5 fixtures
//   * invalid_input:        >= 5 fixtures
//   * incomplete_input:     >= 5 fixtures
//   * unsupported_input:    >= 2 fixtures
//   * regression_sentinel:  >= 3 fixtures
//
// Comparison rules:
//
//   * Success cases use semantic JSON equality, ignoring the
//     metadata block (which depends on the build's EngineVersion).
//     metadata is asserted separately to equal output.CurrentMetadata().
//   * Error cases compare error_type plus envelope shape (status,
//     metadata, non-empty message) per ambiguity A4: until the
//     canon owner pins error message text, only error_type is a
//     hard contract.
package golden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"mademanifest-engine/pkg/trinity/output"
)

// Category is one of the canon golden-pack categories.  See
// trinity.org §"Golden Test Pack Requirements".
type Category string

const (
	CategoryValidBaseline      Category = "valid_baseline"
	CategoryValidEdge          Category = "valid_edge"
	CategoryInvalidInput       Category = "invalid_input"
	CategoryIncompleteInput    Category = "incomplete_input"
	CategoryUnsupportedInput   Category = "unsupported_input"
	CategoryRegressionSentinel Category = "regression_sentinel"
)

// Categories returns the canonical list of categories in the order
// the canon enumerates them.  Tests that walk the pack iterate this
// list to surface any new (or missing) category at the directory
// level rather than via a magic string elsewhere.
func Categories() []Category {
	return []Category{
		CategoryValidBaseline,
		CategoryValidEdge,
		CategoryInvalidInput,
		CategoryIncompleteInput,
		CategoryUnsupportedInput,
		CategoryRegressionSentinel,
	}
}

// MinimumCounts pins the per-category minimums from trinity.org.
// LoadFixtures uses this to fail loudly if the on-disk pack ever
// drops below the canon minimum, even before any HTTP request.
var MinimumCounts = map[Category]int{
	CategoryValidBaseline:      3,
	CategoryValidEdge:          5,
	CategoryInvalidInput:       5,
	CategoryIncompleteInput:    5,
	CategoryUnsupportedInput:   2,
	CategoryRegressionSentinel: 3,
}

// IsErrorCategory returns true when fixtures of this category are
// expected to produce a Trinity ErrorEnvelope (rather than a
// SuccessEnvelope).  The three error categories share a common
// runner code path; the three success categories share a different
// one.
func IsErrorCategory(c Category) bool {
	switch c {
	case CategoryInvalidInput, CategoryIncompleteInput, CategoryUnsupportedInput:
		return true
	}
	return false
}

// Fixture is one (input, expected) pair on disk.  Name is the
// directory leaf (e.g. "schiedam_1990_04_09"); RelativePath includes
// the category prefix for diagnostic display.
type Fixture struct {
	Category     Category
	Name         string
	RelativePath string
	InputPath    string
	ExpectedPath string
}

// LoadFixtures walks the pack rooted at packRoot and returns every
// fixture it finds, grouped by category in deterministic order
// (sorted by Name within each category).
//
// LoadFixtures fails when any required category directory is
// missing or carries fewer fixtures than MinimumCounts requires;
// the canon minimums are part of the contract and a regression in
// fixture count is itself a Phase 11 failure.
func LoadFixtures(packRoot string) ([]Fixture, error) {
	var out []Fixture
	for _, cat := range Categories() {
		dir := filepath.Join(packRoot, string(cat))
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", dir, err)
		}
		var inCat []Fixture
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			caseDir := filepath.Join(dir, name)
			input := filepath.Join(caseDir, "input.json")
			expected := filepath.Join(caseDir, "expected.json")
			if _, err := os.Stat(input); err != nil {
				return nil, fmt.Errorf("fixture %s/%s: missing input.json: %w",
					cat, name, err)
			}
			if _, err := os.Stat(expected); err != nil {
				return nil, fmt.Errorf("fixture %s/%s: missing expected.json: %w",
					cat, name, err)
			}
			inCat = append(inCat, Fixture{
				Category:     cat,
				Name:         name,
				RelativePath: filepath.Join(string(cat), name),
				InputPath:    input,
				ExpectedPath: expected,
			})
		}
		sort.Slice(inCat, func(i, j int) bool {
			return inCat[i].Name < inCat[j].Name
		})
		min := MinimumCounts[cat]
		if len(inCat) < min {
			return nil, fmt.Errorf("category %s: %d fixture(s) on disk, "+
				"canon minimum is %d", cat, len(inCat), min)
		}
		out = append(out, inCat...)
	}
	return out, nil
}

// LoadInput reads a fixture's input.json bytes verbatim.  Negative
// fixtures (incomplete_input, invalid_input, unsupported_input) may
// contain raw JSON that intentionally violates the canonical
// payload shape; the runner posts the bytes as-is.
func (f Fixture) LoadInput() ([]byte, error) {
	return os.ReadFile(f.InputPath)
}

// ExpectedSuccess is the JSON shape of a success-case expected.json.
// It mirrors output.SuccessEnvelope exactly except for the metadata
// block, which is omitted: metadata depends on the running build's
// EngineVersion and would force fixture regeneration on every phase
// bump.  The runner asserts metadata separately against
// output.CurrentMetadata().
type ExpectedSuccess struct {
	Status      string                `json:"status"`
	InputEcho   output.InputEcho      `json:"input_echo"`
	Astrology   output.Astrology      `json:"astrology"`
	HumanDesign output.HumanDesignOut `json:"human_design"`
	GeneKeys    output.GeneKeysOut    `json:"gene_keys"`
}

// ExpectedError is the JSON shape of an error-case expected.json.
// Per ambiguity A4 the runner asserts only error_type and the
// envelope shape (status field, metadata block, non-empty message);
// message text is informational prose and is not pinned.
type ExpectedError struct {
	Status string `json:"status"`
	Error  struct {
		ErrorType string `json:"error_type"`
	} `json:"error"`
}

// LoadExpectedSuccess decodes a success-case expected.json.  Use
// LoadExpectedError for error categories.
func (f Fixture) LoadExpectedSuccess() (ExpectedSuccess, error) {
	var v ExpectedSuccess
	if err := loadJSON(f.ExpectedPath, &v); err != nil {
		return v, err
	}
	if v.Status != string(output.StatusSuccess) {
		return v, fmt.Errorf("fixture %s: expected.json status = %q, want %q",
			f.RelativePath, v.Status, output.StatusSuccess)
	}
	return v, nil
}

// LoadExpectedError decodes an error-case expected.json.
func (f Fixture) LoadExpectedError() (ExpectedError, error) {
	var v ExpectedError
	if err := loadJSON(f.ExpectedPath, &v); err != nil {
		return v, err
	}
	if v.Status != string(output.StatusError) {
		return v, fmt.Errorf("fixture %s: expected.json status = %q, want %q",
			f.RelativePath, v.Status, output.StatusError)
	}
	if v.Error.ErrorType == "" {
		return v, fmt.Errorf("fixture %s: expected.json error.error_type is empty",
			f.RelativePath)
	}
	return v, nil
}

func loadJSON(path string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// CompareSuccess asserts that got equals want by semantic JSON
// equality, with the metadata block excluded from the comparison
// (assert it separately against output.CurrentMetadata()).
//
// Returns nil on match, or an error whose message names the first
// drift.  The structural diff produced here is human-readable —
// not a unified diff, but a breadcrumb trail to the offending
// section.
func CompareSuccess(got output.SuccessEnvelope, want ExpectedSuccess) error {
	if got.Status != want.Status {
		return fmt.Errorf("status: got %q, want %q", got.Status, want.Status)
	}
	if !reflect.DeepEqual(got.InputEcho, want.InputEcho) {
		return fmt.Errorf("input_echo: got %+v, want %+v", got.InputEcho, want.InputEcho)
	}
	if !reflect.DeepEqual(got.Astrology, want.Astrology) {
		return fmt.Errorf("astrology drift: got %+v\nwant %+v", got.Astrology, want.Astrology)
	}
	if !reflect.DeepEqual(got.HumanDesign, want.HumanDesign) {
		return fmt.Errorf("human_design drift: got %+v\nwant %+v",
			got.HumanDesign, want.HumanDesign)
	}
	if !reflect.DeepEqual(got.GeneKeys, want.GeneKeys) {
		return fmt.Errorf("gene_keys drift: got %+v\nwant %+v",
			got.GeneKeys, want.GeneKeys)
	}
	return nil
}

// CompareError asserts that got is a well-formed Trinity error
// envelope whose error_type matches want.Error.ErrorType.  Per
// ambiguity A4, message text is *not* compared; only error_type and
// the structural shape (status field, non-empty message, canonical
// metadata block) are pinned.
//
// CurrentMetadata is taken as the canonical metadata block and
// passed in by the caller so this function does not depend on the
// canon package.  Tests pass output.CurrentMetadata() at the call
// site.
func CompareError(got output.ErrorEnvelope, want ExpectedError, currentMetadata output.Metadata) error {
	if got.Status != want.Status {
		return fmt.Errorf("status: got %q, want %q", got.Status, want.Status)
	}
	if got.Error.Type != want.Error.ErrorType {
		return fmt.Errorf("error_type: got %q, want %q",
			got.Error.Type, want.Error.ErrorType)
	}
	if got.Error.Message == "" {
		return errors.New("error.message is empty; canon requires non-empty message")
	}
	if got.Metadata != currentMetadata {
		return fmt.Errorf("metadata drift: got %+v\nwant %+v",
			got.Metadata, currentMetadata)
	}
	return nil
}

// SemanticJSONEqual compares two JSON byte slices by canonical
// re-serialization: each is decoded into an interface{}, then the
// trees are compared with reflect.DeepEqual.  Map key order does
// not matter; numeric precision is preserved by json.Number.
//
// The runner uses this for the success-path semantic-equality
// option (used by tests that want a stricter compare than the
// field-by-field DeepEqual on output.SuccessEnvelope).
func SemanticJSONEqual(a, b []byte) (bool, error) {
	pa, err := canonicalDecode(a)
	if err != nil {
		return false, fmt.Errorf("decode left: %w", err)
	}
	pb, err := canonicalDecode(b)
	if err != nil {
		return false, fmt.Errorf("decode right: %w", err)
	}
	return reflect.DeepEqual(pa, pb), nil
}

func canonicalDecode(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// EnsurePackRoot returns packRoot/<category> for one category, or
// an error if the directory does not exist.  Used by tests that
// scope themselves to a single category before walking it.
func EnsurePackRoot(packRoot string, c Category) (string, error) {
	dir := filepath.Join(packRoot, string(c))
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	return dir, nil
}
