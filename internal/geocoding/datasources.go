package geocoding

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// loadFromCensusBureau downloads ZIP code data from the US Census Bureau's official Gazetteer files
func (zg *ZipGeocoder) loadFromCensusBureau() error {
	log.Println("Downloading ZIP code data from US Census Bureau...")

	// Try multiple years to future-proof against URL changes
	currentYear := time.Now().Year()
	urls := zg.generateCensusBureauURLs(currentYear)

	client := &http.Client{Timeout: 120 * time.Second}

	var lastErr error
	for i, url := range urls {
		log.Printf("Attempting Census Bureau URL %d/%d: %s", i+1, len(urls), url)

		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			log.Printf("Failed to connect to %s: %v", url, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("census bureau api returned status: %d", resp.StatusCode)
			log.Printf("Census Bureau returned status %d for %s", resp.StatusCode, url)
			continue
		}

		// Success! Process the data
		log.Printf("Successfully connected to Census Bureau: %s", url)
		return zg.processCensusBureauData(resp.Body)
	}

	// All URLs failed
	return fmt.Errorf("all Census Bureau URLs failed, last error: %w", lastErr)
}

// generateCensusBureauURLs creates a list of URLs to try, starting with custom URL if provided
func (zg *ZipGeocoder) generateCensusBureauURLs(currentYear int) []string {
	var urls []string

	// Check for custom URL from environment variable first
	if customURL := os.Getenv("CENSUS_BUREAU_URL"); customURL != "" {
		log.Printf("Using custom Census Bureau URL: %s", customURL)
		urls = append(urls, customURL)
	}

	// Check for custom URL from configuration
	if zg.config != nil && zg.config.CustomCensusBureauURL != "" {
		log.Printf("Using configured Census Bureau URL: %s", zg.config.CustomCensusBureauURL)
		urls = append(urls, zg.config.CustomCensusBureauURL)
	}

	// Default pattern with current and previous years
	basePattern := "https://www2.census.gov/geo/docs/maps-data/data/gazetteer/%d_Gazetteer/%d_Gaz_zcta_national.zip"

	// Try current year and 2 previous years
	for i := 0; i < 3; i++ {
		year := currentYear - i
		url := fmt.Sprintf(basePattern, year, year)
		urls = append(urls, url)
	}

	// Try alternative file naming patterns that Census Bureau might use
	altPatterns := []string{
		"https://www2.census.gov/geo/docs/maps-data/data/gazetteer/Gaz_zcta_national.zip",        // No year
		"https://www2.census.gov/geo/docs/maps-data/data/gazetteer/current/zcta_national.zip",    // Current folder
		"https://www2.census.gov/geo/docs/maps-data/data/gazetteer/latest/Gaz_zcta_national.zip", // Latest folder
		"https://www2.census.gov/geo/docs/maps-data/data/rel2020/zcta520/zcta520.zip",            // Alternative path
	}

	urls = append(urls, altPatterns...)
	return urls
}

// processCensusBureauData handles the actual data processing from any successful URL
func (zg *ZipGeocoder) processCensusBureauData(body io.Reader) error {
	// Read the ZIP file content
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read ZIP file content: %w", err)
	}

	// Open the ZIP archive
	zipReader, err := zip.NewReader(strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open ZIP archive: %w", err)
	}

	// Find the text file inside the ZIP
	var textFile *zip.File
	for _, file := range zipReader.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".txt") {
			textFile = file
			break
		}
	}

	if textFile == nil {
		return fmt.Errorf("no text file found in ZIP archive")
	}

	// Open and read the text file
	textReader, err := textFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open text file: %w", err)
	}
	defer textReader.Close()

	// Parse the tab-delimited Census Bureau format
	csvReader := csv.NewReader(textReader)
	csvReader.Comma = '\t' // Tab-delimited format

	// Begin transaction for bulk insert
	tx, err := zg.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO zip_coordinates (zip_code, latitude, longitude, city, state) 
		VALUES ($1, $2, $3, $4, $5) 
		ON CONFLICT (zip_code) DO UPDATE SET 
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			city = EXCLUDED.city,
			state = EXCLUDED.state,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	header := true

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: skipping malformed record: %v", err)
			continue
		}

		// Skip header row
		if header {
			header = false
			continue
		}

		// Census Bureau format: GEOID, ALAND, AWATER, ALAND_SQMI, AWATER_SQMI, INTPTLAT, INTPTLONG
		// We need: ZIP (GEOID), Latitude (INTPTLAT), Longitude (INTPTLONG)
		if len(record) < 7 {
			log.Printf("Warning: record has insufficient columns: %v", record)
			continue
		}

		zipCode := strings.TrimSpace(record[0]) // GEOID (ZIP code)
		latStr := strings.TrimSpace(record[5])  // INTPTLAT (Internal Point Latitude)
		lonStr := strings.TrimSpace(record[6])  // INTPTLONG (Internal Point Longitude)

		// Validate ZIP code format
		if len(zipCode) != 5 {
			continue
		}

		// Parse coordinates
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			log.Printf("Warning: invalid latitude for ZIP %s: %s", zipCode, latStr)
			continue
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			log.Printf("Warning: invalid longitude for ZIP %s: %s", zipCode, lonStr)
			continue
		}

		// Determine state from ZIP code (first digit gives rough geographic region)
		state := getStateFromZip(zipCode)
		city := "" // Census gazetteer doesn't include city names

		// Insert record
		_, err = stmt.Exec(zipCode, lat, lon, city, state)
		if err != nil {
			log.Printf("Warning: failed to insert ZIP %s: %v", zipCode, err)
			continue
		}

		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully loaded %d ZIP codes from Census Bureau", count)
	return nil
}

