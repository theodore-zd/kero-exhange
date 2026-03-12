package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

type CurrencyService struct {
	pool *pgxpool.Pool
}

func NewCurrencyService(pool *pgxpool.Pool) *CurrencyService {
	return &CurrencyService{pool: pool}
}

func (s *CurrencyService) GetByID(ctx context.Context, id uuid.UUID) (*db.Currency, error) {
	return db.GetCurrencyByUUID(ctx, s.pool, id)
}

func (s *CurrencyService) GetByCode(ctx context.Context, code string) (*db.Currency, error) {
	return db.GetCurrencyByCode(ctx, s.pool, code)
}

func (s *CurrencyService) GetAll(ctx context.Context, params db.PaginationParams) (db.PaginatedResult[db.Currency], error) {
	return db.GetCurrencies(ctx, s.pool, params)
}

func (s *CurrencyService) Create(ctx context.Context, params db.CreateCurrencyParams) (*db.Currency, error) {
	return db.CreateCurrency(ctx, s.pool, params)
}
