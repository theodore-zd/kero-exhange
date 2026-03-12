package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Balance struct {
	UUID       uuid.UUID
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Balance    decimal.Decimal
	UpdatedAt  time.Time
}

type BalanceFilter struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
}

func GetBalanceByWalletAndCurrency(ctx context.Context, pool *pgxpool.Pool, walletID, currencyID uuid.UUID) (*Balance, error) {
	query := `
		SELECT uuid, wallet_id, currency_id, balance, updated_at
		FROM balances
		WHERE wallet_id = $1 AND currency_id = $2
	`
	var b Balance
	err := pool.QueryRow(ctx, query, walletID, currencyID).Scan(
		&b.UUID,
		&b.WalletID,
		&b.CurrencyID,
		&b.Balance,
		&b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get balance by wallet and currency: %w", err)
	}
	return &b, nil
}

type UpsertBalanceParams struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Amount     decimal.Decimal
}

func UpsertBalance(ctx context.Context, pool *pgxpool.Pool, params UpsertBalanceParams) (*Balance, error) {
	query := `
		INSERT INTO balances (wallet_id, currency_id, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (wallet_id, currency_id) 
		DO UPDATE SET balance = balances.balance + EXCLUDED.balance, updated_at = NOW()
		RETURNING uuid, wallet_id, currency_id, balance, updated_at
	`
	var b Balance
	err := pool.QueryRow(ctx, query, params.WalletID, params.CurrencyID, params.Amount).Scan(
		&b.UUID,
		&b.WalletID,
		&b.CurrencyID,
		&b.Balance,
		&b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert balance: %w", err)
	}
	return &b, nil
}

func GetBalanceByUUID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Balance, error) {
	query := `
		SELECT uuid, wallet_id, currency_id, balance, updated_at
		FROM balances
		WHERE uuid = $1
	`
	var b Balance
	err := pool.QueryRow(ctx, query, id).Scan(
		&b.UUID,
		&b.WalletID,
		&b.CurrencyID,
		&b.Balance,
		&b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get balance by uuid: %w", err)
	}
	return &b, nil
}

func GetBalances(ctx context.Context, pool *pgxpool.Pool, params PaginationParams, filter BalanceFilter) (PaginatedResult[Balance], error) {
	params = params.Normalize()

	baseQuery := `SELECT uuid, wallet_id, currency_id, balance, updated_at FROM balances`
	countQuery := `SELECT COUNT(*) FROM balances`
	args := []any{}
	argIdx := 1
	hasWhere := false

	if filter.WalletID != uuid.Nil {
		where := fmt.Sprintf(" WHERE wallet_id = $%d", argIdx)
		baseQuery += where
		countQuery += where
		args = append(args, filter.WalletID)
		argIdx++
		hasWhere = true
	}

	if filter.CurrencyID != uuid.Nil {
		connector := " WHERE "
		if hasWhere {
			connector = " AND "
		}
		where := fmt.Sprintf("%scurrency_id = $%d", connector, argIdx)
		baseQuery += where
		countQuery += where
		args = append(args, filter.CurrencyID)
		argIdx++
	}

	baseQuery += " ORDER BY updated_at DESC"

	return Paginate(ctx, pool, baseQuery, countQuery, args, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Balance, error) {
			var balances []Balance
			for rows.Next() {
				var b Balance
				if err := rows.Scan(&b.UUID, &b.WalletID, &b.CurrencyID, &b.Balance, &b.UpdatedAt); err != nil {
					return nil, err
				}
				balances = append(balances, b)
			}
			return balances, nil
		},
	)
}
