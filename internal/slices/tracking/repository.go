package tracking

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/traigalo/traigalo-core/internal/slices/orders"
)

type Repository struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewRepository(pool *pgxpool.Pool, redis *redis.Client) *Repository {
	return &Repository{
		pool:  pool,
		redis: redis,
	}
}

func (r *Repository) GetOrderRoute(ctx context.Context, orderID uuid.UUID) (*OrderRouteResponse, error) {
	query := `
		SELECT 
			o.status,
			COALESCE(o.driver_id, '00000000-0000-0000-0000-000000000000'::uuid),
			ST_Y(s.location::geometry) as store_lat, ST_X(s.location::geometry) as store_lng, s.name as store_name,
			ST_Y(o.delivery_location::geometry) as client_lat, ST_X(o.delivery_location::geometry) as client_lng, o.delivery_address
		FROM orders o
		JOIN stores s ON o.store_id = s.id
		WHERE o.id = $1
	`
	var status string
	var driverID uuid.UUID
	var storeLat, storeLng float64
	var storeName string
	var clientLat, clientLng float64
	var deliveryAddress string

	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&status, &driverID, &storeLat, &storeLng, &storeName,
		&clientLat, &clientLng, &deliveryAddress,
	)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	waypoints := []Waypoint{
		{
			Type:        "store",
			Latitude:    storeLat,
			Longitude:   storeLng,
			Description: storeName,
		},
	}

	driverLat, driverLng := storeLat, storeLng
	hasDriverPos := false
	if driverID != uuid.Nil && r.redis != nil {
		pos, err := r.redis.GeoPos(ctx, "drivers:locations", driverID.String()).Result()
		if err == nil && len(pos) > 0 && pos[0] != nil {
			driverLat = pos[0].Latitude
			driverLng = pos[0].Longitude
			hasDriverPos = true
		}
	}

	if hasDriverPos {
		waypoints = append([]Waypoint{
			{
				Type:        "driver",
				Latitude:    driverLat,
				Longitude:   driverLng,
				Description: "Repartidor en movimiento",
			},
		}, waypoints...)
	}

	waypoints = append(waypoints, Waypoint{
		Type:        "customer",
		Latitude:    clientLat,
		Longitude:   clientLng,
		Address:     deliveryAddress,
		Description: "Dirección de entrega",
	})

	polyline := generateInterpolatedPolyline(waypoints)

	totalDistanceKM := 0.0
	for i := 0; i < len(waypoints)-1; i++ {
		totalDistanceKM += orders.CalculateDistanceKM(
			waypoints[i].Latitude, waypoints[i].Longitude,
			waypoints[i+1].Latitude, waypoints[i+1].Longitude,
		)
	}

	estimatedMinutes := int(math.Round(totalDistanceKM * 3.0))
	if estimatedMinutes < 10 {
		estimatedMinutes = 10
	}

	return &OrderRouteResponse{
		OrderID:          orderID,
		Status:           status,
		TotalDistanceKM:  math.Round(totalDistanceKM*100) / 100,
		EstimatedMinutes: estimatedMinutes,
		Waypoints:        waypoints,
		Polyline:         polyline,
	}, nil
}

func generateInterpolatedPolyline(waypoints []Waypoint) []LatLng {
	if len(waypoints) == 0 {
		return []LatLng{}
	}
	polyline := []LatLng{}

	for i := 0; i < len(waypoints)-1; i++ {
		start := waypoints[i]
		end := waypoints[i+1]

		steps := 8
		for s := 0; s < steps; s++ {
			t := float64(s) / float64(steps)
			lat := start.Latitude + t*(end.Latitude-start.Latitude)
			lng := start.Longitude + t*(end.Longitude-start.Longitude)
			polyline = append(polyline, LatLng{math.Round(lat*100000) / 100000, math.Round(lng*100000) / 100000})
		}
	}

	last := waypoints[len(waypoints)-1]
	polyline = append(polyline, LatLng{math.Round(last.Latitude*100000) / 100000, math.Round(last.Longitude*100000) / 100000})

	return polyline
}
