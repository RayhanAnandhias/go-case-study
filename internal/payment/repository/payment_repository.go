package repository

import (
	"context"
	"database/sql"
	"time"

	"go-case-study/internal/payment/domain"
)

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) domain.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreatePayment(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payment_schema.payments (id, order_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	payment.CreatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, payment.ID, payment.OrderID, payment.Amount, payment.Status, payment.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}
