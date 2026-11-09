package orders

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPlaced          OrderStatus = "placed"
	StatusAcceptedByStore OrderStatus = "accepted_by_store"
	StatusInKitchen       OrderStatus = "in_kitchen"
	StatusPreparing       OrderStatus = "in_kitchen"
	StatusReadyForPickup  OrderStatus = "ready_for_pickup"
	StatusDriverAssigned  OrderStatus = "driver_assigned"
	StatusDriverInTransit OrderStatus = "driver_in_transit"
	StatusInTransit       OrderStatus = "driver_in_transit"
	StatusDelivered       OrderStatus = "delivered"
	StatusCancelled       OrderStatus = "cancelled"
)

type OrderItem struct {
	ID        uuid.UUID      `json:"id"`
	OrderID   uuid.UUID      `json:"order_id"`
	ProductID uuid.UUID      `json:"product_id"`
	Quantity  int            `json:"quantity"`
	UnitPrice float64        `json:"unit_price"`
	Subtotal  float64        `json:"subtotal"`
	Options   map[string]any `json:"options"`
}

type Order struct {
	ID                uuid.UUID   `json:"id"`
	OrderNumber       int         `json:"order_number"`
	CustomerID        uuid.UUID   `json:"customer_id"`
	StoreID           uuid.UUID   `json:"store_id"`
	DriverID          *uuid.UUID  `json:"driver_id,omitempty"`
	Status            OrderStatus `json:"status"`
	Subtotal          float64     `json:"subtotal"`
	DeliveryFee       float64     `json:"delivery_fee"`
	ServiceFee        float64     `json:"service_fee"`
	TipUSD            float64     `json:"tip_usd"`
	CommissionFee     float64     `json:"commission_fee"`
	Total             float64     `json:"total"`
	TotalUSD          float64     `json:"total_usd"`
	BSRate            float64     `json:"bs_rate"`
	TotalBS           float64     `json:"total_bs"`
	DeliveryLatitude  float64     `json:"delivery_latitude"`
	DeliveryLongitude float64     `json:"delivery_longitude"`
	DeliveryAddress   string      `json:"delivery_address"`
	PaymentMethod     string      `json:"payment_method"`
	PaymentStatus     string      `json:"payment_status"`
	PlacedAt          time.Time   `json:"placed_at"`
	DeliveredAt       *time.Time  `json:"delivered_at,omitempty"`
	Items             []OrderItem `json:"items,omitempty"`
}

type CreateOrderItemRequest struct {
	ProductID uuid.UUID      `json:"product_id"`
	Quantity  int            `json:"quantity"`
	UnitPrice float64        `json:"unit_price,omitempty"`
	Options   map[string]any `json:"options"`
}

type CreateOrderRequest struct {
	CustomerID        uuid.UUID                `json:"customer_id"`
	StoreID           uuid.UUID                `json:"store_id"`
	DeliveryLatitude  float64                  `json:"delivery_latitude"`
	DeliveryLongitude float64                  `json:"delivery_longitude"`
	DeliveryAddress   string                   `json:"delivery_address"`
	PaymentMethod     string                   `json:"payment_method"`
	DeliveryFee       float64                  `json:"delivery_fee,omitempty"`
	ServiceFee        float64                  `json:"service_fee,omitempty"`
	TipUSD            float64                  `json:"tip_usd,omitempty"`
	Items             []CreateOrderItemRequest `json:"items"`
}

type OrderActionRequest struct {
	Action   string     `json:"action"`
	DriverID *uuid.UUID `json:"driver_id,omitempty"`
}

type DeliveryEstimateResponse struct {
	DistanceKM       float64 `json:"distance_km"`
	DeliveryFeeUSD   float64 `json:"delivery_fee_usd"`
	DeliveryFeeBS    float64 `json:"delivery_fee_bs"`
	EstimatedMinutes int     `json:"estimated_minutes"`
}
