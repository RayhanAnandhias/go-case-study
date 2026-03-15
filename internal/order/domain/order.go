package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusFailed    OrderStatus = "FAILED"
)

type Order struct {
	ID          uuid.UUID   `json:"id"`
	UserID      uuid.UUID   `json:"user_id"`
	TotalAmount float64     `json:"total_amount"`
	Status      OrderStatus `json:"status"`
	Items       []OrderItem `json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	UserID uuid.UUID                `json:"-" validate:"-"`
	Items  []CreateOrderItemRequest `json:"items" validate:"required,dive"`
}

type CreateOrderItemRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity" validate:"required,min=1"`
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetOrder(ctx context.Context, id uuid.UUID) (*Order, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
}

type OrderService interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error)
	GetOrder(ctx context.Context, id uuid.UUID) (*Order, error)
}
