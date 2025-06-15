package geocoding

import (
	"fmt"
	"net/http"
	"net/url"
)

func (zg *ZipGeocoder) GetRepresentativesFromZip(zipCode string, apiKey string) (*http.Response, error) {
	coords, err := zg.GetCoordinates(zipCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates for ZIP %s: %w", zipCode, err)
	}

	baseURL := "https://v3.openstates.org/people.geo"
	params := url.Values{}
	params.Add("lat", fmt.Sprintf("%.6f", coords.Latitude))
	params.Add("lng", fmt.Sprintf("%.6f", coords.Longitude))

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("User-Agent", "Lettersmith/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenStates API: %w", err)
	}

	return resp, nil
}

func (zg *ZipGeocoder) GetCoordinatesForDisplay(zipCode string) string {
	coords, err := zg.GetCoordinates(zipCode)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("ZIP %s: %s, %s (%.6f, %.6f)",
		zipCode, coords.City, coords.State, coords.Latitude, coords.Longitude)
}
