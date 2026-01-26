package main

import (
	"os"
	"time"
	"fmt"
	"log"
	"encoding/json"
	"mademanifest-engine/pkg/process_input"
	"mademanifest-engine/pkg/astronomy"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/astrology"
	"mademanifest-engine/pkg/human_design"
	"mademanifest-engine/pkg/gene_keys"
)

func parseDate(dateStr string) time.Time {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Fatalf("Failed to parse birth date: %v", err)
	}
	return date
}

func parseTime(timeStr string) time.Time {
	layouts := []string{"15:04:05", "15:04"}
	var tyme time.Time
	var err error
	for _, layout := range layouts {
		tyme, err = time.Parse(layout, timeStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Fatalf("Failed to parse birth time: %v", err)
	}
	return tyme
}

func assert(cond bool, msg string) {
    if !cond {
		log.Fatalf(msg)
    }
}

func engine(decoder *json.Decoder) map[string]interface{} {
	input, err = process_input.ProcessInput(decoder)
	if err != nil {
		log.Fatalf("Failed to parse the JSON file: %v", err)
	}

	assert(input.Birth.SecondsPolicy=="assume_00",
		"Expected Birth.SecondsPolicy == \"assume_00\"")
	assert(input.EngineContract.Ephemeris == "swiss_ephemeris",
		"Expected EngineContract.Ephemeris == \"swiss_ephemeris\"")
	assert(input.EngineContract.Zodiac == "tropical",
		"Expected EngineContract.Zodiac == \"tropical\"")
	assert(input.EngineContract.Houses == "placidus",
		"Expected EngineContract.Houses == \"placidus\"")
	assert(input.EngineContract.NodePolicyBySystem.HumanDesign,
		"Expected EngineContract.NodePolicyBySystem.HumanDesign")
	assert(input.EngineContract.NodePolicyBySystem.GeneKeys,
		"Expected EngineContract.NodePolicyBySystem.GeneKeys")
	assert(input.EngineContract.NodePolicyBySystem.Astrology == "mean",
		"Expected EngineContract.NodePolicyBySystem.Astrology == \"mean\"")
    assert(input.EngineContract.HumanDesignMapping.IntervalRule == "start_inclusive_end_exclusive",
		"Expected EngineContract.HumanDesignMapping.IntervalRule == \"start_inclusive_end_exclusive\"")


	birthDate := parseDate(input.Birth.Date)
	birthTime := parseTime(input.Birth.TimeHHMM)

	// Convert local time to UTC
	var utcTime time.Time
	utcTime, err = astronomy.ConvertLocalTimeToUTC(time.Date(birthDate.Year(), birthDate.Month(), birthDate.Day(), birthTime.Hour(), birthTime.Minute(), 0, 0, time.UTC), input.Birth.TimezoneIANA)

	// Convert UTC to Julian Day
	julianDay := astronomy.ConvertUTCToJulianDay(utcTime)

	// lat, lon, err_pos := geolocation.GeographicPosition(input.Birth.PlaceName)
	// if err_pos != nil {
	// 	fmt.Printf("Error: %v\n", err_pos)
	// 	return
	// }
	lat := input.Birth.Latitude
	lon := input.Birth.Longitude

	// Calculate ephemeris positions
	ephemeris := ephemeris.NewEphemeris()
	positions := ephemeris.CalculatePositions(julianDay)

	// Calculate astrology data
	astrologyData := astrology.CalculateAstrology(positions,julianDay,lat,lon)

	// Calculate Human Design data
	humanDesignData := human_design.CalculateHumanDesign(positions,
		input.EngineContract.HumanDesignMapping,
		input.EngineContract.DesignTimeSolver.SunOffsetDeg)

	// Derive Gene Keys
	geneKeysData := gene_keys.DeriveGeneKeys(humanDesignData)

	// Output results
	output := map[string]interface{}{
		"astrology":     astrologyData,
		"human_design":  humanDesignData,
		"gene_keys":     geneKeysData,
	}
	return output
}

func main() {
	// swephgo.SetEphePath([]byte("/usr/local/share/swisseph"))

	// Load birth data from JSON
	var file *os.File
	var err error
	var input *process_input.InputData

	file, err = os.Open("golden_test_case_v1_jaimie_1990_04_09_1804_schiedam.json")
	if err != nil {
		log.Fatalf("Failed to open JSON file: %v", err)
	}
	defer file.Close()

	var decoder = json.NewDecoder(file)
	var output = engine(decoder)


	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal output to JSON: %v", err)
	}
	fmt.Println(string(outputJSON))
}
