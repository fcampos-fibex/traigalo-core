package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/traigalo/traigalo-core/internal/slices/drivers"
)

const (
	DriversLocationsKey  = "drivers:locations"
	OfferTTL             = 45 * time.Second
	TelemetryChannel     = drivers.LocationUpdatesChannel
	CandidateListKeyTmpl = "order:dispatch:%s:candidates"
	OfferKeyTmpl         = "order:dispatch:%s:offer"
	OfferLookupKeyTmpl   = "dispatch:offers:%s"
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

func (r *Repository) AssignNearest(ctx context.Context, orderID uuid.UUID) (*DispatchOffer, error) {
	var storeName string
	var storeLat, storeLng float64
	var deliveryFee, tipUSD float64

	orderQuery := `
		SELECT s.name, ST_Y(s.location::geometry), ST_X(s.location::geometry), o.delivery_fee, COALESCE(o.tip_usd, 0)
		FROM orders o
		JOIN stores s ON o.store_id = s.id
		WHERE o.id = $1 AND o.status IN ('placed', 'accepted_by_store', 'in_kitchen', 'preparing', 'ready_for_pickup')
	`
	err := r.pool.QueryRow(ctx, orderQuery, orderID).Scan(&storeName, &storeLat, &storeLng, &deliveryFee, &tipUSD)
	if err != nil {
		return nil, fmt.Errorf("order not eligible or not found: %w", err)
	}

	driverIDs := []string{}
	if r.redis != nil {
		locations, err := r.redis.GeoRadius(ctx, DriversLocationsKey, storeLng, storeLat, &redis.GeoRadiusQuery{
			Radius:    5.0,
			Unit:      "km",
			Sort:      "ASC",
			WithCoord: true,
			WithDist:  true,
			Count:     10,
		}).Result()
		if err == nil {
			for _, loc := range locations {
				driverIDs = append(driverIDs, loc.Name)
			}
		}
	}

	if len(driverIDs) == 0 {
		var demoDriverID string
		err := r.pool.QueryRow(ctx, "SELECT id::text FROM users WHERE role = 'driver' AND is_active = true LIMIT 1").Scan(&demoDriverID)
		if err == nil && demoDriverID != "" {
			driverIDs = append(driverIDs, demoDriverID)
		}
	}

	if len(driverIDs) == 0 {
		return nil, fmt.Errorf("no drivers available within 5km")
	}

	earningsUSD := deliveryFee + tipUSD
	if earningsUSD <= 0 {
		earningsUSD = 2.50
	}

	candidateKey := fmt.Sprintf(CandidateListKeyTmpl, orderID.String())
	if r.redis != nil {
		_ = r.redis.Del(ctx, candidateKey).Err()
		for _, did := range driverIDs {
			_ = r.redis.RPush(ctx, candidateKey, did).Err()
		}
		_ = r.redis.Expire(ctx, candidateKey, 10*time.Minute).Err()
	}

	return r.createOffer(ctx, orderID, storeName, earningsUSD, 0)
}

func (r *Repository) createOffer(ctx context.Context, orderID uuid.UUID, storeName string, earningsUSD float64, candidateIdx int) (*DispatchOffer, error) {
	candidateKey := fmt.Sprintf(CandidateListKeyTmpl, orderID.String())
	var selectedDriverStr string

	if r.redis != nil {
		driverIDVal, err := r.redis.LIndex(ctx, candidateKey, int64(candidateIdx)).Result()
		if err != nil || driverIDVal == "" {
			return nil, fmt.Errorf("no more candidate drivers available")
		}
		selectedDriverStr = driverIDVal
	} else {
		selectedDriverStr = "33333333-3333-3333-3333-333333333333"
	}

	driverUUID, err := uuid.Parse(selectedDriverStr)
	if err != nil {
		return nil, fmt.Errorf("invalid candidate driver uuid: %w", err)
	}

	offer := &DispatchOffer{
		OfferID:      uuid.New(),
		OrderID:      orderID,
		DriverID:     driverUUID,
		StoreName:    storeName,
		EarningsUSD:  earningsUSD,
		ExpiresIn:    45,
		CreatedAt:    time.Now().UTC(),
		CandidateIdx: candidateIdx,
	}

	offerJSON, _ := json.Marshal(offer)

	if r.redis != nil {
		orderOfferKey := fmt.Sprintf(OfferKeyTmpl, orderID.String())
		offerLookupKey := fmt.Sprintf(OfferLookupKeyTmpl, offer.OfferID.String())

		_ = r.redis.Set(ctx, orderOfferKey, offerJSON, OfferTTL).Err()
		_ = r.redis.Set(ctx, offerLookupKey, offerJSON, OfferTTL).Err()

		eventPayload, _ := json.Marshal(DispatchOfferEvent{
			Event:       "dispatch_offer",
			OfferID:     offer.OfferID,
			OrderID:     offer.OrderID,
			DriverID:    offer.DriverID,
			StoreName:   offer.StoreName,
			EarningsUSD: offer.EarningsUSD,
			ExpiresIn:   offer.ExpiresIn,
		})
		_ = r.redis.Publish(ctx, TelemetryChannel, eventPayload).Err()
		_ = r.redis.Publish(ctx, fmt.Sprintf("drivers:%s:dispatch", offer.DriverID.String()), eventPayload).Err()
	}

	return offer, nil
}

func (r *Repository) ExecuteOfferAction(ctx context.Context, offerID uuid.UUID, action OfferActionRequest) (*DispatchOffer, error) {
	offerLookupKey := fmt.Sprintf(OfferLookupKeyTmpl, offerID.String())
	var offer DispatchOffer

	if r.redis != nil {
		offerRaw, err := r.redis.Get(ctx, offerLookupKey).Bytes()
		if err != nil {
			return nil, fmt.Errorf("dispatch offer expired or not found")
		}
		if err := json.Unmarshal(offerRaw, &offer); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("dispatch offer not found")
	}

	switch action.Action {
	case "accept":
		assignQuery := `
			UPDATE orders 
			SET status = 'driver_assigned', driver_id = $1 
			WHERE id = $2 AND status IN ('placed', 'accepted_by_store', 'in_kitchen', 'preparing', 'ready_for_pickup')
		`
		cmd, err := r.pool.Exec(ctx, assignQuery, offer.DriverID, offer.OrderID)
		if err != nil {
			return nil, err
		}
		if cmd.RowsAffected() == 0 {
			return nil, pgx.ErrNoRows
		}

		if r.redis != nil {
			orderOfferKey := fmt.Sprintf(OfferKeyTmpl, offer.OrderID.String())
			candidateKey := fmt.Sprintf(CandidateListKeyTmpl, offer.OrderID.String())
			_ = r.redis.Del(ctx, offerLookupKey).Err()
			_ = r.redis.Del(ctx, orderOfferKey).Err()
			_ = r.redis.Del(ctx, candidateKey).Err()

			notifyPayload, _ := json.Marshal(map[string]any{
				"event":     "driver_assigned",
				"order_id":  offer.OrderID,
				"driver_id": offer.DriverID,
				"status":    "driver_assigned",
			})
			_ = r.redis.Publish(ctx, TelemetryChannel, notifyPayload).Err()
		}

		return &offer, nil

	case "reject":
		if r.redis != nil {
			orderOfferKey := fmt.Sprintf(OfferKeyTmpl, offer.OrderID.String())
			_ = r.redis.Del(ctx, offerLookupKey).Err()
			_ = r.redis.Del(ctx, orderOfferKey).Err()
		}

		nextOffer, err := r.createOffer(ctx, offer.OrderID, offer.StoreName, offer.EarningsUSD, offer.CandidateIdx+1)
		if err != nil {
			return nil, fmt.Errorf("offer rejected, no more candidate drivers: %w", err)
		}
		return nextOffer, nil

	default:
		return nil, fmt.Errorf("unknown action: %s (must be 'accept' or 'reject')", action.Action)
	}
}
