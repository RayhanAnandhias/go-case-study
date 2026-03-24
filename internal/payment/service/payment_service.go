package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go-case-study/internal/payment/domain"
)

type paymentService struct {
	repo   domain.PaymentRepository
	tracer trace.Tracer
}

func NewPaymentService(repo domain.PaymentRepository) domain.PaymentService {
	return &paymentService{
		repo:   repo,
		tracer: otel.Tracer("payment-service"),
	}
}

func (s *paymentService) ProcessPayment(ctx context.Context, req domain.ProcessPaymentRequest) (*domain.Payment, error) {
	ctx, span := s.tracer.Start(ctx, "ProcessPayment")
	defer span.End()

	status := domain.PaymentStatusFailed
	if req.Amount > 0 {
		status = domain.PaymentStatusSuccess
	}

	payment := &domain.Payment{
		ID:        uuid.New(),
		OrderID:   req.OrderID,
		Amount:    req.Amount,
		Status:    status,
		CreatedAt: time.Now().UTC(),
	}

	err := s.repo.CreatePayment(ctx, payment)
	if err != nil {
		return nil, err
	}

	return payment, nil
}
