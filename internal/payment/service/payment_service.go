package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go-case-study/internal/payment/domain"
)

type paymentService struct {
	repo domain.PaymentRepository
}

func NewPaymentService(repo domain.PaymentRepository) domain.PaymentService {
	return &paymentService{repo: repo}
}

func (s *paymentService) ProcessPayment(ctx context.Context, req domain.ProcessPaymentRequest) (*domain.Payment, error) {
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