// getStateFromZip provides a rough state approximation based on ZIP code prefix
// This is a simplified mapping - in practice, ZIP boundaries can cross state lines
func getStateFromZip(zipCode string) string {
	if len(zipCode) < 1 {
		return ""
	}

	// Basic ZIP code to state mapping (first digit)
	switch zipCode[0] {
	case '0':
		return "MA" // Northeast (MA, CT, ME, NH, VT, RI)
	case '1':
		return "NY" // NY, PA, DE
	case '2':
		return "VA" // DC, MD, NC, SC, VA, WV
	case '3':
		return "FL" // AL, FL, GA, MS, TN
	case '4':
		return "KY" // IN, KY, MI, OH
	case '5':
		return "IA" // IA, MN, MT, ND, SD, WI
	case '6':
		return "TX" // IL, KS, MO, NE, TX
	case '7':
		return "TX" // AR, LA, OK, TX
	case '8':
		return "CO" // AZ, CO, ID, NM, NV, UT, WY
	case '9':
		return "CA" // AK, AS, CA, GU, HI, MH, FM, MP, PW, OR, WA
	default:
		return ""
	}
}

// loadFromBackupSource loads a minimal dataset of major cities if Census Bureau fails
func (zg *ZipGeocoder) loadFromBackupSource() error {
	log.Println("Loading backup ZIP code data for major cities...")

	// Major cities backup data - this ensures the system works even if external APIs fail
	backupData := []struct {
		zip   string
		lat   float64
		lon   float64
		city  string
		state string
	}{
		{"10001", 40.7505, -73.9934, "New York", "NY"},
		{"90210", 34.0901, -118.4065, "Beverly Hills", "CA"},
		{"02101", 42.3584, -71.0598, "Boston", "MA"},
		{"60601", 41.8781, -87.6298, "Chicago", "IL"},
		{"77001", 29.7604, -95.3698, "Houston", "TX"},
		{"85001", 33.4484, -112.0740, "Phoenix", "AZ"},
		{"19101", 39.9526, -75.1652, "Philadelphia", "PA"},
		{"92101", 32.7157, -117.1611, "San Diego", "CA"},
		{"75201", 32.7767, -96.7970, "Dallas", "TX"},
		{"95101", 37.3382, -121.8863, "San Jose", "CA"},
		{"78701", 30.2672, -97.7431, "Austin", "TX"},
		{"32801", 28.5383, -81.3792, "Orlando", "FL"},
		{"80201", 39.7392, -104.9903, "Denver", "CO"},
		{"20001", 38.9072, -77.0369, "Washington", "DC"},
		{"33101", 25.7617, -80.1918, "Miami", "FL"},
	}

	// Begin transaction
	tx, err := zg.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO zip_coordinates (zip_code, latitude, longitude, city, state) 
		VALUES ($1, $2, $3, $4, $5) 
		ON CONFLICT (zip_code) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, data := range backupData {
		_, err = stmt.Exec(data.zip, data.lat, data.lon, data.city, data.state)
		if err != nil {
			log.Printf("Warning: failed to insert backup ZIP %s: %v", data.zip, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully loaded %d backup ZIP codes", len(backupData))
	return nil
}
