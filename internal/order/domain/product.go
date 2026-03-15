package domain

import (
	"context"

	"github.com/google/uuid"
)

type ProductRepository interface {
	GetPrice(ctx context.Context, productID uuid.UUID) (float64, error)
}
