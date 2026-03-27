package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/go-common"
)

type CreateCurrencyParams struct {
	Code        string
	Name        string
	Description *string
}

func CreateCurrency(ctx context.Context, pool *pgxpool.Pool, params CreateCurrencyParams) (*Currency, error) {
	query := `
		INSERT INTO currencies (code, name, description)
		VALUES ($1, $2, $3)
		RETURNING uuid, code, name, description, created_at
	`
	var c Currency
	err := pool.QueryRow(ctx, query, params.Code, params.Name, params.Description).Scan(
		&c.UUID,
		&c.Code,
		&c.Name,
		&c.Description,
		&c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create currency: %w", err)
	}
	return &c, nil
}

type Currency struct {
	UUID        uuid.UUID
	Code        string
	Name        string
	Description *string
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

func GetCurrencyByUUID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Currency, error) {
	query := `
		SELECT uuid, code, name, description, created_at, deleted_at
		FROM currencies
		WHERE uuid = $1 AND deleted_at IS NULL
	`
	var c Currency
	err := pool.QueryRow(ctx, query, id).Scan(
		&c.UUID,
		&c.Code,
		&c.Name,
		&c.Description,
		&c.CreatedAt,
		&c.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get currency by uuid: %w", err)
	}
	return &c, nil
}

func GetCurrencyByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Currency, error) {
	query := `
		SELECT uuid, code, name, description, created_at, deleted_at
		FROM currencies
		WHERE code = $1 AND deleted_at IS NULL
	`
	var c Currency
	err := pool.QueryRow(ctx, query, code).Scan(
		&c.UUID,
		&c.Code,
		&c.Name,
		&c.Description,
		&c.CreatedAt,
		&c.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get currency by code: %w", err)
	}
	return &c, nil
}

func GetCurrencies(ctx context.Context, pool *pgxpool.Pool, params PaginationParams) (PaginatedResult[Currency], error) {
	params = params.Normalize()

	baseQuery := `SELECT uuid, code, name, description, created_at, deleted_at FROM currencies WHERE deleted_at IS NULL ORDER BY code ASC`
	countQuery := `SELECT COUNT(*) FROM currencies WHERE deleted_at IS NULL`

	return Paginate(ctx, pool, baseQuery, countQuery, nil, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Currency, error) {
			var currencies []Currency
			for rows.Next() {
				var c Currency
				if err := rows.Scan(&c.UUID, &c.Code, &c.Name, &c.Description, &c.CreatedAt, &c.DeletedAt); err != nil {
					return nil, err
				}
				currencies = append(currencies, c)
			}
			return currencies, nil
		},
	)
}

func DeleteCurrency(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE transactions SET deleted_at = NOW() WHERE currency_id = $1`, id)
	if err != nil {
		return fmt.Errorf("soft delete transactions: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE balances SET deleted_at = NOW() WHERE currency_id = $1`, id)
	if err != nil {
		return fmt.Errorf("soft delete balances: %w", err)
	}

	result, err := tx.Exec(ctx, `UPDATE currencies SET deleted_at = NOW() WHERE uuid = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete currency: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("currency not found or already deleted")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	common.LogInfo("SoftDeleteCurrency: deleted currency and cascade", "currency_id", id)
	return nil
}
