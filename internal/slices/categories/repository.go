package categories

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

func (r *Repository) List(ctx context.Context, limit, offset int) ([]Category, error) {
	query := `
		SELECT id, name, slug, COALESCE(image_url, ''), sort_order, created_at
		FROM categories
		ORDER BY sort_order ASC, name ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ImageURL, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	query := `
		SELECT id, name, slug, COALESCE(image_url, ''), sort_order, created_at
		FROM categories
		WHERE id = $1
	`
	var c Category
	err := r.pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.Slug, &c.ImageURL, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) Create(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	query := `
		INSERT INTO categories (name, slug, image_url, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	var c Category
	c.Name = req.Name
	c.Slug = req.Slug
	c.ImageURL = req.ImageURL
	c.SortOrder = req.SortOrder

	err := r.pool.QueryRow(ctx, query, req.Name, req.Slug, req.ImageURL, req.SortOrder).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert category: %w", err)
	}

	return &c, nil
}
