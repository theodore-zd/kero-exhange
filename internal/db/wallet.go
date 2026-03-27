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

type Wallet struct {
	UUID            uuid.UUID
	PassphraseHash  string
	AccessTokenHash string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func GetWalletByUUID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Wallet, error) {
	query := `
		SELECT uuid, passphrase_hash, access_token_hash, created_at, updated_at
		FROM wallet
		WHERE uuid = $1
	`
	var w Wallet
	var hash *string
	var passphraseHash *string
	err := pool.QueryRow(ctx, query, id).Scan(
		&w.UUID,
		&passphraseHash,
		&hash,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get wallet by uuid: %w", err)
	}
	if passphraseHash != nil {
		w.PassphraseHash = *passphraseHash
	}
	if hash != nil {
		w.AccessTokenHash = *hash
	}
	return &w, nil
}

func GetWallets(ctx context.Context, pool *pgxpool.Pool, params PaginationParams) (PaginatedResult[Wallet], error) {
	params = params.Normalize()

	baseQuery := `SELECT uuid, passphrase_hash, access_token_hash, created_at, updated_at FROM wallet`
	countQuery := `SELECT COUNT(*) FROM wallet`
	args := []any{}

	baseQuery += " ORDER BY created_at DESC"

	return Paginate(ctx, pool, baseQuery, countQuery, args, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Wallet, error) {
			var wallets []Wallet
			for rows.Next() {
				var w Wallet
				var passphraseHash *string
				var accessTokenHash *string
				if err := rows.Scan(&w.UUID, &passphraseHash, &accessTokenHash, &w.CreatedAt, &w.UpdatedAt); err != nil {
					return nil, err
				}
				if passphraseHash != nil {
					w.PassphraseHash = *passphraseHash
				}
				if accessTokenHash != nil {
					w.AccessTokenHash = *accessTokenHash
				}
				wallets = append(wallets, w)
			}
			return wallets, nil
		},
	)
}

func GetWalletByAccessTokenHash(ctx context.Context, pool *pgxpool.Pool, accessTokenHash string) (*Wallet, error) {
	query := `
		SELECT uuid, passphrase_hash, access_token_hash, created_at, updated_at
		FROM wallet
		WHERE access_token_hash = $1
	`
	var w Wallet
	var passphraseHash *string
	var storedAccessTokenHash *string
	err := pool.QueryRow(ctx, query, accessTokenHash).Scan(
		&w.UUID,
		&passphraseHash,
		&storedAccessTokenHash,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get wallet by access token hash: %w", err)
	}
	if passphraseHash != nil {
		w.PassphraseHash = *passphraseHash
	}
	if storedAccessTokenHash != nil {
		w.AccessTokenHash = *storedAccessTokenHash
	}
	return &w, nil
}

func GetWalletByPassphraseHash(ctx context.Context, pool *pgxpool.Pool, passphraseHash string) (*Wallet, error) {
	query := `
		SELECT uuid, passphrase_hash, access_token_hash, created_at, updated_at
		FROM wallet
		WHERE passphrase_hash = $1
	`
	var w Wallet
	var storedPassphraseHash *string
	var storedAccessTokenHash *string
	err := pool.QueryRow(ctx, query, passphraseHash).Scan(
		&w.UUID,
		&storedPassphraseHash,
		&storedAccessTokenHash,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get wallet by passphrase hash: %w", err)
	}
	if storedPassphraseHash != nil {
		w.PassphraseHash = *storedPassphraseHash
	}
	if storedAccessTokenHash != nil {
		w.AccessTokenHash = *storedAccessTokenHash
	}
	return &w, nil
}

type CreateWalletParams struct {
	PassphraseHash  string
	AccessTokenHash string
}

func CreateWallet(ctx context.Context, pool *pgxpool.Pool, params CreateWalletParams) (*Wallet, error) {
	return createWallet(ctx, pool, params)
}

func CreateWalletTx(ctx context.Context, tx pgx.Tx, params CreateWalletParams) (*Wallet, error) {
	return createWallet(ctx, tx, params)
}

type walletQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func createWallet(ctx context.Context, q walletQuerier, params CreateWalletParams) (*Wallet, error) {
	query := `
		INSERT INTO wallet (passphrase_hash, access_token_hash)
		VALUES ($1, $2)
		RETURNING uuid, passphrase_hash, access_token_hash, created_at, updated_at
	`
	var w Wallet
	var passphraseHash *string
	var accessTokenHash *string
	err := q.QueryRow(ctx, query, params.PassphraseHash, params.AccessTokenHash).Scan(
		&w.UUID,
		&passphraseHash,
		&accessTokenHash,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	if passphraseHash != nil {
		w.PassphraseHash = *passphraseHash
	}
	if accessTokenHash != nil {
		w.AccessTokenHash = *accessTokenHash
	}
	return &w, nil
}

func UpdateWalletAccessTokenHash(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, accessTokenHash string) error {
	return updateWalletAccessTokenHash(ctx, pool, id, accessTokenHash)
}

func UpdateWalletAccessTokenHashTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, accessTokenHash string) error {
	return updateWalletAccessTokenHash(ctx, tx, id, accessTokenHash)
}

func UpdateWalletPassphraseHash(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, passphraseHash string) error {
	query := `
		UPDATE wallet
		SET passphrase_hash = $2, updated_at = NOW()
		WHERE uuid = $1
	`
	_, err := pool.Exec(ctx, query, id, passphraseHash)
	if err != nil {
		return fmt.Errorf("update wallet passphrase hash: %w", err)
	}
	return nil
}

func UpdateWalletPassphraseHashTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, passphraseHash string) error {
	query := `
		UPDATE wallet
		SET passphrase_hash = $2, updated_at = NOW()
		WHERE uuid = $1
	`
	_, err := tx.Exec(ctx, query, id, passphraseHash)
	if err != nil {
		return fmt.Errorf("update wallet passphrase hash: %w", err)
	}
	return nil
}

type walletExec interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func updateWalletAccessTokenHash(ctx context.Context, q walletExec, id uuid.UUID, accessTokenHash string) error {
	query := `
		UPDATE wallet
		SET access_token_hash = $2, updated_at = NOW()
		WHERE uuid = $1
	`
	_, err := q.Exec(ctx, query, id, accessTokenHash)
	if err != nil {
		return fmt.Errorf("update wallet access token hash: %w", err)
	}
	return nil
}

func DeleteWallet(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	query := `DELETE FROM wallet WHERE uuid = $1`
	_, err := pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	return nil
}
