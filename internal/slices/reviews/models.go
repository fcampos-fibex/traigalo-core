package reviews

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID         uuid.UUID  `json:"id"`
	OrderID    uuid.UUID  `json:"order_id"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
	StoreID    uuid.UUID  `json:"store_id"`
	DriverID   *uuid.UUID `json:"driver_id,omitempty"`
	Rating     int        `json:"rating"`
	Comment    string     `json:"comment"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateReviewRequest struct {
	OrderID    uuid.UUID  `json:"order_id"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
	StoreID    uuid.UUID  `json:"store_id"`
	DriverID   *uuid.UUID `json:"driver_id,omitempty"`
	Rating     int        `json:"rating"`
	Comment    string     `json:"comment"`
}
