package products

import (
	"time"

	"github.com/google/uuid"
)

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

type CreateProductRequest struct {
	StoreID     uuid.UUID      `json:"store_id"`
	CategoryID  *uuid.UUID     `json:"category_id,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	BasePrice   float64        `json:"base_price"`
	IsAvailable bool           `json:"is_available"`
	Attributes  map[string]any `json:"attributes"`
}

type UpdateProductRequest struct {
	CategoryID  *uuid.UUID      `json:"category_id,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	BasePrice   *float64        `json:"base_price,omitempty"`
	IsAvailable *bool           `json:"is_available,omitempty"`
	Attributes  *map[string]any `json:"attributes,omitempty"`
}
