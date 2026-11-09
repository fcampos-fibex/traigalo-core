package payments

import (
	"time"

	"github.com/google/uuid"
)

type PaymentReceipt struct {
	ID              uuid.UUID  `json:"id"`
	OrderID         uuid.UUID  `json:"order_id"`
	PaymentMethod   string     `json:"payment_method"`
	ReferenceNumber string     `json:"reference_number"`
	IssuingBank     string     `json:"issuing_bank"`
	ReceivingBank   string     `json:"receiving_bank"`
	AmountUSD       float64    `json:"amount_usd"`
	AmountBS        float64    `json:"amount_bs"`
	BSRate          float64    `json:"bs_rate"`
	Status          string     `json:"status"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CreateReceiptRequest struct {
	OrderID         uuid.UUID `json:"order_id"`
	PaymentMethod   string    `json:"payment_method"`
	ReferenceNumber string    `json:"reference_number"`
	IssuingBank     string    `json:"issuing_bank"`
	ReceivingBank   string    `json:"receiving_bank"`
	AmountUSD       float64   `json:"amount_usd"`
	AmountBS        float64   `json:"amount_bs"`
	BSRate          float64   `json:"bs_rate"`
	Notes           string    `json:"notes,omitempty"`
}

type ReceiptActionRequest struct {
	Action string `json:"action"`
	Notes  string `json:"notes,omitempty"`
}
