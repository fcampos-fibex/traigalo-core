package dispatch

import (
	"time"

	"github.com/google/uuid"
)

type AssignDispatchRequest struct {
	OrderID uuid.UUID `json:"order_id"`
}

type DispatchOffer struct {
	OfferID      uuid.UUID `json:"offer_id"`
	OrderID      uuid.UUID `json:"order_id"`
	DriverID     uuid.UUID `json:"driver_id"`
	StoreName    string    `json:"store_name"`
	EarningsUSD  float64   `json:"earnings_usd"`
	ExpiresIn    int       `json:"expires_in"`
	CreatedAt    time.Time `json:"created_at"`
	CandidateIdx int       `json:"candidate_idx"`
}

type OfferActionRequest struct {
	Action string `json:"action"`
}

type DispatchOfferEvent struct {
	Event       string    `json:"event"`
	OfferID     uuid.UUID `json:"offer_id"`
	OrderID     uuid.UUID `json:"order_id"`
	DriverID    uuid.UUID `json:"driver_id"`
	StoreName   string    `json:"store_name"`
	EarningsUSD float64   `json:"earnings_usd"`
	ExpiresIn   int       `json:"expires_in"`
}
