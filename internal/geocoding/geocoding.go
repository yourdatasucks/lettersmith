package geocoding

import (
	"database/sql"
	"fmt"
	"log"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
	City      string
	State     string
}

type ZipGeocoder struct {
	db     *sql.DB
	config *GeocodingConfig
}

type GeocodingConfig struct {
	CustomCensusBureauURL string
}

func NewZipGeocoder(db *sql.DB) *ZipGeocoder {
	return &ZipGeocoder{
		db:     db,
		config: &GeocodingConfig{},
	}
}

func NewZipGeocoderWithConfig(db *sql.DB, config *GeocodingConfig) *ZipGeocoder {
	return &ZipGeocoder{
		db:     db,
		config: config,
	}
}

// GetCoordinates returns the latitude and longitude for a given ZIP code
func (zg *ZipGeocoder) GetCoordinates(zipCode string) (*Coordinates, error) {
	// Clean ZIP code - take only first 5 digits
	if len(zipCode) > 5 {
		zipCode = zipCode[:5]
	}
	if len(zipCode) != 5 {
		return nil, fmt.Errorf("invalid ZIP code format: %s", zipCode)
	}

	var coords Coordinates
	query := `
		SELECT latitude, longitude, city, state 
		FROM zip_coordinates 
		WHERE zip_code = $1
	`

	err := zg.db.QueryRow(query, zipCode).Scan(
		&coords.Latitude,
		&coords.Longitude,
		&coords.City,
		&coords.State,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ZIP code %s not found", zipCode)
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &coords, nil
}

// CheckDataFreshness returns the number of ZIP codes in the database
// and can be used to determine if data needs to be refreshed
func (zg *ZipGeocoder) CheckDataFreshness() (int, error) {
	var count int
	err := zg.db.QueryRow("SELECT COUNT(*) FROM zip_coordinates").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count ZIP coordinates: %w", err)
	}
	return count, nil
}

// LoadZipData loads ZIP code coordinate data into the database
// It tries the Census Bureau source first, then falls back to backup data
func (zg *ZipGeocoder) LoadZipData() error {
	// Check if we already have data and don't need to reload
	var count int
	err := zg.db.QueryRow("SELECT COUNT(*) FROM zip_coordinates").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count > 1000 { // If we have a reasonable amount of data, skip loading
		log.Printf("ZIP coordinate database already contains %d records, skipping update", count)
		return nil
	}

	log.Println("Loading ZIP code coordinate data...")

	// Try official Census Bureau source first (our agreed-upon authoritative source)
	err = zg.loadFromCensusBureau()
	if err != nil {
		log.Printf("Failed to load from Census Bureau: %v", err)
		log.Println("Falling back to backup data...")

		// Fall back to backup data for major cities
		err = zg.loadFromBackupSource()
		if err != nil {
			return fmt.Errorf("failed to load backup ZIP data: %w", err)
		}
	}

	// Verify we loaded data successfully
	err = zg.db.QueryRow("SELECT COUNT(*) FROM zip_coordinates").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify data load: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("no ZIP coordinate data was loaded")
	}

	log.Printf("ZIP coordinate database now contains %d records", count)
	return nil
}

// The following method implementations are in datasources.go:
// - loadFromCensusBureau()
// - loadFromBackupSource()
