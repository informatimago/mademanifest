package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"mademanifest-engine/pkg/astrology"
	"mademanifest-engine/pkg/astronomy"
	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/emit_golden"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/gene_keys"
	"mademanifest-engine/pkg/human_design"
	"mademanifest-engine/pkg/process_input"
)

func parseDate(dateStr string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse birth date: %w", err)
	}
	return date, nil
}

func parseClock(timeStr string) (time.Time, error) {
	layouts := []string{"15:04:05", "15:04"}
	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, timeStr)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse birth time: %w", err)
}

func require(cond bool, msg string) error {
	if !cond {
		return fmt.Errorf(msg)
	}
	return nil
}

func ResolveCanonPaths(canonDir, gateSequenceFile, mandalaConstantsFile, nodePolicyFile string) (canon.Paths, error) {
	if canonDir == "" {
		canonDir = "canon"
	}
	if !filepath.IsAbs(canonDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return canon.Paths{}, fmt.Errorf("resolve current working directory: %w", err)
		}
		canonDir = filepath.Join(cwd, canonDir)
	}
	canonDir = filepath.Clean(canonDir)

	resolveCanonFile := func(fileArg, defaultName string) (string, error) {
		path := fileArg
		if path == "" {
			path = defaultName
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(canonDir, path)
		}
		path = filepath.Clean(path)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("find canon file %s: %w", path, err)
		}
		return path, nil
	}

	mandalaConstants, err := resolveCanonFile(mandalaConstantsFile, "mandala_constants.json")
	if err != nil {
		return canon.Paths{}, err
	}
	nodePolicy, err := resolveCanonFile(nodePolicyFile, "node_policy.json")
	if err != nil {
		return canon.Paths{}, err
	}
	gateSequence, err := resolveCanonFile(gateSequenceFile, "gate_sequence_v1.json")
	if err != nil {
		return canon.Paths{}, err
	}

	return canon.Paths{
		MandalaConstants: mandalaConstants,
		NodePolicy:       nodePolicy,
		GateSequence:     gateSequence,
	}, nil
}

func Run(reader io.Reader, canonPaths canon.Paths) (emit_golden.GoldenCase, error) {
	if err := canon.LoadGateSequenceV1(canonPaths.GateSequence); err != nil {
		return emit_golden.GoldenCase{}, fmt.Errorf("load gate sequence: %w", err)
	}

	decoder := json.NewDecoder(reader)
	input, err := process_input.ProcessInput(decoder, canonPaths)
	if err != nil {
		return emit_golden.GoldenCase{}, fmt.Errorf("parse input JSON: %w", err)
	}
	output := *input

	checks := []error{
		require(input.Birth.SecondsPolicy == "assume_00", "expected Birth.SecondsPolicy == \"assume_00\""),
		require(input.EngineContract.Ephemeris == "swiss_ephemeris", "expected EngineContract.Ephemeris == \"swiss_ephemeris\""),
		require(input.EngineContract.Zodiac == "tropical", "expected EngineContract.Zodiac == \"tropical\""),
		require(input.EngineContract.Houses == "placidus", "expected EngineContract.Houses == \"placidus\""),
		require(bool(input.EngineContract.NodePolicyBySystem.HumanDesign), "expected EngineContract.NodePolicyBySystem.HumanDesign"),
		require(bool(input.EngineContract.NodePolicyBySystem.GeneKeys), "expected EngineContract.NodePolicyBySystem.GeneKeys"),
		require(input.EngineContract.NodePolicyBySystem.Astrology == "mean", "expected EngineContract.NodePolicyBySystem.Astrology == \"mean\""),
		require(input.EngineContract.HumanDesignMapping.IntervalRule == "start_inclusive_end_exclusive", "expected EngineContract.HumanDesignMapping.IntervalRule == \"start_inclusive_end_exclusive\""),
	}
	for _, checkErr := range checks {
		if checkErr != nil {
			return emit_golden.GoldenCase{}, checkErr
		}
	}

	birthDate, err := parseDate(input.Birth.Date)
	if err != nil {
		return emit_golden.GoldenCase{}, err
	}
	birthTime, err := parseClock(input.Birth.TimeHHMM)
	if err != nil {
		return emit_golden.GoldenCase{}, err
	}

	localTime := time.Date(
		birthDate.Year(), birthDate.Month(), birthDate.Day(),
		birthTime.Hour(), birthTime.Minute(), 0, 0,
		time.Local,
	)
	utcTime, err := astronomy.ConvertLocalTimeToUTC(localTime, input.Birth.TimezoneIANA)
	if err != nil {
		return emit_golden.GoldenCase{}, fmt.Errorf("convert local time to UTC: %w", err)
	}

	julianDay := astronomy.ConvertUTCToJulianDay(utcTime)
	lat := input.Birth.Latitude
	lon := input.Birth.Longitude

	positions := ephemeris.CalculatePositions(julianDay)
	astrologyData := astrology.CalculateAstrology(positions, julianDay, lat, lon)
	humanDesignData := human_design.CalculateHumanDesign(
		julianDay,
		human_design.LongitudesAt,
		input.EngineContract.HumanDesignMapping,
		input.EngineContract.DesignTimeSolver,
	)
	geneKeysData := gene_keys.DeriveGeneKeys(humanDesignData)

	output.Expected.Astrology = astrologyData
	output.Expected.HumanDesign = humanDesignData
	output.Expected.GeneKeys = geneKeysData
	return output, nil
}

func Render(output emit_golden.GoldenCase, dosLineEndings bool) ([]byte, error) {
	outputJSON, err := emit_golden.EmitGoldenJSON(output)
	if err != nil {
		return nil, fmt.Errorf("marshal output JSON: %w", err)
	}
	if dosLineEndings {
		normalized := bytes.ReplaceAll(outputJSON, []byte("\r\n"), []byte("\n"))
		outputJSON = bytes.ReplaceAll(normalized, []byte("\n"), []byte("\r\n"))
	}
	return outputJSON, nil
}
