package payments

import (
	"context"
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

func (r *Repository) CreateReceipt(ctx context.Context, req CreateReceiptRequest) (*PaymentReceipt, error) {
	var orderExists bool
	err := r.pool.QueryRow(ctx, "SELECT true FROM orders WHERE id = $1", req.OrderID).Scan(&orderExists)
	if err != nil || !orderExists {
		return nil, fmt.Errorf("invalid order id: %w", err)
	}

	query := `
		INSERT INTO payment_receipts (
			id, order_id, payment_method, reference_number, bank_origin, bank_destination,
			amount_usd, amount_bs, bs_rate, status, notes
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, 'pending', $9
		)
		RETURNING id, created_at
	`
	var rc PaymentReceipt
	rc.OrderID = req.OrderID
	rc.PaymentMethod = req.PaymentMethod
	rc.ReferenceNumber = req.ReferenceNumber
	rc.IssuingBank = req.IssuingBank
	rc.ReceivingBank = req.ReceivingBank
	rc.AmountUSD = req.AmountUSD
	rc.AmountBS = req.AmountBS
	rc.Status = "pending"
	rc.Notes = req.Notes

	err = r.pool.QueryRow(
		ctx, query,
		req.OrderID, req.PaymentMethod, req.ReferenceNumber, req.IssuingBank, req.ReceivingBank,
		req.AmountUSD, req.AmountBS, req.BSRate, req.Notes,
	).Scan(&rc.ID, &rc.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert receipt: %w", err)
	}

	return &rc, nil
}

func (r *Repository) ListReceipts(ctx context.Context, orderID *uuid.UUID, limit, offset int) ([]PaymentReceipt, error) {
	query := `
		SELECT id, order_id, payment_method, reference_number, COALESCE(bank_origin, ''),
		       COALESCE(bank_destination, ''), amount_usd, amount_bs, status, verified_at,
		       COALESCE(notes, ''), created_at
		FROM payment_receipts
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if orderID != nil {
		query += fmt.Sprintf(" AND order_id = $%d", argIdx)
		args = append(args, *orderID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []PaymentReceipt
	for rows.Next() {
		var rc PaymentReceipt
		if err := rows.Scan(
			&rc.ID, &rc.OrderID, &rc.PaymentMethod, &rc.ReferenceNumber, &rc.IssuingBank,
			&rc.ReceivingBank, &rc.AmountUSD, &rc.AmountBS, &rc.Status, &rc.VerifiedAt,
			&rc.Notes, &rc.CreatedAt,
		); err != nil {
			return nil, err
		}
		receipts = append(receipts, rc)
	}

	return receipts, nil
}

func (r *Repository) ExecuteReceiptAction(ctx context.Context, id uuid.UUID, action ReceiptActionRequest) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var orderID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT order_id FROM payment_receipts WHERE id = $1", id).Scan(&orderID)
	if err != nil {
		return err
	}

	switch action.Action {
	case "verify":
		now := time.Now().UTC()
		cmd, err := tx.Exec(ctx, "UPDATE payment_receipts SET status = 'verified', verified_at = $1, notes = COALESCE($2, notes) WHERE id = $3", now, action.Notes, id)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		_, err = tx.Exec(ctx, "UPDATE orders SET payment_status = 'paid' WHERE id = $1", orderID)
		if err != nil {
			return err
		}
	case "reject":
		cmd, err := tx.Exec(ctx, "UPDATE payment_receipts SET status = 'rejected', notes = COALESCE($1, notes) WHERE id = $2", action.Notes, id)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}

	return tx.Commit(ctx)
}
