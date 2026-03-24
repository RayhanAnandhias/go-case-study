package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go-case-study/internal/order/domain"
)

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) GetPrice(ctx context.Context, productID uuid.UUID) (float64, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).(float64), args.Error(1)
}

type MockKafkaWriter struct {
	mock.Mock
}

func (m *MockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func TestOrderService_CreateOrder(t *testing.T) {
	userID := uuid.New()
	productID := uuid.New()
	price := 100.0
	quantity := 2
	totalAmount := price * float64(quantity)

	req := domain.CreateOrderRequest{
		UserID: userID,
		Items: []domain.CreateOrderItemRequest{
			{
				ProductID: productID,
				Quantity:  quantity,
			},
		},
	}

	t.Run("Success", func(t *testing.T) {
		repo := new(MockOrderRepository)
		productRepo := new(MockProductRepository)
		kafkaWriter := new(MockKafkaWriter)
		db, dbMock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})

		// Product price mock
		productRepo.On("GetPrice", mock.Anything, productID).Return(price, nil)

		// DB mock
		dbMock.ExpectBegin()
		repo.On("CreateOrder", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil)

		// HTTP mock for payment
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/payments", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Kafka mock
		kafkaWriter.On("WriteMessages", mock.Anything, mock.Anything).Return(nil)

		// Status update mock
		repo.On("UpdateOrderStatus", mock.Anything, mock.Anything, domain.OrderStatusPaid).Return(nil)

		dbMock.ExpectCommit()

		svc := NewOrderService(repo, productRepo, redisClient, db, kafkaWriter, server.URL, http.DefaultClient)

		ctx := context.WithValue(context.Background(), "auth_token", "test-token")
		order, err := svc.CreateOrder(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, domain.OrderStatusPaid, order.Status)
		assert.Equal(t, totalAmount, order.TotalAmount)

		repo.AssertExpectations(t)
		productRepo.AssertExpectations(t)
		kafkaWriter.AssertExpectations(t)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("Rate Limit Exceeded", func(t *testing.T) {
		repo := new(MockOrderRepository)
		productRepo := new(MockProductRepository)
		kafkaWriter := new(MockKafkaWriter)
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})

		// Pre-fill miniredis with 0 tokens to trigger rate limit exceeded
		mr.HSet("rate_limit:"+userID.String(), "tokens", "0")
		mr.HSet("rate_limit:"+userID.String(), "last_refresh", fmt.Sprintf("%d", time.Now().Unix()))

		svc := NewOrderService(repo, productRepo, redisClient, db, kafkaWriter, "http://payment-service", http.DefaultClient)

		ctx := context.Background()
		order, err := svc.CreateOrder(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit exceeded")
		assert.Nil(t, order)
	})

	t.Run("Payment Failure", func(t *testing.T) {
		repo := new(MockOrderRepository)
		productRepo := new(MockProductRepository)
		kafkaWriter := new(MockKafkaWriter)
		db, dbMock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})

		// Product price mock
		productRepo.On("GetPrice", mock.Anything, productID).Return(price, nil)

		// DB mock
		dbMock.ExpectBegin()
		repo.On("CreateOrder", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil)

		// HTTP mock for payment (returns 500)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		dbMock.ExpectRollback()

		svc := NewOrderService(repo, productRepo, redisClient, db, kafkaWriter, server.URL, http.DefaultClient)

		ctx := context.WithValue(context.Background(), "auth_token", "test-token")
		order, err := svc.CreateOrder(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payment failed")
		assert.Nil(t, order)

		repo.AssertExpectations(t)
		productRepo.AssertExpectations(t)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}
