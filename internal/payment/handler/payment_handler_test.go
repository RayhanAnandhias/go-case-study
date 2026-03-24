package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go-case-study/internal/payment/domain"
)

// MockPaymentService is a mock implementation of domain.PaymentService
type MockPaymentService struct {
	mock.Mock
}

func (m *MockPaymentService) ProcessPayment(ctx context.Context, req domain.ProcessPaymentRequest) (*domain.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func TestPaymentHandler_ProcessPayment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockService := new(MockPaymentService)
		h := NewPaymentHandler(mockService)

		router := gin.New()
		router.POST("/payments", h.ProcessPayment)

		orderID := uuid.New()
		reqBody := domain.ProcessPaymentRequest{
			OrderID: orderID,
			Amount:  100.0,
		}

		expectedPayment := &domain.Payment{
			ID:        uuid.New(),
			OrderID:   orderID,
			Amount:    100.0,
			Status:    domain.PaymentStatusSuccess,
			CreatedAt: time.Now(),
		}

		mockService.On("ProcessPayment", mock.Anything, reqBody).Return(expectedPayment, nil)

		jsonReq, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(jsonReq))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var responsePayment domain.Payment
		err := json.Unmarshal(w.Body.Bytes(), &responsePayment)
		assert.NoError(t, err)
		assert.Equal(t, expectedPayment.ID, responsePayment.ID)
		assert.Equal(t, expectedPayment.Status, responsePayment.Status)
		mockService.AssertExpectations(t)
	})

	t.Run("Bad Request - Invalid JSON", func(t *testing.T) {
		mockService := new(MockPaymentService)
		h := NewPaymentHandler(mockService)

		router := gin.New()
		router.POST("/payments", h.ProcessPayment)

		req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer([]byte("{invalid-json}")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "ProcessPayment", mock.Anything, mock.Anything)
	})
}
