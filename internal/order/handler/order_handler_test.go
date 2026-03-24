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

	"go-case-study/internal/order/domain"
	"go-case-study/pkg/auth"
	"go-case-study/pkg/middleware"
)

// MockOrderService is a mock implementation of domain.OrderService
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderService) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func TestOrderHandler_CreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	t.Run("Success", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService, jwtSecret)

		router := gin.New()
		router.Use(middleware.RequireAuth(jwtSecret))
		router.POST("/orders", handler.CreateOrder)

		userID := uuid.New()
		productID := uuid.New()
		token, _ := auth.GenerateMockJWT(userID.String(), jwtSecret)

		reqBody := domain.CreateOrderRequest{
			Items: []domain.CreateOrderItemRequest{
				{
					ProductID: productID,
					Quantity:  2,
				},
			},
		}

		expectedOrder := &domain.Order{
			ID:          uuid.New(),
			UserID:      userID,
			TotalAmount: 100.0,
			Status:      domain.OrderStatusPaid,
			Items: []domain.OrderItem{
				{
					ID:        uuid.New(),
					ProductID: productID,
					Quantity:  2,
					Price:     50.0,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockService.On("CreateOrder", mock.Anything, mock.MatchedBy(func(req domain.CreateOrderRequest) bool {
			return req.UserID == userID && len(req.Items) == 1 && req.Items[0].ProductID == productID
		})).Return(expectedOrder, nil)

		jsonReq, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var responseOrder domain.Order
		err := json.Unmarshal(w.Body.Bytes(), &responseOrder)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrder.ID, responseOrder.ID)
		assert.Equal(t, userID, responseOrder.UserID)
		mockService.AssertExpectations(t)
	})

	t.Run("Unauthorized - No Token", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService, jwtSecret)

		router := gin.New()
		router.Use(middleware.RequireAuth(jwtSecret))
		router.POST("/orders", handler.CreateOrder)

		reqBody := domain.CreateOrderRequest{
			Items: []domain.CreateOrderItemRequest{
				{
					ProductID: uuid.New(),
					Quantity:  2,
				},
			},
		}

		jsonReq, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonReq))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockService.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Unauthorized - Invalid Token", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService, jwtSecret)

		router := gin.New()
		router.Use(middleware.RequireAuth(jwtSecret))
		router.POST("/orders", handler.CreateOrder)

		token := "invalid-token"

		reqBody := domain.CreateOrderRequest{
			Items: []domain.CreateOrderItemRequest{
				{
					ProductID: uuid.New(),
					Quantity:  2,
				},
			},
		}

		jsonReq, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockService.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Validation Error - Invalid JSON", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService, jwtSecret)

		router := gin.New()
		router.Use(middleware.RequireAuth(jwtSecret))
		router.POST("/orders", handler.CreateOrder)

		userID := uuid.New()
		token, _ := auth.GenerateMockJWT(userID.String(), jwtSecret)

		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer([]byte("{invalid-json}")))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Validation Error - Missing Items", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService, jwtSecret)

		router := gin.New()
		router.Use(middleware.RequireAuth(jwtSecret))
		router.POST("/orders", handler.CreateOrder)

		userID := uuid.New()
		token, _ := auth.GenerateMockJWT(userID.String(), jwtSecret)

		reqBody := domain.CreateOrderRequest{
			Items: []domain.CreateOrderItemRequest{},
		}

		jsonReq, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(jsonReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})
}
