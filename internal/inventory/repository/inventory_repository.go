package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"go-case-study/internal/inventory/domain"
)

type txKey struct{}

func InjectTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func ExtractTx(ctx context.Context) *sql.Tx {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return nil
}

type Queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type inventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) domain.InventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) getQueryer(ctx context.Context) Queryer {
	if tx := ExtractTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *inventoryRepository) UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	q := r.getQueryer(ctx)

	query := `
		UPDATE inventory_schema.products
		SET stock = stock - $1, updated_at = $2
		WHERE id = $3 AND stock >= $1
	`
	res, err := q.ExecContext(ctx, query, quantity, time.Now().UTC(), productID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return errors.New("insufficient stock or product not found")
	}
	
	return nil
}

func (r *inventoryRepository) IsEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	q := r.getQueryer(ctx)
	var count int
	err := q.QueryRowContext(ctx, "SELECT COUNT(1) FROM inventory_schema.processed_events WHERE event_id = $1", eventID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *inventoryRepository) MarkEventProcessed(ctx context.Context, eventID uuid.UUID) error {
	q := r.getQueryer(ctx)
	_, err := q.ExecContext(ctx, "INSERT INTO inventory_schema.processed_events (event_id, processed_at) VALUES ($1, $2)", eventID, time.Now().UTC())
	return err
}
