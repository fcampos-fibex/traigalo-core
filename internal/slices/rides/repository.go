package rides

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/traigalo/traigalo-core/internal/slices/drivers"
	"github.com/traigalo/traigalo-core/internal/slices/orders"
	"github.com/traigalo/traigalo-core/internal/slices/rates"
)

type Repository struct {
	pool         *pgxpool.Pool
	redis        *redis.Client
	ratesService *rates.Service
}

func NewRepository(pool *pgxpool.Pool, redis *redis.Client, ratesService *rates.Service) *Repository {
	return &Repository{
		pool:         pool,
		redis:        redis,
		ratesService: ratesService,
	}
}

func CalculateRideFare(vehicleType string, distanceKM float64) float64 {
	baseFare := 2.50
	perKM := 0.80

	switch vehicleType {
	case "moto":
		baseFare = 1.50
		perKM = 0.50
	case "comfort":
		baseFare = 4.00
		perKM = 1.20
	case "cargo":
		baseFare = 5.00
		perKM = 1.50
	default:
		vehicleType = "standard"
		baseFare = 2.50
		perKM = 0.80
	}

	fare := baseFare + (distanceKM * perKM)
	if fare < baseFare {
		fare = baseFare
	}
	return math.Round(fare*100) / 100
}

func (r *Repository) Create(ctx context.Context, req CreateRideRequest) (*Ride, error) {
	distKM := orders.CalculateDistanceKM(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)
	fareUSD := CalculateRideFare(req.VehicleType, distKM)

	var bsRate float64 = 896.12
	if r.ratesService != nil {
		if rate, err := r.ratesService.GetOfficialRate(ctx); err == nil && rate > 0 {
			bsRate = rate
		}
	}
	fareBS := math.Round(fareUSD*bsRate*100) / 100

	if req.PaymentMethod == "" {
		req.PaymentMethod = "cash"
	}

	if req.CustomerID == uuid.Nil {
		req.CustomerID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	}

	query := `
		INSERT INTO rides (
			customer_id, vehicle_type, status,
			pickup_location, pickup_address, dropoff_location, dropoff_address,
			distance_km, fare_usd, bs_rate, fare_bs, payment_method, payment_status
		) VALUES (
			$1, $2, 'requested',
			ST_SetSRID(ST_MakePoint($3, $4), 4326), $5,
			ST_SetSRID(ST_MakePoint($6, $7), 4326), $8,
			$9, $10, $11, $12, $13, 'pending'
		)
		RETURNING id, created_at
	`
	var rd Ride
	rd.CustomerID = req.CustomerID
	rd.VehicleType = req.VehicleType
	rd.Status = StatusRequested
	rd.PickupLatitude = req.PickupLatitude
	rd.PickupLongitude = req.PickupLongitude
	rd.PickupAddress = req.PickupAddress
	rd.DropoffLatitude = req.DropoffLatitude
	rd.DropoffLongitude = req.DropoffLongitude
	rd.DropoffAddress = req.DropoffAddress
	rd.DistanceKM = math.Round(distKM*100) / 100
	rd.FareUSD = fareUSD
	rd.BSRate = bsRate
	rd.FareBS = fareBS
	rd.PaymentMethod = req.PaymentMethod
	rd.PaymentStatus = "pending"

	err := r.pool.QueryRow(
		ctx, query,
		req.CustomerID, req.VehicleType,
		req.PickupLongitude, req.PickupLatitude, req.PickupAddress,
		req.DropoffLongitude, req.DropoffLatitude, req.DropoffAddress,
		rd.DistanceKM, fareUSD, bsRate, fareBS, req.PaymentMethod,
	).Scan(&rd.ID, &rd.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create ride: %w", err)
	}

	if r.redis != nil {
		offerKey := fmt.Sprintf("ride:dispatch:%s:offer", rd.ID.String())
		offerData, _ := json.Marshal(map[string]any{
			"event":            "ride_offer",
			"ride_id":          rd.ID,
			"vehicle_type":     rd.VehicleType,
			"pickup_address":   rd.PickupAddress,
			"dropoff_address":  rd.DropoffAddress,
			"distance_km":      rd.DistanceKM,
			"earnings_usd":     rd.FareUSD,
			"pickup_latitude":  rd.PickupLatitude,
			"pickup_longitude": rd.PickupLongitude,
			"expires_in":       45,
		})
		_ = r.redis.Set(ctx, offerKey, offerData, 45*time.Second).Err()
		_ = r.redis.Publish(ctx, drivers.LocationUpdatesChannel, offerData).Err()
	}

	return &rd, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Ride, error) {
	query := `
		SELECT 
			id, customer_id, driver_id, vehicle_type, status,
			ST_Y(pickup_location::geometry), ST_X(pickup_location::geometry), pickup_address,
			ST_Y(dropoff_location::geometry), ST_X(dropoff_location::geometry), dropoff_address,
			distance_km, fare_usd, bs_rate, fare_bs, payment_method, payment_status,
			created_at, started_at, completed_at
		FROM rides
		WHERE id = $1
	`
	var rd Ride
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rd.ID, &rd.CustomerID, &rd.DriverID, &rd.VehicleType, &rd.Status,
		&rd.PickupLatitude, &rd.PickupLongitude, &rd.PickupAddress,
		&rd.DropoffLatitude, &rd.DropoffLongitude, &rd.DropoffAddress,
		&rd.DistanceKM, &rd.FareUSD, &rd.BSRate, &rd.FareBS,
		&rd.PaymentMethod, &rd.PaymentStatus, &rd.CreatedAt, &rd.StartedAt, &rd.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rd, nil
}

func (r *Repository) ExecuteAction(ctx context.Context, id uuid.UUID, action RideActionRequest) error {
	var updateQuery string
	var args []any

	switch action.Action {
	case "accept":
		if action.DriverID == nil {
			return fmt.Errorf("driver_id is required to accept ride")
		}
		updateQuery = "UPDATE rides SET status = 'driver_assigned', driver_id = $2 WHERE id = $1 AND status = 'requested'"
		args = []any{id, *action.DriverID}

	case "start_trip", "start":
		now := time.Now().UTC()
		updateQuery = "UPDATE rides SET status = 'in_trip', started_at = $2 WHERE id = $1 AND status = 'driver_assigned'"
		args = []any{id, now}

	case "complete":
		now := time.Now().UTC()
		updateQuery = "UPDATE rides SET status = 'completed', payment_status = 'paid', completed_at = $2 WHERE id = $1 AND status = 'in_trip'"
		args = []any{id, now}

	case "cancel":
		updateQuery = "UPDATE rides SET status = 'cancelled' WHERE id = $1 AND status NOT IN ('completed', 'cancelled')"
		args = []any{id}

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}

	cmd, err := r.pool.Exec(ctx, updateQuery, args...)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if r.redis != nil {
		eventData, _ := json.Marshal(map[string]any{
			"event":   "ride_status_updated",
			"ride_id": id,
			"action":  action.Action,
		})
		_ = r.redis.Publish(ctx, drivers.LocationUpdatesChannel, eventData).Err()
	}

	return nil
}
