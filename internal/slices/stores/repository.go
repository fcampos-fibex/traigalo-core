package stores

import (
	"context"
	"encoding/json"
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

func (r *Repository) List(ctx context.Context, onlyOpen bool, limit, offset int) ([]Store, error) {
	query := `
		SELECT 
			id, owner_id, name, slug, COALESCE(description, ''), 
			COALESCE(logo_url, ''), COALESCE(cover_url, ''), 
			ST_Y(location::geometry) as lat, ST_X(location::geometry) as lng,
			address_line, commission_rate, is_open, created_at, updated_at
		FROM stores
	`
	if onlyOpen {
		query += " WHERE is_open = true"
	}
	query += " ORDER BY name ASC LIMIT $1 OFFSET $2"

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []Store
	for rows.Next() {
		var s Store
		if err := rows.Scan(
			&s.ID, &s.OwnerID, &s.Name, &s.Slug, &s.Description,
			&s.LogoURL, &s.CoverURL, &s.Latitude, &s.Longitude,
			&s.AddressLine, &s.CommissionRate, &s.IsOpen,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}

	return stores, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Store, error) {
	query := `
		SELECT 
			id, owner_id, name, slug, COALESCE(description, ''), 
			COALESCE(logo_url, ''), COALESCE(cover_url, ''), 
			ST_Y(location::geometry) as lat, ST_X(location::geometry) as lng,
			address_line, commission_rate, is_open, created_at, updated_at
		FROM stores
		WHERE id = $1
	`
	var s Store
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.OwnerID, &s.Name, &s.Slug, &s.Description,
		&s.LogoURL, &s.CoverURL, &s.Latitude, &s.Longitude,
		&s.AddressLine, &s.CommissionRate, &s.IsOpen,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *Repository) ListProducts(ctx context.Context, storeID uuid.UUID) ([]Product, error) {
	query := `
		SELECT 
			id, store_id, category_id, name, COALESCE(description, ''), 
			base_price, is_available, attributes, created_at, updated_at
		FROM products
		WHERE store_id = $1 AND is_available = true
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, storeID)
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

func (r *Repository) Create(ctx context.Context, req CreateStoreRequest) (*Store, error) {
	query := `
		INSERT INTO stores (
			owner_id, name, slug, description, logo_url, cover_url, 
			location, address_line, commission_rate, is_open
		) VALUES (
			$1, $2, $3, $4, $5, $6, 
			ST_SetSRID(ST_MakePoint($7, $8), 4326), $9, $10, true
		)
		RETURNING id, created_at, updated_at
	`
	var s Store
	s.OwnerID = req.OwnerID
	s.Name = req.Name
	s.Slug = req.Slug
	s.Description = req.Description
	s.LogoURL = req.LogoURL
	s.CoverURL = req.CoverURL
	s.Latitude = req.Latitude
	s.Longitude = req.Longitude
	s.AddressLine = req.AddressLine
	s.CommissionRate = req.CommissionRate
	s.IsOpen = true

	err := r.pool.QueryRow(
		ctx, query,
		req.OwnerID, req.Name, req.Slug, req.Description, req.LogoURL, req.CoverURL,
		req.Longitude, req.Latitude, req.AddressLine, req.CommissionRate,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert store: %w", err)
	}

	return &s, nil
}
