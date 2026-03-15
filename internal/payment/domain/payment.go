package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

type Payment struct {
	ID        uuid.UUID     `json:"id"`
	OrderID   uuid.UUID     `json:"order_id"`
	Amount    float64       `json:"amount"`
	Status    PaymentStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

type ProcessPaymentRequest struct {
	OrderID uuid.UUID `json:"order_id" validate:"required"`
	Amount  float64   `json:"amount" validate:"required,gt=0"`
}

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *Payment) error
}

type PaymentService interface {
	ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (*Payment, error)
}
