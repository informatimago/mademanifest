package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
	
	"mademanifest-engine/pkg/astronomy"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/astrology"
	"mademanifest-engine/pkg/human_design"
	"mademanifest-engine/pkg/gene_keys"
)

func TestFullPipeline(t *testing.T) {
	// Simulate reading from the golden test case
	birthData := map[string]interface{}{
		"birth": map[string]interface{}{
			"date": "1990-04-09",
			"time_hh_mm": "18:04",
			"timezone_iana": "Europe/Amsterdam",
		},
	}
	
	// Parse date and time
	birthDateStr := birthData["birth"].(map[string]interface{})["date"].(string)
	birthTimeStr := birthData["birth"].(map[string]interface{})["time_hh_mm"].(string)
	timezone := birthData["birth"].(map[string]interface{})["timezone_iana"].(string)
	
	// Parse birth date and time
	birthDate := parseDate(birthDateStr)
	birthTime := parseTime(birthTimeStr)
	
	// Convert local time to UTC
	utcTime, err := astronomy.ConvertLocalTimeToUTC(time.Date(birthDate.Year(), birthDate.Month(), birthDate.Day(), birthTime.Hour(), birthTime.Minute(), 0, 0, time.UTC), timezone)
	if err != nil {
		t.Fatalf("Failed to convert to UTC: %v", err)
	}
	
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
	
	// Verify all components were calculated
	if astrologyData == nil {
		t.Error("Astrology data should not be nil")
	}
	
	if humanDesignData == nil {
		t.Error("Human design data should not be nil")
	}
	
	if geneKeysData == nil {
		t.Error("Gene keys data should not be nil")
	}
	
	// Verify the structure of output
	output := map[string]interface{}{
		"astrology":     astrologyData,
		"human_design":  humanDesignData,
		"gene_keys":     geneKeysData,
	}
	
	// Verify output is valid JSON
	_, err = json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Errorf("Output should be valid JSON: %v", err)
	}
}

// Helper function to parse date - copied from main.go
func parseDate(dateStr string) time.Date {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(err)
	}
	return date
}

// Helper function to parse time - copied from main.go
func parseTime(timeStr string) time.Time {
	layouts := []string{"15:04:05", "15:04"}
	var time time.Time
	var err error
	for _, layout := range layouts {
		time, err = time.Parse(layout, timeStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		panic(err)
	}
	return time
}
