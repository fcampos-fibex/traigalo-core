package drivers

import (
	"context"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const DriversGeoKey = "drivers:locations"

type Repository struct {
	redis *redis.Client
}

func NewRepository(redis *redis.Client) *Repository {
	return &Repository{redis: redis}
}

func (r *Repository) UpdateLocation(ctx context.Context, update LocationUpdate) error {
	return r.redis.GeoAdd(ctx, DriversGeoKey, &redis.GeoLocation{
		Name:      update.DriverID.String(),
		Latitude:  update.Latitude,
		Longitude: update.Longitude,
	}).Err()
}

func (r *Repository) FindNearby(ctx context.Context, lat, lng, radiusKM float64, count int) ([]NearbyDriver, error) {
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
