package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

type WalletService struct {
	pool *pgxpool.Pool
}

func NewWalletService(pool *pgxpool.Pool) *WalletService {
	return &WalletService{pool: pool}
}

func (s *WalletService) GetByID(ctx context.Context, id uuid.UUID) (*db.Wallet, error) {
	return db.GetWalletByUUID(ctx, s.pool, id)
}

func (s *WalletService) GetAll(ctx context.Context, params db.PaginationParams, filter db.WalletFilter) (db.PaginatedResult[db.Wallet], error) {
	return db.GetWallets(ctx, s.pool, params, filter)
}
