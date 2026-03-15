package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go-case-study/internal/order/domain"
)

type productRepositoryImpl struct {
	redisClient *redis.Client
	db          *sql.DB
}

func NewProductRepository(rdb *redis.Client, db *sql.DB) domain.ProductRepository {
	return &productRepositoryImpl{redisClient: rdb, db: db}
}

func (p *productRepositoryImpl) GetPrice(ctx context.Context, productID uuid.UUID) (float64, error) {
	cacheKey := fmt.Sprintf("product:%s:price", productID)
	
	// Cache-aside: try getting from redis
	val, err := p.redisClient.Get(ctx, cacheKey).Float64()
	if err == nil {
		return val, nil
	} else if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("redis error: %w", err)
	}

	// Fetch from DB
	err = p.db.QueryRowContext(ctx, "SELECT price FROM inventory_schema.products WHERE id = $1", productID).Scan(&val)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("product not found")
		}
		return 0, fmt.Errorf("db error: %w", err)
	}

	// Store in redis
	_ = p.redisClient.Set(ctx, cacheKey, val, 10*time.Minute).Err()

	return val, nil
}
