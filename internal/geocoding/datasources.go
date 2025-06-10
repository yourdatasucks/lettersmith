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

func (zg *ZipGeocoder) loadFromCensusBureau() error {
	log.Println("Downloading ZIP code data from US Census Bureau...")

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

		log.Printf("Successfully connected to Census Bureau: %s", url)
		return zg.processCensusBureauData(resp.Body)
	}

	return fmt.Errorf("all Census Bureau URLs failed, last error: %w", lastErr)
}

func (zg *ZipGeocoder) generateCensusBureauURLs(currentYear int) []string {
	var urls []string

	if customURL := os.Getenv("CENSUS_BUREAU_URL"); customURL != "" {
		log.Printf("Using custom Census Bureau URL: %s", customURL)
		urls = append(urls, customURL)
	}

	if zg.config != nil && zg.config.CustomCensusBureauURL != "" {
		log.Printf("Using configured Census Bureau URL: %s", zg.config.CustomCensusBureauURL)
		urls = append(urls, zg.config.CustomCensusBureauURL)
	}

	basePattern := "https://www2.census.gov/geo/docs/maps-data/data/gazetteer/%d_Gazetteer/%d_Gaz_zcta_national.zip"

	for i := 0; i < 3; i++ {
		year := currentYear - i
		url := fmt.Sprintf(basePattern, year, year)
		urls = append(urls, url)
	}

	altPatterns := []string{
		"https://www2.census.gov/geo/docs/maps-data/data/gazetteer/Gaz_zcta_national.zip",
		"https://www2.census.gov/geo/docs/maps-data/data/gazetteer/current/zcta_national.zip",
		"https://www2.census.gov/geo/docs/maps-data/data/gazetteer/latest/Gaz_zcta_national.zip",
		"https://www2.census.gov/geo/docs/maps-data/data/rel2020/zcta520/zcta520.zip",
	}

	urls = append(urls, altPatterns...)
	return urls
}

func (zg *ZipGeocoder) processCensusBureauData(body io.Reader) error {

	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read ZIP file content: %w", err)
	}

	zipReader, err := zip.NewReader(strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open ZIP archive: %w", err)
	}

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

	textReader, err := textFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open text file: %w", err)
	}
	defer textReader.Close()

	csvReader := csv.NewReader(textReader)
	csvReader.Comma = '\t'

	tx, err := zg.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

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

		if header {
			header = false
			continue
		}

		if len(record) < 7 {
			log.Printf("Warning: record has insufficient columns: %v", record)
			continue
		}

		zipCode := strings.TrimSpace(record[0])
		latStr := strings.TrimSpace(record[5])
		lonStr := strings.TrimSpace(record[6])

		if len(zipCode) != 5 {
			continue
		}

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

		state := getStateFromZip(zipCode)
		city := ""

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

