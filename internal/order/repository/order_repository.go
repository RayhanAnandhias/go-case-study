package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go-case-study/internal/order/domain"

	"github.com/google/uuid"
)

type txKey struct{}

// InjectTx puts a transaction into the context
func InjectTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ExtractTx gets a transaction from the context
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

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) domain.OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) getQueryer(ctx context.Context) Queryer {
	if tx := ExtractTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	q := r.getQueryer(ctx)

	query := `
		INSERT INTO order_schema.orders (id, user_id, total_amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := q.ExecContext(ctx, query, order.ID, order.UserID, order.TotalAmount, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		itemQuery := `
			INSERT INTO order_schema.order_items (id, order_id, product_id, quantity, price, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = q.ExecContext(ctx, itemQuery, item.ID, item.OrderID, item.ProductID, item.Quantity, item.Price, item.CreatedAt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *orderRepository) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	q := r.getQueryer(ctx)

	order := &domain.Order{}
	query := `
		SELECT id, user_id, total_amount, status, created_at, updated_at
		FROM order_schema.orders
		WHERE id = $1
	`
	err := q.QueryRowContext(ctx, query, id).Scan(
		&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}

	rows, err := q.QueryContext(ctx, "SELECT id, order_id, product_id, quantity, unit_price, created_at FROM order_schema.order_items WHERE order_id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &item.CreatedAt); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	q := r.getQueryer(ctx)

	query := `
		UPDATE order_schema.orders
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := q.ExecContext(ctx, query, status, time.Now().UTC(), id)
	return err
}
