package drivers

import (
	"time"

	"github.com/google/uuid"
)

type LocationUpdate struct {
	DriverID  uuid.UUID `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

type NearbyDriver struct {
	DriverID   uuid.UUID `json:"driver_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	DistanceKM float64   `json:"distance_km"`
}
