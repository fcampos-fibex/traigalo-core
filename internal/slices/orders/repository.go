package orders

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/traigalo/traigalo-core/internal/slices/rates"
)

type Repository struct {
	pool         *pgxpool.Pool
	ratesService *rates.Service
	redis        *redis.Client
}

func NewRepository(pool *pgxpool.Pool, ratesService *rates.Service, redis *redis.Client) *Repository {
	return &Repository{
		pool:         pool,
		ratesService: ratesService,
		redis:        redis,
	}
}

func (r *Repository) Create(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var commissionRate float64
	err = tx.QueryRow(ctx, "SELECT commission_rate FROM stores WHERE id = $1", req.StoreID).Scan(&commissionRate)
	if err != nil {
		return nil, fmt.Errorf("invalid store id: %w", err)
	}

	var subtotal float64
	type evaluatedItem struct {
		productID uuid.UUID
		quantity  int
		unitPrice float64
		subtotal  float64
		options   map[string]any
	}
	evaluatedItems := make([]evaluatedItem, 0, len(req.Items))

	for _, it := range req.Items {
		var catalogPrice float64
		var isAvailable bool
		err := tx.QueryRow(ctx, "SELECT price, is_available FROM products WHERE id = $1 AND store_id = $2", it.ProductID, req.StoreID).Scan(&catalogPrice, &isAvailable)
		if err != nil {
			return nil, fmt.Errorf("product %s not found in store: %w", it.ProductID, err)
		}
		if !isAvailable {
			return nil, fmt.Errorf("product %s is not available", it.ProductID)
		}

		price := catalogPrice
		if it.UnitPrice > 0 {
			price = it.UnitPrice
		}

		itemSubtotal := price * float64(it.Quantity)
		subtotal += itemSubtotal

		evaluatedItems = append(evaluatedItems, evaluatedItem{
			productID: it.ProductID,
			quantity:  it.Quantity,
			unitPrice: price,
			subtotal:  itemSubtotal,
			options:   it.Options,
		})
	}

	commissionFee := subtotal * (commissionRate / 100.0)
	totalUSD := subtotal + req.DeliveryFee + req.ServiceFee

	var bsRate float64 = 896.12
	if r.ratesService != nil {
		if rate, err := r.ratesService.GetOfficialRate(ctx); err == nil && rate > 0 {
			bsRate = rate
		}
	}
	totalBS := totalUSD * bsRate

	orderQuery := `
		INSERT INTO orders (
			customer_id, store_id, status, subtotal, delivery_fee, 
			service_fee, commission_fee, total, total_usd, bs_rate, total_bs,
			delivery_location, delivery_address, payment_method, payment_status
		) VALUES (
			$1, $2, 'placed', $3, $4, 
			$5, $6, $7, $8, $9, $10,
			ST_SetSRID(ST_MakePoint($11, $12), 4326), $13, $14, 'pending'
		)
		RETURNING id, order_number, placed_at
	`

	var o Order
	o.CustomerID = req.CustomerID
	o.StoreID = req.StoreID
	o.Status = StatusPlaced
	o.Subtotal = subtotal
	o.DeliveryFee = req.DeliveryFee
	o.ServiceFee = req.ServiceFee
	o.CommissionFee = commissionFee
	o.Total = totalUSD
	o.TotalUSD = totalUSD
	o.BSRate = bsRate
	o.TotalBS = totalBS
	o.DeliveryLatitude = req.DeliveryLatitude
	o.DeliveryLongitude = req.DeliveryLongitude
	o.DeliveryAddress = req.DeliveryAddress
	o.PaymentMethod = req.PaymentMethod
	o.PaymentStatus = "pending"

	err = tx.QueryRow(
		ctx, orderQuery,
		req.CustomerID, req.StoreID, subtotal, req.DeliveryFee,
		req.ServiceFee, commissionFee, totalUSD, totalUSD, bsRate, totalBS,
		req.DeliveryLongitude, req.DeliveryLatitude, req.DeliveryAddress, req.PaymentMethod,
	).Scan(&o.ID, &o.OrderNumber, &o.PlacedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	itemQuery := `
		INSERT INTO order_items (
			order_id, product_id, quantity, unit_price, subtotal, options
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	for _, item := range evaluatedItems {
		optionsJSON, _ := json.Marshal(item.options)
		if len(optionsJSON) == 0 {
			optionsJSON = []byte("{}")
		}

		var itemID uuid.UUID
		err := tx.QueryRow(
			ctx, itemQuery,
			o.ID, item.productID, item.quantity, item.unitPrice, item.subtotal, optionsJSON,
		).Scan(&itemID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}

		o.Items = append(o.Items, OrderItem{
			ID:        itemID,
			OrderID:   o.ID,
			ProductID: item.productID,
			Quantity:  item.quantity,
			UnitPrice: item.unitPrice,
			Subtotal:  item.subtotal,
			Options:   item.options,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if r.redis != nil {
		_ = r.redis.SAdd(ctx, "orders:pending", o.ID.String()).Err()
		_ = r.redis.Set(ctx, fmt.Sprintf("orders:tracking:%s", o.ID.String()), "pending", 24*time.Hour).Err()
	}

	return &o, nil
}

func (r *Repository) List(ctx context.Context, status string, customerID, storeID, driverID *uuid.UUID, limit, offset int) ([]Order, error) {
	query := `
		SELECT 
			id, order_number, customer_id, store_id, driver_id, status, 
			subtotal, delivery_fee, service_fee, COALESCE(commission_fee, 0), total,
			COALESCE(total_usd, total), COALESCE(bs_rate, 0), COALESCE(total_bs, 0),
			ST_Y(delivery_location::geometry) as lat, ST_X(delivery_location::geometry) as lng,
			delivery_address, payment_method, payment_status, placed_at, delivered_at
		FROM orders
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if customerID != nil {
		query += fmt.Sprintf(" AND customer_id = $%d", argIdx)
		args = append(args, *customerID)
		argIdx++
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
		argIdx++
	}
	if driverID != nil {
		query += fmt.Sprintf(" AND driver_id = $%d", argIdx)
		args = append(args, *driverID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY placed_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerID, &o.StoreID, &o.DriverID, &o.Status,
			&o.Subtotal, &o.DeliveryFee, &o.ServiceFee, &o.CommissionFee, &o.Total,
			&o.TotalUSD, &o.BSRate, &o.TotalBS,
			&o.DeliveryLatitude, &o.DeliveryLongitude, &o.DeliveryAddress,
			&o.PaymentMethod, &o.PaymentStatus, &o.PlacedAt, &o.DeliveredAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	return orders, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	query := `
		SELECT 
			id, order_number, customer_id, store_id, driver_id, status, 
			subtotal, delivery_fee, service_fee, COALESCE(commission_fee, 0), total,
			COALESCE(total_usd, total), COALESCE(bs_rate, 0), COALESCE(total_bs, 0),
			ST_Y(delivery_location::geometry) as lat, ST_X(delivery_location::geometry) as lng,
			delivery_address, payment_method, payment_status, placed_at, delivered_at
		FROM orders
		WHERE id = $1
	`
	var o Order
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.OrderNumber, &o.CustomerID, &o.StoreID, &o.DriverID, &o.Status,
		&o.Subtotal, &o.DeliveryFee, &o.ServiceFee, &o.CommissionFee, &o.Total,
		&o.TotalUSD, &o.BSRate, &o.TotalBS,
		&o.DeliveryLatitude, &o.DeliveryLongitude, &o.DeliveryAddress,
		&o.PaymentMethod, &o.PaymentStatus, &o.PlacedAt, &o.DeliveredAt,
	)
	if err != nil {
		return nil, err
	}

	itemsQuery := `
		SELECT id, order_id, product_id, quantity, unit_price, subtotal, options
		FROM order_items
		WHERE order_id = $1
	`
	rows, err := r.pool.Query(ctx, itemsQuery, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var it OrderItem
			var rawOpts []byte
			if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.Quantity, &it.UnitPrice, &it.Subtotal, &rawOpts); err == nil {
				if len(rawOpts) > 0 {
					_ = json.Unmarshal(rawOpts, &it.Options)
				}
				o.Items = append(o.Items, it)
			}
		}
	}

	return &o, nil
}

