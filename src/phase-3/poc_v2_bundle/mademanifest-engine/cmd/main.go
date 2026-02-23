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
	"mademanifest-engine/pkg/emit_golden"
)

func parseDate(dateStr string) time.Time {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Fatalf("Failed to parse birth date: %v", err)
	}
	// log.Printf("Parsed date %v into %v",dateStr,date)
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
	// log.Printf("Parsed time %v into %v",timeStr, tyme)
	return tyme
}

func assert(cond bool, msg string) {
    if !cond {
		log.Fatalf(msg)
    }
}

func engine(decoder *json.Decoder) emit_golden.GoldenCase {
	input, err := process_input.ProcessInput(decoder)
	if err != nil {
		log.Fatalf("Failed to parse the JSON file: %v", err)
	}
	var output = *input

	// log.Printf("input = %v",input)
	// log.Printf("output = %v",output)


	assert(input.Birth.SecondsPolicy=="assume_00",
		"Expected Birth.SecondsPolicy == \"assume_00\"")
	assert(input.EngineContract.Ephemeris == "swiss_ephemeris",
		"Expected EngineContract.Ephemeris == \"swiss_ephemeris\"")
	assert(input.EngineContract.Zodiac == "tropical",
		"Expected EngineContract.Zodiac == \"tropical\"")
	assert(input.EngineContract.Houses == "placidus",
		"Expected EngineContract.Houses == \"placidus\"")
	assert(bool(input.EngineContract.NodePolicyBySystem.HumanDesign),
		"Expected EngineContract.NodePolicyBySystem.HumanDesign")
	assert(bool(input.EngineContract.NodePolicyBySystem.GeneKeys),
		"Expected EngineContract.NodePolicyBySystem.GeneKeys")
	assert(input.EngineContract.NodePolicyBySystem.Astrology == "mean",
		"Expected EngineContract.NodePolicyBySystem.Astrology == \"mean\"")
    assert(input.EngineContract.HumanDesignMapping.IntervalRule == "start_inclusive_end_exclusive",
		"Expected EngineContract.HumanDesignMapping.IntervalRule == \"start_inclusive_end_exclusive\"")


	birthDate := parseDate(input.Birth.Date)
	birthTime := parseTime(input.Birth.TimeHHMM)

	// Convert local time to UTC
	var utcTime time.Time
	localTime := time.Date(birthDate.Year(), birthDate.Month(), birthDate.Day(),
		birthTime.Hour(), birthTime.Minute(), 0, 0,
		time.Local)
	utcTime, err = astronomy.ConvertLocalTimeToUTC(localTime, input.Birth.TimezoneIANA)
	if err != nil {
		log.Fatalf("Error: %v localTime= %v",err,localTime)
	}

	// Convert UTC to Julian Day
	julianDay := astronomy.ConvertUTCToJulianDay(utcTime)
	// log.Printf("birthDate = %v",birthDate)
	// log.Printf("birthTime = %v",birthTime)
	// log.Printf("utcTime   = %v",utcTime)
	// log.Printf("julianDay = %v",julianDay)

	// lat, lon, err_pos := geolocation.GeographicPosition(input.Birth.PlaceName)
	// if err_pos != nil {
	// 	fmt.Printf("Error: %v\n", err_pos)
	// 	return
	// }
	lat := input.Birth.Latitude
	lon := input.Birth.Longitude

	// Calculate ephemeris positions
	positions := ephemeris.CalculatePositions(julianDay)

	// Calculate astrology data
	astrologyData := astrology.CalculateAstrology(positions,julianDay,lat,lon)

	// Calculate Human Design data
	humanDesignData := human_design.CalculateHumanDesign(
		julianDay,
		human_design.LongitudesAt,
		input.EngineContract.HumanDesignMapping,
		input.EngineContract.DesignTimeSolver,
	)

	// Derive Gene Keys
	geneKeysData := gene_keys.DeriveGeneKeys(humanDesignData)

	output.Expected.Astrology = astrologyData
	output.Expected.HumanDesign = humanDesignData
	output.Expected.GeneKeys = geneKeysData
	return output
}

func main() {

    if len(os.Args) < 3 {
        fmt.Fprintf(os.Stderr, "Usage: %s $inputFile $outputFile\n", os.Args[0])
        os.Exit(1)
    }

    inputFile := os.Args[1]
    outputFile := os.Args[2]



	// swephgo.SetEphePath([]byte("/usr/local/share/swisseph"))

	// Load birth data from JSON
	var file *os.File
	var err error

	file, err = os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open JSON file: %v", err)
	}
	defer file.Close()

	var decoder = json.NewDecoder(file)
	var output = engine(decoder)

	outputJSON, err := emit_golden.EmitGoldenJSON(output)
	// log.Printf("outputJSON = %v",string(outputJSON))
	if err != nil {
		log.Fatalf("Failed to marshal output to JSON: %v", err)
	}

    if err := os.WriteFile(outputFile, outputJSON, 0644); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
        os.Exit(1)
    }
}
