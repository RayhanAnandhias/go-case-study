package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"go-case-study/internal/inventory/domain"
	"go-case-study/internal/inventory/repository"
)

type inventoryService struct {
	repo domain.InventoryRepository
	db   *sql.DB
}

func NewInventoryService(repo domain.InventoryRepository, db *sql.DB) domain.InventoryService {
	return &inventoryService{
		repo: repo,
		db:   db,
	}
}

func (s *inventoryService) ProcessOrderEvent(ctx context.Context, event *domain.OrderEvent) error {
	// Start DB transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	ctxWithTx := repository.InjectTx(ctx, tx)

	// Check if already processed
	processed, err := s.repo.IsEventProcessed(ctxWithTx, event.ID)
	if err != nil {
		return fmt.Errorf("failed to check if event is processed: %w", err)
	}

	if processed {
		log.Printf("Event %s already processed. Skipping.", event.ID)
		return nil // Idempotent
	}

	// Update stock for each item
	for _, item := range event.Items {
		if err := s.repo.UpdateStock(ctxWithTx, item.ProductID, item.Quantity); err != nil {
			return fmt.Errorf("failed to update stock for product %s: %w", item.ProductID, err)
		}
	}

	// Mark event as processed
	if err := s.repo.MarkEventProcessed(ctxWithTx, event.ID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}
