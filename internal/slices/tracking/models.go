package tracking

import (
	"github.com/google/uuid"
)

type LatLng [2]float64

type Waypoint struct {
	Type        string  `json:"type"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Address     string  `json:"address,omitempty"`
	Description string  `json:"description,omitempty"`
}

type OrderRouteResponse struct {
	OrderID          uuid.UUID  `json:"order_id"`
	Status           string     `json:"status"`
	TotalDistanceKM  float64    `json:"total_distance_km"`
	EstimatedMinutes int        `json:"estimated_minutes"`
	Waypoints        []Waypoint `json:"waypoints"`
	Polyline         []LatLng   `json:"polyline"`
}
