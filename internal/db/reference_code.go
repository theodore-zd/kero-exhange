package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReferenceCode struct {
	UUID       uuid.UUID
	Code       string
	ProviderID uuid.UUID
	UsedAt     *time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

func GetReferenceCodeByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*ReferenceCode, error) {
	return getReferenceCodeByCode(ctx, pool, code, false)
}

func MarkReferenceCodeUsed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	return markReferenceCodeUsed(ctx, pool, id)
}

type CreateReferenceCodeParams struct {
	Code       string
	ProviderID uuid.UUID
	ExpiresAt  time.Time
}

func CreateReferenceCode(ctx context.Context, pool *pgxpool.Pool, params CreateReferenceCodeParams) (*ReferenceCode, error) {
	query := `
		INSERT INTO reference_codes (code, provider_id, expires_at)
		VALUES ($1, $2, $3)
		RETURNING uuid, code, provider_id, used_at, expires_at, created_at
	`
	var rc ReferenceCode
	var usedAt *time.Time
	err := pool.QueryRow(ctx, query, params.Code, params.ProviderID, params.ExpiresAt).Scan(
		&rc.UUID,
		&rc.Code,
		&rc.ProviderID,
		&usedAt,
		&rc.ExpiresAt,
		&rc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create reference code: %w", err)
	}
	rc.UsedAt = usedAt
	return &rc, nil
}

func GetReferenceCodeByCodeTx(ctx context.Context, tx pgx.Tx, code string) (*ReferenceCode, error) {
	return getReferenceCodeByCode(ctx, tx, code, true)
}

func MarkReferenceCodeUsedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return markReferenceCodeUsed(ctx, tx, id)
}

type referenceCodeQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func getReferenceCodeByCode(ctx context.Context, q referenceCodeQuerier, code string, forUpdate bool) (*ReferenceCode, error) {
	query := `
		SELECT uuid, code, provider_id, used_at, expires_at, created_at
		FROM reference_codes
		WHERE code = $1
	`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var rc ReferenceCode
	var usedAt *time.Time
	err := q.QueryRow(ctx, query, code).Scan(
		&rc.UUID,
		&rc.Code,
		&rc.ProviderID,
		&usedAt,
		&rc.ExpiresAt,
		&rc.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get reference code by code: %w", err)
	}
	rc.UsedAt = usedAt
	return &rc, nil
}

type referenceCodeExec interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func markReferenceCodeUsed(ctx context.Context, q referenceCodeExec, id uuid.UUID) error {
	query := `
		UPDATE reference_codes
		SET used_at = NOW()
		WHERE uuid = $1 AND used_at IS NULL
	`
	tag, err := q.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark reference code used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func DeleteReferenceCode(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	query := `DELETE FROM reference_codes WHERE uuid = $1`
	_, err := pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete reference code: %w", err)
	}
	return nil
}
