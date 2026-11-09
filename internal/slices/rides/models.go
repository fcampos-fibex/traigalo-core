package rides

import (
	"time"

	"github.com/google/uuid"
)

type RideStatus string

const (
	StatusRequested      RideStatus = "requested"
	StatusDriverAssigned RideStatus = "driver_assigned"
	StatusInTrip         RideStatus = "in_trip"
	StatusCompleted      RideStatus = "completed"
	StatusCancelled      RideStatus = "cancelled"
)

type Ride struct {
	ID              uuid.UUID  `json:"id"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	DriverID        *uuid.UUID `json:"driver_id,omitempty"`
	VehicleType     string     `json:"vehicle_type"`
	Status          RideStatus `json:"status"`
	PickupLatitude  float64    `json:"pickup_latitude"`
	PickupLongitude float64    `json:"pickup_longitude"`
	PickupAddress   string     `json:"pickup_address"`
	DropoffLatitude float64    `json:"dropoff_latitude"`
	DropoffLongitude float64   `json:"dropoff_longitude"`
	DropoffAddress  string     `json:"dropoff_address"`
	DistanceKM      float64    `json:"distance_km"`
	FareUSD         float64    `json:"fare_usd"`
	BSRate          float64    `json:"bs_rate"`
	FareBS          float64    `json:"fare_bs"`
	PaymentMethod   string     `json:"payment_method"`
	PaymentStatus   string     `json:"payment_status"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

type CreateRideRequest struct {
	CustomerID       uuid.UUID `json:"customer_id,omitempty"`
	VehicleType      string    `json:"vehicle_type"`
	PickupLatitude   float64   `json:"pickup_latitude"`
	PickupLongitude  float64   `json:"pickup_longitude"`
	PickupAddress    string    `json:"pickup_address"`
	DropoffLatitude  float64   `json:"dropoff_latitude"`
	DropoffLongitude float64   `json:"dropoff_longitude"`
	DropoffAddress   string    `json:"dropoff_address"`
	PaymentMethod    string    `json:"payment_method,omitempty"`
}

type RideActionRequest struct {
	Action   string     `json:"action"`
	DriverID *uuid.UUID `json:"driver_id,omitempty"`
}