func (r *Repository) ExecuteAction(ctx context.Context, id uuid.UUID, action OrderActionRequest) error {
	var updateQuery string
	var args []any

	switch action.Action {
	case "accept":
		updateQuery = "UPDATE orders SET status = 'accepted_by_store' WHERE id = $1 AND status = 'placed'"
		args = []any{id}
	case "prepare":
		updateQuery = "UPDATE orders SET status = 'preparing' WHERE id = $1 AND status = 'accepted_by_store'"
		args = []any{id}
	case "ready":
		updateQuery = "UPDATE orders SET status = 'ready_for_pickup' WHERE id = $1 AND status = 'preparing'"
		args = []any{id}
	case "assign-driver":
		if action.DriverID == nil {
			return fmt.Errorf("driver_id is required for assign-driver action")
		}
		updateQuery = "UPDATE orders SET status = 'driver_assigned', driver_id = $2 WHERE id = $1 AND status IN ('placed', 'accepted_by_store', 'preparing', 'ready_for_pickup')"
		args = []any{id, *action.DriverID}
	case "transit":
		updateQuery = "UPDATE orders SET status = 'in_transit' WHERE id = $1 AND status = 'driver_assigned'"
		args = []any{id}
	case "deliver":
		updateQuery = "UPDATE orders SET status = 'delivered', payment_status = 'paid', delivered_at = $2 WHERE id = $1 AND status = 'in_transit'"
		args = []any{id, time.Now().UTC()}
	case "cancel":
		updateQuery = "UPDATE orders SET status = 'cancelled' WHERE id = $1 AND status != 'delivered'"
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

	return nil
}
