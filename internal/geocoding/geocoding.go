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


func (zg *ZipGeocoder) GetCoordinates(zipCode string) (*Coordinates, error) {
	
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



func (zg *ZipGeocoder) CheckDataFreshness() (int, error) {
	var count int
	err := zg.db.QueryRow("SELECT COUNT(*) FROM zip_coordinates").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count ZIP coordinates: %w", err)
	}
	return count, nil
}



func (zg *ZipGeocoder) LoadZipData() error {
	
	var count int
	err := zg.db.QueryRow("SELECT COUNT(*) FROM zip_coordinates").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count > 1000 { 
		log.Printf("ZIP coordinate database already contains %d records, skipping update", count)
		return nil
	}

	log.Println("Loading ZIP code coordinate data...")

	
	err = zg.loadFromCensusBureau()
	if err != nil {
		log.Printf("Failed to load from Census Bureau: %v", err)
		log.Println("Falling back to backup data...")

		
		err = zg.loadFromBackupSource()
		if err != nil {
			return fmt.Errorf("failed to load backup ZIP data: %w", err)
		}
	}

	
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




