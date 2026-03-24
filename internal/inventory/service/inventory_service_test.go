package service

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go-case-study/internal/inventory/domain"
)

// MockInventoryRepository is a mock of domain.InventoryRepository
type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) IsEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	args := m.Called(ctx, eventID)
	return args.Bool(0), args.Error(1)
}

func (m *MockInventoryRepository) MarkEventProcessed(ctx context.Context, eventID uuid.UUID) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

func TestProcessOrderEvent(t *testing.T) {
	eventID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &domain.OrderEvent{
		ID: eventID,
		Items: []domain.OrderEventItem{
			{ProductID: productID1, Quantity: 2},
			{ProductID: productID2, Quantity: 1},
		},
	}

	t.Run("Success - New Event", func(t *testing.T) {
		db, mockDB, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		repo := new(MockInventoryRepository)
		svc := NewInventoryService(repo, db)

		mockDB.ExpectBegin()
		repo.On("IsEventProcessed", mock.Anything, eventID).Return(false, nil)
		repo.On("UpdateStock", mock.Anything, productID1, 2).Return(nil)
		repo.On("UpdateStock", mock.Anything, productID2, 1).Return(nil)
		repo.On("MarkEventProcessed", mock.Anything, eventID).Return(nil)
		mockDB.ExpectCommit()

		err = svc.ProcessOrderEvent(context.Background(), event)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("Success - Already Processed", func(t *testing.T) {
		db, mockDB, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		repo := new(MockInventoryRepository)
		svc := NewInventoryService(repo, db)

		mockDB.ExpectBegin()
		repo.On("IsEventProcessed", mock.Anything, eventID).Return(true, nil)
		mockDB.ExpectRollback() // Early return calls deferred Rollback

		err = svc.ProcessOrderEvent(context.Background(), event)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("Failure - Stock Update Error", func(t *testing.T) {
		db, mockDB, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		repo := new(MockInventoryRepository)
		svc := NewInventoryService(repo, db)

		mockDB.ExpectBegin()
		repo.On("IsEventProcessed", mock.Anything, eventID).Return(false, nil)
		repo.On("UpdateStock", mock.Anything, productID1, 2).Return(errors.New("db error"))
		mockDB.ExpectRollback()

		err = svc.ProcessOrderEvent(context.Background(), event)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update stock")
		repo.AssertExpectations(t)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})
}
