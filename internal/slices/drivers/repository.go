package drivers

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const DriversGeoKey = "drivers:locations"
const LocationUpdatesChannel = "drivers:telemetry:updates"

type Repository struct {
	redis *redis.Client
}

func NewRepository(redis *redis.Client) *Repository {
	return &Repository{redis: redis}
}

func (r *Repository) UpdateLocation(ctx context.Context, update LocationUpdate) error {
	if r.redis == nil {
		return nil
	}

	if err := r.redis.GeoAdd(ctx, DriversGeoKey, &redis.GeoLocation{
		Name:      update.DriverID.String(),
		Latitude:  update.Latitude,
		Longitude: update.Longitude,
	}).Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(OrderTrackingEvent{
		OrderID:   update.OrderID,
		DriverID:  update.DriverID,
		Latitude:  update.Latitude,
		Longitude: update.Longitude,
		Timestamp: update.Timestamp,
	})
	if err == nil {
		_ = r.redis.Publish(ctx, LocationUpdatesChannel, payload).Err()
	}

	return nil
}

func (r *Repository) FindNearby(ctx context.Context, lat, lng, radiusKM float64, count int) ([]NearbyDriver, error) {
	if r.redis == nil {
		return []NearbyDriver{}, nil
	}

	locations, err := r.redis.GeoRadius(ctx, DriversGeoKey, lng, lat, &redis.GeoRadiusQuery{
		Radius:      radiusKM,
		Unit:        "km",
		WithCoord:   true,
		WithDist:    true,
		Count:       count,
		Sort:        "ASC",
	}).Result()

	if err != nil {
		return nil, err
	}

	var results []NearbyDriver
	for _, loc := range locations {
		driverID, err := uuid.Parse(loc.Name)
		if err != nil {
			continue
		}
		results = append(results, NearbyDriver{
			DriverID:   driverID,
			Latitude:   loc.Latitude,
			Longitude:  loc.Longitude,
			DistanceKM: loc.Dist,
		})
	}

	return results, nil
}
