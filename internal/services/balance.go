package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

type BalanceService struct {
	pool *pgxpool.Pool
}

func NewBalanceService(pool *pgxpool.Pool) *BalanceService {
	return &BalanceService{pool: pool}
}

func (s *BalanceService) GetByUUID(ctx context.Context, id uuid.UUID) (*db.Balance, error) {
	return db.GetBalanceByUUID(ctx, s.pool, id)
}

func (s *BalanceService) GetByWalletAndCurrency(ctx context.Context, walletID, currencyID uuid.UUID) (*db.Balance, error) {
	return db.GetBalanceByWalletAndCurrency(ctx, s.pool, walletID, currencyID)
}

func (s *BalanceService) GetAll(ctx context.Context, params db.PaginationParams, filter db.BalanceFilter) (db.PaginatedResult[db.Balance], error) {
	return db.GetBalances(ctx, s.pool, params, filter)
}

type IssueCurrencyParams struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Amount     decimal.Decimal
	Reference  *string
}

type IssueCurrencyResult struct {
	Balance     *db.Balance
	Transaction *db.Transaction
}

func (s *BalanceService) IssueCurrency(ctx context.Context, params IssueCurrencyParams) (*IssueCurrencyResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	balance, err := db.UpsertBalance(ctx, s.pool, db.UpsertBalanceParams{
		WalletID:   params.WalletID,
		CurrencyID: params.CurrencyID,
		Amount:     params.Amount,
	})
	if err != nil {
		return nil, err
	}

	transaction, err := db.CreateTransaction(ctx, s.pool, db.CreateTransactionParams{
		WalletID:   params.WalletID,
		CurrencyID: params.CurrencyID,
		Amount:     params.Amount,
		Type:       db.TransactionTypeAdminIssued,
		Reference:  params.Reference,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &IssueCurrencyResult{
		Balance:     balance,
		Transaction: transaction,
	}, nil
}
