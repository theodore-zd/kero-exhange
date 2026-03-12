package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

type TransactionService struct {
	pool *pgxpool.Pool
}

func NewTransactionService(pool *pgxpool.Pool) *TransactionService {
	return &TransactionService{pool: pool}
}

func (s *TransactionService) GetByID(ctx context.Context, id uuid.UUID) (*db.Transaction, error) {
	return db.GetTransactionByUUID(ctx, s.pool, id)
}

func (s *TransactionService) GetAll(ctx context.Context, params db.PaginationParams, filter db.TransactionFilter) (db.PaginatedResult[db.Transaction], error) {
	return db.GetTransactions(ctx, s.pool, params, filter)
}
