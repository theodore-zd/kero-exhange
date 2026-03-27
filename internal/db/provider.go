package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Provider struct {
	UUID       uuid.UUID
	APIKeyHash string
	Name       string
	CreatedAt  time.Time
}

func GetProviderByAPIKeyHash(ctx context.Context, pool *pgxpool.Pool, apiKeyHash string) (*Provider, error) {
	query := `
		SELECT uuid, api_key_hash, name, created_at
		FROM providers
		WHERE api_key_hash = $1
	`
	var p Provider
	err := pool.QueryRow(ctx, query, apiKeyHash).Scan(
		&p.UUID,
		&p.APIKeyHash,
		&p.Name,
		&p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get provider by api key hash: %w", err)
	}
	return &p, nil
}

func CreateProvider(ctx context.Context, pool *pgxpool.Pool, apiKeyHash, name string) (*Provider, error) {
	query := `
		INSERT INTO providers (api_key_hash, name)
		VALUES ($1, $2)
		RETURNING uuid, api_key_hash, name, created_at
	`
	var p Provider
	err := pool.QueryRow(ctx, query, apiKeyHash, name).Scan(
		&p.UUID,
		&p.APIKeyHash,
		&p.Name,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}
	return &p, nil
}

func ListProviders(ctx context.Context, pool *pgxpool.Pool, params PaginationParams) (PaginatedResult[Provider], error) {
	params = params.Normalize()

	baseQuery := `SELECT uuid, api_key_hash, name, created_at FROM providers`
	countQuery := `SELECT COUNT(*) FROM providers`
	args := []any{}

	baseQuery += " ORDER BY created_at DESC"

	return Paginate(ctx, pool, baseQuery, countQuery, args, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Provider, error) {
			var providers []Provider
			for rows.Next() {
				var p Provider
				if err := rows.Scan(&p.UUID, &p.APIKeyHash, &p.Name, &p.CreatedAt); err != nil {
					return nil, err
				}
				providers = append(providers, p)
			}
			return providers, nil
		},
	)
}

func DeleteProvider(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	query := `DELETE FROM providers WHERE uuid = $1`
	_, err := pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	return nil
}

func UpdateProviderAPIKey(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, apiKeyHash string) error {
	query := `
		UPDATE providers
		SET api_key_hash = $2
		WHERE uuid = $1
	`
	_, err := pool.Exec(ctx, query, id, apiKeyHash)
	if err != nil {
		return fmt.Errorf("update provider api key: %w", err)
	}
	return nil
}
