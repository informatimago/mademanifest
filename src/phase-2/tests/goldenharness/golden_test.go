package goldenharness

import (
	"bytes"
	"os"
	"testing"
)

// ComputeGoldenOutput must be implemented by the engine.
// It must return the exact JSON output, byte-for-byte.
func ComputeGoldenOutput() ([]byte, error) {
	return nil, nil
}

func TestGoldenCaseV1(t *testing.T) {
	expected, err := os.ReadFile("../../data/GOLDEN_TEST_CASE_V1.json")
	if err != nil {
		t.Fatalf("failed to read golden fixture: %v", err)
	}

	actual, err := ComputeGoldenOutput()
	if err != nil {
		t.Fatalf("engine returned error: %v", err)
	}

	if !bytes.Equal(expected, actual) {
		t.Fatalf("golden test case mismatch: output does not match canonical fixture")
	}
}
