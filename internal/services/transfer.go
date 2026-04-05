package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

var (
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrSameWallet           = errors.New("source and destination wallet are the same")
	ErrDestinationNotFound  = errors.New("destination wallet not found")
	ErrCurrencyNotFound     = errors.New("currency not found")
)

type TransferService struct {
	pool *pgxpool.Pool
}

func NewTransferService(pool *pgxpool.Pool) *TransferService {
	return &TransferService{pool: pool}
}

type TransferParams struct {
	SourceWalletID uuid.UUID
	DestWalletID   uuid.UUID
	CurrencyID     uuid.UUID
	Amount         decimal.Decimal
}

type TransferResult struct {
	TransferID uuid.UUID
	Debit      *db.Transaction
	Credit     *db.Transaction
}

func (s *TransferService) Transfer(ctx context.Context, params TransferParams) (*TransferResult, error) {
	if params.SourceWalletID == params.DestWalletID {
		return nil, ErrSameWallet
	}

	if !params.Amount.IsPositive() {
		return nil, ErrInsufficientBalance
	}

	dest, err := db.GetWalletByUUID(ctx, s.pool, params.DestWalletID)
	if err != nil {
		return nil, fmt.Errorf("get destination wallet: %w", err)
	}
	if dest == nil {
		return nil, ErrDestinationNotFound
	}

	currency, err := db.GetCurrencyByUUID(ctx, s.pool, params.CurrencyID)
	if err != nil {
		return nil, fmt.Errorf("get currency: %w", err)
	}
	if currency == nil {
		return nil, ErrCurrencyNotFound
	}

	balance, err := db.GetBalanceByWalletAndCurrency(ctx, s.pool, params.SourceWalletID, params.CurrencyID)
	if err != nil {
		return nil, fmt.Errorf("get source balance: %w", err)
	}
	if balance == nil || balance.Balance.LessThan(params.Amount) {
		return nil, ErrInsufficientBalance
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transfer transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	transferID := uuid.New()

	debit, err := db.CreateTransactionTx(ctx, tx, db.CreateTransactionParams{
		WalletID:   params.SourceWalletID,
		CurrencyID: params.CurrencyID,
		Amount:     params.Amount.Neg(),
		Type:       db.TransactionTypeTransfer,
		TransferID: &transferID,
	})
	if err != nil {
		return nil, fmt.Errorf("create debit transaction: %w", err)
	}

	credit, err := db.CreateTransactionTx(ctx, tx, db.CreateTransactionParams{
		WalletID:   params.DestWalletID,
		CurrencyID: params.CurrencyID,
		Amount:     params.Amount,
		Type:       db.TransactionTypeTransfer,
		TransferID: &transferID,
	})
	if err != nil {
		return nil, fmt.Errorf("create credit transaction: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transfer transaction: %w", err)
	}

	return &TransferResult{
		TransferID: transferID,
		Debit:      debit,
		Credit:     credit,
	}, nil
}
