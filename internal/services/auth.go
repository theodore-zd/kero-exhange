package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/crypto"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

const (
	tokenExpiry = 24 * time.Hour
)

var (
	ErrInvalidReferenceCode = errors.New("invalid reference code")
	ErrReferenceCodeUsed    = errors.New("reference code already used")
	ErrReferenceCodeExpired = errors.New("reference code expired")
	ErrInvalidAPIKey        = errors.New("invalid API key")
	ErrWalletNotFound       = errors.New("wallet not found")
)

const (
	codeLength          = 16
	referenceCodeExpiry = 1 * time.Hour
)

type AuthService struct {
	pool *pgxpool.Pool
}

func NewAuthService(pool *pgxpool.Pool) *AuthService {
	return &AuthService{pool: pool}
}

func (s *AuthService) GenerateReferenceCode(ctx context.Context, providerID uuid.UUID) (*db.ReferenceCode, error) {
	code, err := generateRandomCode(codeLength)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(referenceCodeExpiry)
	rc, err := db.CreateReferenceCode(ctx, s.pool, db.CreateReferenceCodeParams{
		Code:       code,
		ProviderID: providerID,
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func (s *AuthService) SignUp(ctx context.Context, referenceCode string) (accessToken, walletUUID, passphrase string, err error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", "", fmt.Errorf("begin signup transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	rc, err := db.GetReferenceCodeByCodeTx(ctx, tx, referenceCode)
	if err != nil {
		return "", "", "", err
	}
	if rc == nil {
		return "", "", "", ErrInvalidReferenceCode
	}
	if rc.UsedAt != nil {
		return "", "", "", ErrReferenceCodeUsed
	}
	if time.Now().After(rc.ExpiresAt) {
		return "", "", "", ErrReferenceCodeExpired
	}

	passphrase, err = crypto.GenerateSecurePassphrase()
	if err != nil {
		return "", "", "", fmt.Errorf("generate passphrase: %w", err)
	}

	passphraseHash := crypto.HashPassphrase(passphrase)
	accessToken = generateAccessToken()
	accessTokenHash := crypto.HashAccessToken(passphrase, accessToken)

	wallet, err := db.CreateWalletTx(ctx, tx, db.CreateWalletParams{
		PassphraseHash:  passphraseHash,
		AccessTokenHash: accessTokenHash,
	})
	if err != nil {
		return "", "", "", fmt.Errorf("create wallet: %w", err)
	}

	if err := db.MarkReferenceCodeUsedTx(ctx, tx, rc.UUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", ErrReferenceCodeUsed
		}
		return "", "", "", fmt.Errorf("mark reference code used (reference_code_id=%s): %w", rc.UUID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", "", fmt.Errorf("commit signup transaction: %w", err)
	}

	return accessToken, wallet.UUID.String(), passphrase, nil
}

func (s *AuthService) SignIn(ctx context.Context, passphrase string) (accessToken, walletUUID string, err error) {
	passphraseHash := crypto.HashPassphrase(passphrase)
	wallet, err := db.GetWalletByPassphraseHash(ctx, s.pool, passphraseHash)
	if err != nil {
		return "", "", err
	}
	if wallet == nil {
		return "", "", ErrWalletNotFound
	}

	accessToken = generateAccessToken()
	accessTokenHash := crypto.HashAccessToken(passphrase, accessToken)

	err = db.UpdateWalletAccessTokenHash(ctx, s.pool, wallet.UUID, accessTokenHash)
	if err != nil {
		return "", "", fmt.Errorf("update wallet access token hash (wallet_id=%s): %w", wallet.UUID, err)
	}

	expiresAt := time.Now().Add(tokenExpiry)
	_, err = db.CreateAccessToken(ctx, s.pool, db.CreateAccessTokenParams{
		WalletID:  wallet.UUID,
		Token:     accessToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", "", fmt.Errorf("create access token: %w", err)
	}

	common.LogInfo("Wallet signed in", "wallet_id", wallet.UUID.String())

	return accessToken, wallet.UUID.String(), nil
}

func (s *AuthService) SignOut(ctx context.Context, walletUUID uuid.UUID) error {
	return db.DeleteAccessTokenByWallet(ctx, s.pool, walletUUID)
}

func (s *AuthService) CleanupExpiredTokens(ctx context.Context) error {
	return db.DeleteExpiredAccessTokens(ctx, s.pool)
}

func (s *AuthService) GetProviderByAPIKey(ctx context.Context, apiKey string) (*db.Provider, error) {
	apiKeyHash := crypto.HashPassphrase(apiKey)
	return db.GetProviderByAPIKeyHash(ctx, s.pool, apiKeyHash)
}

func generateRandomCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = alphanumericChars[int(bytes[i])%len(alphanumericChars)]
	}
	return string(result), nil
}

func generateAccessToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

var alphanumericChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
