package reviews

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, req CreateReviewRequest) (*Review, error) {
	var orderExists bool
	err := r.pool.QueryRow(ctx, "SELECT true FROM orders WHERE id = $1", req.OrderID).Scan(&orderExists)
	if err != nil || !orderExists {
		return nil, fmt.Errorf("invalid order id: %w", err)
	}

	query := `
		INSERT INTO reviews (order_id, customer_id, store_id, driver_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	var rev Review
	rev.OrderID = req.OrderID
	rev.CustomerID = req.CustomerID
	rev.StoreID = req.StoreID
	rev.DriverID = req.DriverID
	rev.Rating = req.Rating
	rev.Comment = req.Comment

	err = r.pool.QueryRow(
		ctx, query,
		req.OrderID, req.CustomerID, req.StoreID, req.DriverID, req.Rating, req.Comment,
	).Scan(&rev.ID, &rev.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert review: %w", err)
	}

	return &rev, nil
}

func (r *Repository) List(ctx context.Context, storeID, driverID *uuid.UUID, limit, offset int) ([]Review, error) {
	query := `
		SELECT id, order_id, customer_id, store_id, driver_id, rating, COALESCE(comment, ''), created_at
		FROM reviews
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

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

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var rev Review
		if err := rows.Scan(
			&rev.ID, &rev.OrderID, &rev.CustomerID, &rev.StoreID, &rev.DriverID,
			&rev.Rating, &rev.Comment, &rev.CreatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}

	return reviews, nil
}
