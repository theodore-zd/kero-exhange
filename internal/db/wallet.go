package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Wallet struct {
	UUID            uuid.UUID
	PublicKey       string
	AccessTokenHash *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type WalletFilter struct {
	PublicKey string
}

func GetWalletByUUID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Wallet, error) {
	query := `
		SELECT uuid, public_key, access_token_hash, created_at, updated_at
		FROM wallet
		WHERE uuid = $1
	`
	var w Wallet
	err := pool.QueryRow(ctx, query, id).Scan(
		&w.UUID,
		&w.PublicKey,
		&w.AccessTokenHash,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get wallet by uuid: %w", err)
	}
	return &w, nil
}

func GetWallets(ctx context.Context, pool *pgxpool.Pool, params PaginationParams, filter WalletFilter) (PaginatedResult[Wallet], error) {
	params = params.Normalize()

	baseQuery := `SELECT uuid, public_key, access_token_hash, created_at, updated_at FROM wallet`
	countQuery := `SELECT COUNT(*) FROM wallet`
	args := []any{}
	argIdx := 1

	if filter.PublicKey != "" {
		where := fmt.Sprintf(" WHERE public_key LIKE $%d", argIdx)
		baseQuery += where
		countQuery += where
		args = append(args, filter.PublicKey+"%")
		argIdx++
	}

	baseQuery += " ORDER BY created_at DESC"

	return Paginate(ctx, pool, baseQuery, countQuery, args, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Wallet, error) {
			var wallets []Wallet
			for rows.Next() {
				var w Wallet
				if err := rows.Scan(&w.UUID, &w.PublicKey, &w.AccessTokenHash, &w.CreatedAt, &w.UpdatedAt); err != nil {
					return nil, err
				}
				wallets = append(wallets, w)
			}
			return wallets, nil
		},
	)
}
