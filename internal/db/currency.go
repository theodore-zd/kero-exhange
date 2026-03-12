package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
}

func GetCurrencyByUUID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Currency, error) {
	query := `
		SELECT uuid, code, name, description, created_at
		FROM currencies
		WHERE uuid = $1
	`
	var c Currency
	err := pool.QueryRow(ctx, query, id).Scan(
		&c.UUID,
		&c.Code,
		&c.Name,
		&c.Description,
		&c.CreatedAt,
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
		SELECT uuid, code, name, description, created_at
		FROM currencies
		WHERE code = $1
	`
	var c Currency
	err := pool.QueryRow(ctx, query, code).Scan(
		&c.UUID,
		&c.Code,
		&c.Name,
		&c.Description,
		&c.CreatedAt,
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

	baseQuery := `SELECT uuid, code, name, description, created_at FROM currencies ORDER BY code ASC`
	countQuery := `SELECT COUNT(*) FROM currencies`

	return Paginate(ctx, pool, baseQuery, countQuery, nil, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Currency, error) {
			var currencies []Currency
			for rows.Next() {
				var c Currency
				if err := rows.Scan(&c.UUID, &c.Code, &c.Name, &c.Description, &c.CreatedAt); err != nil {
					return nil, err
				}
				currencies = append(currencies, c)
			}
			return currencies, nil
		},
	)
}
