package products

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, storeID, categoryID *uuid.UUID, search string, limit, offset int) ([]Product, error) {
	query := `
		SELECT id, store_id, category_id, name, COALESCE(description, ''), 
		       base_price, is_available, attributes, created_at, updated_at
		FROM products
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
		argIdx++
	}

	if categoryID != nil {
		query += fmt.Sprintf(" AND category_id = $%d", argIdx)
		args = append(args, *categoryID)
		argIdx++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var rawAttrs []byte
		if err := rows.Scan(
			&p.ID, &p.StoreID, &p.CategoryID, &p.Name, &p.Description,
			&p.BasePrice, &p.IsAvailable, &rawAttrs, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(rawAttrs) > 0 {
			_ = json.Unmarshal(rawAttrs, &p.Attributes)
		}
		products = append(products, p)
	}

	return products, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	query := `
		SELECT id, store_id, category_id, name, COALESCE(description, ''), 
		       base_price, is_available, attributes, created_at, updated_at
		FROM products
		WHERE id = $1
	`
	var p Product
	var rawAttrs []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.StoreID, &p.CategoryID, &p.Name, &p.Description,
		&p.BasePrice, &p.IsAvailable, &rawAttrs, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(rawAttrs) > 0 {
		_ = json.Unmarshal(rawAttrs, &p.Attributes)
	}

	return &p, nil
}

func (r *Repository) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	var storeExists bool
	err := r.pool.QueryRow(ctx, "SELECT true FROM stores WHERE id = $1", req.StoreID).Scan(&storeExists)
	if err != nil || !storeExists {
		return nil, fmt.Errorf("invalid store id: %w", err)
	}

	query := `
		INSERT INTO products (
			store_id, category_id, name, description, base_price, is_available, attributes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		RETURNING id, created_at, updated_at
	`

	attrsJSON, _ := json.Marshal(req.Attributes)
	if len(attrsJSON) == 0 {
		attrsJSON = []byte("{}")
	}

	var p Product
	p.StoreID = req.StoreID
	p.CategoryID = req.CategoryID
	p.Name = req.Name
	p.Description = req.Description
	p.BasePrice = req.BasePrice
	p.IsAvailable = req.IsAvailable
	p.Attributes = req.Attributes

	err = r.pool.QueryRow(
		ctx, query,
		req.StoreID, req.CategoryID, req.Name, req.Description, req.BasePrice, req.IsAvailable, attrsJSON,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert product: %w", err)
	}

	return &p, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateProductRequest) (*Product, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.CategoryID != nil {
		existing.CategoryID = req.CategoryID
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.BasePrice != nil {
		existing.BasePrice = *req.BasePrice
	}
	if req.IsAvailable != nil {
		existing.IsAvailable = *req.IsAvailable
	}
	if req.Attributes != nil {
		existing.Attributes = *req.Attributes
	}

	attrsJSON, _ := json.Marshal(existing.Attributes)
	if len(attrsJSON) == 0 {
		attrsJSON = []byte("{}")
	}

	query := `
		UPDATE products
		SET category_id = $1, name = $2, description = $3, base_price = $4,
		    is_available = $5, attributes = $6, updated_at = $7
		WHERE id = $8
		RETURNING updated_at
	`
	now := time.Now().UTC()
	err = r.pool.QueryRow(
		ctx, query,
		existing.CategoryID, existing.Name, existing.Description,
		existing.BasePrice, existing.IsAvailable, attrsJSON, now, id,
	).Scan(&existing.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
