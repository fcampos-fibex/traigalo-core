package stores

import (
	"time"

	"github.com/google/uuid"
)

type Store struct {
	ID             uuid.UUID `json:"id"`
	OwnerID        uuid.UUID `json:"owner_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	LogoURL        string    `json:"logo_url"`
	CoverURL       string    `json:"cover_url"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	AddressLine    string    `json:"address_line"`
	CommissionRate float64   `json:"commission_rate"`
	IsOpen         bool      `json:"is_open"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Product struct {
	ID          uuid.UUID      `json:"id"`
	StoreID     uuid.UUID      `json:"store_id"`
	CategoryID  *uuid.UUID     `json:"category_id,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	BasePrice   float64        `json:"base_price"`
	IsAvailable bool           `json:"is_available"`
	Attributes  map[string]any `json:"attributes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type CreateStoreRequest struct {
	OwnerID        uuid.UUID `json:"owner_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	LogoURL        string    `json:"logo_url"`
	CoverURL       string    `json:"cover_url"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	AddressLine    string    `json:"address_line"`
	CommissionRate float64   `json:"commission_rate"`
}

type CreateProductRequest struct {
	CategoryID  *uuid.UUID     `json:"category_id,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	BasePrice   float64        `json:"base_price"`
	Attributes  map[string]any `json:"attributes"`
}
