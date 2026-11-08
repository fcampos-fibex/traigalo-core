package orders

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPlaced          OrderStatus = "placed"
	StatusAcceptedByStore OrderStatus = "accepted_by_store"
	StatusPreparing       OrderStatus = "preparing"
	StatusReadyForPickup  OrderStatus = "ready_for_pickup"
	StatusDriverAssigned  OrderStatus = "driver_assigned"
	StatusInTransit       OrderStatus = "in_transit"
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
	DeliveryFee       float64                  `json:"delivery_fee"`
	ServiceFee        float64                  `json:"service_fee"`
	Items             []CreateOrderItemRequest `json:"items"`
}

type OrderActionRequest struct {
	Action   string     `json:"action"`
	DriverID *uuid.UUID `json:"driver_id,omitempty"`
}
