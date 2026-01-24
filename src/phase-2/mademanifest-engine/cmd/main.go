package main

import (
	"encoding/json"
	"fmt"
	"log"
	"mademanifest-engine/pkg/astronomy"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/astrology"
	"mademanifest-engine/pkg/human_design"
	"mademanifest-engine/pkg/gene_keys"
	"os"
	"time"
)

func main() {
	// Load birth data from JSON
	file, err := os.Open("golden_test_case_v1_jaimie_1990_04_09_1804_schiedam.json")
	if err != nil {
		log.Fatalf("Failed to open JSON file: %v", err)
	}
	defer file.Close()

	var birthData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&birthData); err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
	}

	// Extract birth data
	birthDateStr := birthData["birth"].(map[string]interface{})["date"].(string)
	birthTimeStr := birthData["birth"].(map[string]interface{})["time_hh_mm"].(string)
	placeName := birthData["birth"].(map[string]interface{})["place_name"].(string)
	timezone := birthData["birth"].(map[string]interface{})["timezone_iana"].(string)

	birthDate, err := time.Parse("2006-01-02", birthDateStr)
	if err != nil {
		log.Fatalf("Failed to parse birth date: %v", err)
	}
	birthTime, err := time.Parse("15:04", birthTimeStr)
	if err != nil {
		log.Fatalf("Failed to parse birth time: %v", err)
	}

	// Convert local time to UTC
	utcTime := astronomy.ConvertLocalTimeToUTC(time.Date(birthDate.Year(), birthDate.Month(), birthDate.Day(), birthTime.Hour(), birthTime.Minute(), 0, 0, time.UTC), timezone)

	// Convert UTC to Julian Day
	julianDay := astronomy.ConvertUTCToJulianDay(utcTime)

	// Calculate ephemeris positions
	ephemeris := ephemeris.NewEphemeris()
	positions := ephemeris.CalculatePositions(julianDay)

	// Calculate astrology data
	astrologyData := astrology.CalculateAstrology(positions)

	// Calculate Human Design data
	humanDesignData := human_design.CalculateHumanDesign(positions, 88.0)

	// Derive Gene Keys
	geneKeysData := gene_keys.DeriveGeneKeys(humanDesignData)

	// Output results
	output := map[string]interface{}{
		"astrology":     astrologyData,
		"human_design":  humanDesignData,
		"gene_keys":     geneKeysData,
	}
	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal output to JSON: %v", err)
	}
	fmt.Println(string(outputJSON))
}
