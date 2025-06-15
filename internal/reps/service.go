package reps

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Service) GetUserRepresentatives(userZip string) ([]Representative, error) {
	query := `
		SELECT id, name, title, state, district, party, email, phone, 
		       office_address, website, external_id, created_at, updated_at
		FROM representatives 
		WHERE state = (
			SELECT state FROM zip_coordinates WHERE zip_code = $1 LIMIT 1
		)
		ORDER BY title, name
	`

	rows, err := s.db.Query(query, userZip)
	if err != nil {
		return nil, fmt.Errorf("failed to query representatives: %w", err)
	}
	defer rows.Close()

	var representatives []Representative
	for rows.Next() {
		var rep Representative
		err := rows.Scan(
			&rep.ID, &rep.Name, &rep.Title, &rep.State, &rep.District, &rep.Party,
			&rep.Email, &rep.Phone, &rep.OfficeAddress, &rep.Website, &rep.ExternalID,
			&rep.CreatedAt, &rep.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan representative: %w", err)
		}
		representatives = append(representatives, rep)
	}

	return representatives, nil
}

func (s *Service) SyncFromOpenStates(latitude, longitude float64, apiKey, userState string) error {
	url := fmt.Sprintf("https://v3.openstates.org/people.geo?lat=%f&lng=%f", latitude, longitude)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("User-Agent", "Lettersmith/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call OpenStates API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenStates API returned status %d", resp.StatusCode)
	}

	var apiResponse OpenStatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("failed to decode API response: %w", err)
	}

	for _, osRep := range apiResponse.Results {
		if err := s.upsertRepresentative(osRep, userState); err != nil {
			return fmt.Errorf("failed to store representative %s: %w", osRep.Name, err)
		}
	}

	return nil
}

func (s *Service) upsertRepresentative(osRep OpenStatesRep, userState string) error {
	var district, email, website, phone, address string

	party := osRep.Party

	if osRep.CurrentRole != nil {
		district = osRep.CurrentRole.District
	}

	email = osRep.Email

	if len(osRep.Links) > 0 {
		website = osRep.Links[0].URL
	}

	if len(osRep.Offices) > 0 {
		phone = osRep.Offices[0].Phone
		address = osRep.Offices[0].Address
	}

	state := userState
	title := ""
	if osRep.CurrentRole != nil {
		title = osRep.CurrentRole.Title
	}

	query := `
		INSERT INTO representatives (name, title, state, district, party, email, phone, office_address, website, external_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			title = EXCLUDED.title,
			state = EXCLUDED.state,
			district = EXCLUDED.district,
			party = EXCLUDED.party,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			office_address = EXCLUDED.office_address,
			website = EXCLUDED.website,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.Exec(query, osRep.Name, title, state,
		nullString(district), nullString(party), nullString(email),
		nullString(phone), nullString(address), nullString(website),
		nullString(osRep.ID))

	return err
}

func (s *Service) UpdateRepresentative(id int, updates map[string]interface{}) error {
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	allowedFields := map[string]bool{
		"name": true, "title": true, "district": true, "party": true,
		"email": true, "phone": true, "office_address": true, "website": true,
	}

	for field, value := range updates {
		if !allowedFields[field] {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	query := fmt.Sprintf("UPDATE representatives SET %s WHERE id = $%d",
		strings.Join(setParts, ", "), argIndex)
	args = append(args, id)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update representative: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("representative not found")
	}

	return nil
}

func (s *Service) DeleteRepresentative(id int) error {
	query := "DELETE FROM representatives WHERE id = $1"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete representative: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("representative not found")
	}

	return nil
}

func (s *Service) GetRepresentativeByID(id int) (*Representative, error) {
	query := `
		SELECT id, name, title, state, district, party, email, phone, 
		       office_address, website, external_id, created_at, updated_at
		FROM representatives WHERE id = $1
	`

	var rep Representative
	err := s.db.QueryRow(query, id).Scan(
		&rep.ID, &rep.Name, &rep.Title, &rep.State, &rep.District, &rep.Party,
		&rep.Email, &rep.Phone, &rep.OfficeAddress, &rep.Website, &rep.ExternalID,
		&rep.CreatedAt, &rep.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("representative not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get representative: %w", err)
	}

	return &rep, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func ExtractIDFromPath(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return 0, fmt.Errorf("invalid path format")
	}

	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format: %w", err)
	}

	return id, nil
}
