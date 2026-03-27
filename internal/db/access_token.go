package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateAccessTokenParams struct {
	WalletID  uuid.UUID
	Token     string
	ExpiresAt time.Time
}

type AccessToken struct {
	UUID       uuid.UUID
	WalletID   uuid.UUID
	Token      string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

func CreateAccessToken(ctx context.Context, pool *pgxpool.Pool, params CreateAccessTokenParams) (*AccessToken, error) {
	query := `
		INSERT INTO access_tokens (wallet_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING uuid, wallet_id, token, expires_at, created_at, last_used_at
	`
	var at AccessToken
	err := pool.QueryRow(ctx, query, params.WalletID, params.Token, params.ExpiresAt).Scan(
		&at.UUID,
		&at.WalletID,
		&at.Token,
		&at.ExpiresAt,
		&at.CreatedAt,
		&at.LastUsedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create access token: %w", err)
	}
	return &at, nil
}

func GetAccessTokenByToken(ctx context.Context, pool *pgxpool.Pool, token string) (*AccessToken, error) {
	query := `
		SELECT uuid, wallet_id, token, expires_at, created_at, last_used_at
		FROM access_tokens
		WHERE token = $1
	`
	var at AccessToken
	err := pool.QueryRow(ctx, query, token).Scan(
		&at.UUID,
		&at.WalletID,
		&at.Token,
		&at.ExpiresAt,
		&at.CreatedAt,
		&at.LastUsedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get access token by token: %w", err)
	}
	return &at, nil
}

func UpdateAccessTokenLastUsedAt(ctx context.Context, pool *pgxpool.Pool, tokenUUID uuid.UUID) error {
	query := `
		UPDATE access_tokens
		SET last_used_at = NOW()
		WHERE uuid = $1
	`
	_, err := pool.Exec(ctx, query, tokenUUID)
	if err != nil {
		return fmt.Errorf("update access token last used at (token_uuid=%s): %w", tokenUUID, err)
	}
	return nil
}

func DeleteAccessTokenByWallet(ctx context.Context, pool *pgxpool.Pool, walletID uuid.UUID) error {
	query := `
		DELETE FROM access_tokens
		WHERE wallet_id = $1
	`
	_, err := pool.Exec(ctx, query, walletID)
	if err != nil {
		return fmt.Errorf("delete access tokens by wallet (wallet_id=%s): %w", walletID, err)
	}
	return nil
}

func DeleteExpiredAccessTokens(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		DELETE FROM access_tokens
		WHERE expires_at < NOW()
	`
	result, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("delete expired access tokens: %w", err)
	}
	if result.RowsAffected() > 0 {
		return fmt.Errorf("deleted %d expired access tokens", result.RowsAffected())
	}
	return nil
}
