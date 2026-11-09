package database

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/001_initial_schema.sql
var initialSchemaSQL string

//go:embed sql/002_payments_and_reviews.sql
var paymentsAndReviewsSQL string

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, initialSchemaSQL); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, paymentsAndReviewsSQL)
	return err
}
