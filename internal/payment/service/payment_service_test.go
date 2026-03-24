package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go-case-study/internal/payment/domain"
)

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) CreatePayment(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func TestPaymentService_ProcessPayment(t *testing.T) {
	orderID := uuid.New()
	amount := 100.0

	req := domain.ProcessPaymentRequest{
		OrderID: orderID,
		Amount:  amount,
	}

	t.Run("Success", func(t *testing.T) {
		repo := new(MockPaymentRepository)
		svc := NewPaymentService(repo)

		repo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
			return p.OrderID == orderID && p.Amount == amount && p.Status == domain.PaymentStatusSuccess
		})).Return(nil)

		payment, err := svc.ProcessPayment(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, domain.PaymentStatusSuccess, payment.Status)
		assert.Equal(t, orderID, payment.OrderID)
		assert.Equal(t, amount, payment.Amount)

		repo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		repo := new(MockPaymentRepository)
		svc := NewPaymentService(repo)

		repo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(errors.New("db error"))

		payment, err := svc.ProcessPayment(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, payment)
		assert.Equal(t, "db error", err.Error())

		repo.AssertExpectations(t)
	})
}
