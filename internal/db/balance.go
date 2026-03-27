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
	UUID         uuid.UUID
	WalletID     uuid.UUID
	CurrencyID   uuid.UUID
	Balance      decimal.Decimal
	UpdatedAt    time.Time
	CurrencyCode string
	CurrencyName string
	DeletedAt    *time.Time
}

type BalanceFilter struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
}

func GetBalanceByUUID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Balance, error) {
	query := `
		SELECT b.uuid, b.wallet_id, b.currency_id, b.balance, b.updated_at, b.deleted_at, c.code, c.name
		FROM balances b
		LEFT JOIN currencies c ON b.currency_id = c.uuid
		WHERE b.uuid = $1 AND b.deleted_at IS NULL AND c.deleted_at IS NULL
	`
	var b Balance
	var code, name *string
	var deletedAt *time.Time
	err := pool.QueryRow(ctx, query, id).Scan(
		&b.UUID,
		&b.WalletID,
		&b.CurrencyID,
		&b.Balance,
		&b.UpdatedAt,
		&deletedAt,
		&code,
		&name,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get balance by uuid: %w", err)
	}
	if code != nil {
		b.CurrencyCode = *code
	}
	if name != nil {
		b.CurrencyName = *name
	}
	b.DeletedAt = deletedAt
	return &b, nil
}

func GetBalanceByWalletAndCurrency(ctx context.Context, pool *pgxpool.Pool, walletID, currencyID uuid.UUID) (*Balance, error) {
	query := `
		SELECT b.uuid, b.wallet_id, b.currency_id, b.balance, b.updated_at, b.deleted_at, c.code, c.name
		FROM balances b
		LEFT JOIN currencies c ON b.currency_id = c.uuid
		WHERE b.wallet_id = $1 AND b.currency_id = $2 AND b.deleted_at IS NULL AND c.deleted_at IS NULL
	`
	var b Balance
	var code, name *string
	var deletedAt *time.Time
	err := pool.QueryRow(ctx, query, walletID, currencyID).Scan(
		&b.UUID,
		&b.WalletID,
		&b.CurrencyID,
		&b.Balance,
		&b.UpdatedAt,
		&deletedAt,
		&code,
		&name,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get balance by wallet and currency: %w", err)
	}
	if code != nil {
		b.CurrencyCode = *code
	}
	if name != nil {
		b.CurrencyName = *name
	}
	b.DeletedAt = deletedAt
	return &b, nil
}

func GetBalances(ctx context.Context, pool *pgxpool.Pool, params PaginationParams, filter BalanceFilter) (PaginatedResult[Balance], error) {
	params = params.Normalize()

	baseQuery := `
		SELECT b.uuid, b.wallet_id, b.currency_id, b.balance, b.updated_at, b.deleted_at, c.code, c.name
		FROM balances b
		LEFT JOIN currencies c ON b.currency_id = c.uuid
		WHERE b.deleted_at IS NULL AND c.deleted_at IS NULL
	`
	countQuery := `SELECT COUNT(*) FROM balances WHERE deleted_at IS NULL`
	args := []any{}
	argIdx := 1

	if filter.WalletID != uuid.Nil {
		whereBase := fmt.Sprintf(" AND b.wallet_id = $%d", argIdx)
		whereCount := fmt.Sprintf(" AND wallet_id = $%d", argIdx)
		baseQuery += whereBase
		countQuery += whereCount
		args = append(args, filter.WalletID)
		argIdx++
	}

	if filter.CurrencyID != uuid.Nil {
		connectorBase := " AND "
		connectorCount := " AND "
		whereBase := fmt.Sprintf("%sb.currency_id = $%d", connectorBase, argIdx)
		whereCount := fmt.Sprintf("%scurrency_id = $%d", connectorCount, argIdx)
		baseQuery += whereBase
		countQuery += whereCount
		args = append(args, filter.CurrencyID)
		argIdx++
	}

	baseQuery += " ORDER BY b.updated_at DESC"

	return Paginate(ctx, pool, baseQuery, countQuery, args, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Balance, error) {
			var balances []Balance
			for rows.Next() {
				var b Balance
				var code, name *string
				var deletedAt *time.Time
				if err := rows.Scan(&b.UUID, &b.WalletID, &b.CurrencyID, &b.Balance, &b.UpdatedAt, &deletedAt, &code, &name); err != nil {
					return nil, err
				}
				if code != nil {
					b.CurrencyCode = *code
				}
				if name != nil {
					b.CurrencyName = *name
				}
				b.DeletedAt = deletedAt
				balances = append(balances, b)
			}
			return balances, nil
		},
	)
}
