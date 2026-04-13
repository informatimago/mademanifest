package geolocation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// GeographicPosition fetches the latitude and longitude for a given place name.
func GeographicPosition(placeName string) (float64, float64, error) {
	// Encode the place name for the URL
	encodedPlace := url.QueryEscape(placeName)

	// Construct the Nominatim API URL
	apiURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?format=json&q=%s", encodedPlace)

	// Make the HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query geocoding API: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse the JSON response
	var results []struct {
		Lat  string `json:"lat"`
		Lon  string `json:"lon"`
		Name string `json:"display_name"`
	}
	err = json.Unmarshal(body, &results)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Check if results are empty
	if len(results) == 0 {
		return 0, 0, errors.New("no results found for the given place name")
	}

	// Parse latitude and longitude
	var lat, lon float64
	_, err = fmt.Sscanf(results[0].Lat, "%f", &lat)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse latitude: %v", err)
	}
	_, err = fmt.Sscanf(results[0].Lon, "%f", &lon)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse longitude: %v", err)
	}

	return lat, lon, nil
}
