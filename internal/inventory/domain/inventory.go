package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProcessedEvent struct {
	EventID     uuid.UUID `json:"event_id"`
	ProcessedAt time.Time `json:"processed_at"`
}

type OrderEventItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type OrderEvent struct {
	ID    uuid.UUID        `json:"id"`
	Items []OrderEventItem `json:"items"`
}

type InventoryRepository interface {
	UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error
	IsEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID uuid.UUID) error
}

type InventoryService interface {
	ProcessOrderEvent(ctx context.Context, event *OrderEvent) error
}
