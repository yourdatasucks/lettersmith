package reps

import (
	"database/sql"
	"time"
)

type Representative struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Title         string    `json:"title"`
	State         string    `json:"state"`
	District      *string   `json:"district,omitempty"`
	Party         *string   `json:"party,omitempty"`
	Email         *string   `json:"email,omitempty"`
	Phone         *string   `json:"phone,omitempty"`
	OfficeAddress *string   `json:"office_address,omitempty"`
	Website       *string   `json:"website,omitempty"`
	ExternalID    *string   `json:"external_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type OpenStatesResponse struct {
	Results []OpenStatesRep `json:"results"`
}

type OpenStatesRep struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Party       string             `json:"party"`
	CurrentRole *OpenStatesRole    `json:"current_role"`
	Email       string             `json:"email"`
	Links       []OpenStatesLink   `json:"links"`
	Offices     []OpenStatesOffice `json:"offices"`
}

type OpenStatesRole struct {
	Title             string `json:"title"`
	OrgClassification string `json:"org_classification"`
	District          string `json:"district"`
}

type OpenStatesLink struct {
	URL string `json:"url"`
}

type OpenStatesOffice struct {
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}
