package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"mademanifest-engine/pkg/engine"
)

func testRepoPath(parts ...string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to resolve caller path")
	}
	base := filepath.Dir(filename)
	allParts := append([]string{base}, parts...)
	return filepath.Clean(filepath.Join(allParts...))
}

func normalizeLF(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}

func TestFullPipelineMatchesGoldenFixture(t *testing.T) {
	inputPath := testRepoPath("..", "golden", "GOLDEN_TEST_CASE_V1.json")
	canonDir := testRepoPath("..", "canon")
	ephePath := testRepoPath("..", "ephemeris", "data", "REQUIRED_EPHEMERIS_FILES")

	t.Setenv("SE_EPHE_PATH", ephePath)

	canonPaths, err := engine.ResolveCanonPaths(canonDir, "", "", "")
	if err != nil {
		t.Fatalf("resolve canon paths: %v", err)
	}

	file, err := os.Open(inputPath)
	if err != nil {
		t.Fatalf("open golden input: %v", err)
	}
	defer file.Close()

	output, err := engine.Run(file, canonPaths)
	if err != nil {
		t.Fatalf("engine run: %v", err)
	}

	actual, err := engine.Render(output, false)
	if err != nil {
		t.Fatalf("render output: %v", err)
	}

	expected, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	if !bytes.Equal(normalizeLF(actual), normalizeLF(expected)) {
		t.Fatalf("rendered output does not match golden fixture")
	}
}