func getStateFromZip(zipCode string) string {
	if len(zipCode) < 3 {
		return ""
	}

	if strings.HasPrefix(zipCode, "006") || strings.HasPrefix(zipCode, "007") || strings.HasPrefix(zipCode, "009") {
		return "PR"
	}

	if strings.HasPrefix(zipCode, "008") {
		return "VI"
	}

	if strings.HasPrefix(zipCode, "969") {
		return "GU"
	}

	if strings.HasPrefix(zipCode, "967") || strings.HasPrefix(zipCode, "968") {
		return "HI"
	}

	switch zipCode[0] {
	case '0':

		if zipCode >= "01000" && zipCode <= "02799" {
			return "MA"
		}
		if zipCode >= "02800" && zipCode <= "02999" {
			return "RI"
		}
		if zipCode >= "03000" && zipCode <= "03999" {
			return "NH"
		}
		if zipCode >= "04000" && zipCode <= "04999" {
			return "ME"
		}
		if zipCode >= "05000" && zipCode <= "05999" {
			return "VT"
		}
		if zipCode >= "06000" && zipCode <= "06999" {
			return "CT"
		}
		if zipCode >= "07000" && zipCode <= "08999" {
			return "NJ"
		}
		return "CT"
	case '1':
		if zipCode >= "10000" && zipCode <= "14999" {
			return "NY"
		}
		if zipCode >= "15000" && zipCode <= "19699" {
			return "PA"
		}
		if zipCode >= "19700" && zipCode <= "19999" {
			return "DE"
		}
		return "PA"
	case '2':
		if zipCode >= "20000" && zipCode <= "20599" {
			return "DC"
		}
		if zipCode >= "20600" && zipCode <= "21999" {
			return "MD"
		}
		if zipCode >= "22000" && zipCode <= "24699" {
			return "VA"
		}
		if zipCode >= "25000" && zipCode <= "26999" {
			return "WV"
		}
		if zipCode >= "27000" && zipCode <= "28999" {
			return "NC"
		}
		if zipCode >= "29000" && zipCode <= "29999" {
			return "SC"
		}
		return "NC"
	case '3':
		if zipCode >= "30000" && zipCode <= "31999" {
			return "GA"
		}
		if zipCode >= "32000" && zipCode <= "34999" {
			return "FL"
		}
		if zipCode >= "35000" && zipCode <= "36999" {
			return "AL"
		}
		if zipCode >= "37000" && zipCode <= "38599" {
			return "TN"
		}
		if zipCode >= "38600" && zipCode <= "39799" {
			return "MS"
		}
		return "AL"
	case '4':
		if zipCode >= "40000" && zipCode <= "42799" {
			return "KY"
		}
		if zipCode >= "43000" && zipCode <= "45999" {
			return "OH"
		}
		if zipCode >= "46000" && zipCode <= "47999" {
			return "IN"
		}
		return "MI"
	case '5':
		if zipCode >= "50000" && zipCode <= "52999" {
			return "IA"
		}
		if zipCode >= "53000" && zipCode <= "54999" {
			return "WI"
		}
		if zipCode >= "55000" && zipCode <= "56799" {
			return "MN"
		}
		if zipCode >= "57000" && zipCode <= "57999" {
			return "SD"
		}
		if zipCode >= "58000" && zipCode <= "58899" {
			return "ND"
		}
		if zipCode >= "59000" && zipCode <= "59999" {
			return "MT"
		}
		return "SD"
	case '6':
		if zipCode >= "60000" && zipCode <= "62999" {
			return "IL"
		}
		if zipCode >= "63000" && zipCode <= "65999" {
			return "MO"
		}
		if zipCode >= "66000" && zipCode <= "67999" {
			return "KS"
		}
		if zipCode >= "68000" && zipCode <= "69399" {
			return "NE"
		}
		return "KS"
	case '7':
		if zipCode >= "70000" && zipCode <= "71499" {
			return "LA"
		}
		if zipCode >= "72000" && zipCode <= "72999" {
			return "AR"
		}
		if zipCode >= "73000" && zipCode <= "74999" {
			return "OK"
		}
		return "TX"
	case '8':
		if zipCode >= "80000" && zipCode <= "81999" {
			return "CO"
		}
		if zipCode >= "82000" && zipCode <= "83199" {
			return "WY"
		}
		if zipCode >= "83200" && zipCode <= "83899" {
			return "ID"
		}
		if zipCode >= "84000" && zipCode <= "84999" {
			return "UT"
		}
		if zipCode >= "85000" && zipCode <= "86999" {
			return "AZ"
		}
		if zipCode >= "87000" && zipCode <= "88999" {
			return "NM"
		}
		if zipCode >= "89000" && zipCode <= "89999" {
			return "NV"
		}
		return "NM"
	case '9':
		if zipCode >= "90000" && zipCode <= "96199" {
			return "CA"
		}
		if zipCode >= "96700" && zipCode <= "96999" {
			return "HI"
		}
		if zipCode >= "97000" && zipCode <= "97999" {
			return "OR"
		}
		if zipCode >= "98000" && zipCode <= "99999" {
			return "WA"
		}
		return "AK"
	default:
		return ""
	}
}

func (zg *ZipGeocoder) loadFromBackupSource() error {
	log.Println("Loading backup ZIP code data for major cities...")

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
