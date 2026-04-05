package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/crypto"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

var (
	seedWalletAUUID  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	seedWalletBUUID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	seedProviderUUID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

const (
	seedWalletAPassphrase  = "test-user-alpha"
	seedWalletBPassphrase  = "test-user-beta"
	seedWalletAAccessToken = "seed-access-token-wallet-alpha"
	seedWalletBAccessToken = "seed-access-token-wallet-beta"
	seedProviderAPIKey     = "seed-provider-api-key"
	seedProviderName       = "Seed Test Provider"
)

type SeedService struct {
	pool *pgxpool.Pool
}

func NewSeedService(pool *pgxpool.Pool) *SeedService {
	return &SeedService{pool: pool}
}

func (s *SeedService) Seed(ctx context.Context) error {
	// Check if seed data already exists
	existing, err := db.GetWalletByUUID(ctx, s.pool, seedWalletAUUID)
	if err != nil {
		return fmt.Errorf("seed: check existing wallet: %w", err)
	}
	if existing != nil {
		common.LogInfo("Seed data already exists, skipping")
		return nil
	}

	common.LogInfo("Seeding development data...")

	// Ensure USD currency exists
	usd, err := db.GetCurrencyByCode(ctx, s.pool, "USD")
	if err != nil {
		return fmt.Errorf("seed: get USD currency: %w", err)
	}
	if usd == nil {
		usdDesc := "US Dollar"
		usd, err = db.CreateCurrency(ctx, s.pool, db.CreateCurrencyParams{
			Code:        "USD",
			Name:        "US Dollar",
			Description: &usdDesc,
		})
		if err != nil {
			return fmt.Errorf("seed: create USD currency: %w", err)
		}
	}

	// Ensure EUR currency exists
	eur, err := db.GetCurrencyByCode(ctx, s.pool, "EUR")
	if err != nil {
		return fmt.Errorf("seed: get EUR currency: %w", err)
	}
	if eur == nil {
		eurDesc := "Euro"
		eur, err = db.CreateCurrency(ctx, s.pool, db.CreateCurrencyParams{
			Code:        "EUR",
			Name:        "Euro",
			Description: &eurDesc,
		})
		if err != nil {
			return fmt.Errorf("seed: create EUR currency: %w", err)
		}
	}

	// Create wallet A with specific UUID
	walletAPassphraseHash := crypto.HashPassphrase(seedWalletAPassphrase)
	walletATokenHash := crypto.HashAccessToken(seedWalletAUUID.String(), seedWalletAAccessToken)
	if err := s.insertWalletWithUUID(ctx, seedWalletAUUID, walletAPassphraseHash, walletATokenHash); err != nil {
		return fmt.Errorf("seed: create wallet A: %w", err)
	}

	// Create wallet B with specific UUID
	walletBPassphraseHash := crypto.HashPassphrase(seedWalletBPassphrase)
	walletBTokenHash := crypto.HashAccessToken(seedWalletBUUID.String(), seedWalletBAccessToken)
	if err := s.insertWalletWithUUID(ctx, seedWalletBUUID, walletBPassphraseHash, walletBTokenHash); err != nil {
		return fmt.Errorf("seed: create wallet B: %w", err)
	}

	// Create access tokens for both wallets (far future expiry)
	tokenExpiry := time.Now().Add(365 * 24 * time.Hour)

	if _, err := db.CreateAccessToken(ctx, s.pool, db.CreateAccessTokenParams{
		WalletID:  seedWalletAUUID,
		Token:     walletATokenHash,
		ExpiresAt: tokenExpiry,
	}); err != nil {
		return fmt.Errorf("seed: create access token for wallet A: %w", err)
	}

	if _, err := db.CreateAccessToken(ctx, s.pool, db.CreateAccessTokenParams{
		WalletID:  seedWalletBUUID,
		Token:     walletBTokenHash,
		ExpiresAt: tokenExpiry,
	}); err != nil {
		return fmt.Errorf("seed: create access token for wallet B: %w", err)
	}

	// Fund wallet A: 10,000 USD, 5,000 EUR
	seedRef := "seed"
	if _, err := db.CreateTransaction(ctx, s.pool, db.CreateTransactionParams{
		WalletID:   seedWalletAUUID,
		CurrencyID: usd.UUID,
		Amount:     decimal.NewFromInt(10000),
		Type:       db.TransactionTypeDeposit,
		Reference:  &seedRef,
	}); err != nil {
		return fmt.Errorf("seed: fund wallet A USD: %w", err)
	}

	if _, err := db.CreateTransaction(ctx, s.pool, db.CreateTransactionParams{
		WalletID:   seedWalletAUUID,
		CurrencyID: eur.UUID,
		Amount:     decimal.NewFromInt(5000),
		Type:       db.TransactionTypeDeposit,
		Reference:  &seedRef,
	}); err != nil {
		return fmt.Errorf("seed: fund wallet A EUR: %w", err)
	}

	// Fund wallet B: 5,000 USD, 2,500 EUR
	if _, err := db.CreateTransaction(ctx, s.pool, db.CreateTransactionParams{
		WalletID:   seedWalletBUUID,
		CurrencyID: usd.UUID,
		Amount:     decimal.NewFromInt(5000),
		Type:       db.TransactionTypeDeposit,
		Reference:  &seedRef,
	}); err != nil {
		return fmt.Errorf("seed: fund wallet B USD: %w", err)
	}

	if _, err := db.CreateTransaction(ctx, s.pool, db.CreateTransactionParams{
		WalletID:   seedWalletBUUID,
		CurrencyID: eur.UUID,
		Amount:     decimal.NewFromInt(2500),
		Type:       db.TransactionTypeDeposit,
		Reference:  &seedRef,
	}); err != nil {
		return fmt.Errorf("seed: fund wallet B EUR: %w", err)
	}

	// Create test provider with specific UUID
	providerAPIKeyHash := crypto.HashPassphrase(seedProviderAPIKey)
	if err := s.insertProviderWithUUID(ctx, seedProviderUUID, providerAPIKeyHash, seedProviderName); err != nil {
		return fmt.Errorf("seed: create provider: %w", err)
	}

	// Log all credentials
	common.LogInfo("=== SEED DATA CREDENTIALS ===")
	common.LogInfo("Wallet A", "uuid", seedWalletAUUID, "passphrase", seedWalletAPassphrase, "access_token", seedWalletAAccessToken)
	common.LogInfo("Wallet B", "uuid", seedWalletBUUID, "passphrase", seedWalletBPassphrase, "access_token", seedWalletBAccessToken)
	common.LogInfo("Provider", "uuid", seedProviderUUID, "api_key", seedProviderAPIKey)
	common.LogInfo("=== END SEED DATA ===")

	return nil
}

func (s *SeedService) insertWalletWithUUID(ctx context.Context, id uuid.UUID, passphraseHash, accessTokenHash string) error {
	query := `
		INSERT INTO wallet (uuid, passphrase_hash, access_token_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (uuid) DO NOTHING
	`
	_, err := s.pool.Exec(ctx, query, id, passphraseHash, accessTokenHash)
	if err != nil {
		return fmt.Errorf("insert wallet with uuid: %w", err)
	}
	return nil
}

func (s *SeedService) insertProviderWithUUID(ctx context.Context, id uuid.UUID, apiKeyHash, name string) error {
	query := `
		INSERT INTO providers (uuid, api_key_hash, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (uuid) DO NOTHING
	`
	_, err := s.pool.Exec(ctx, query, id, apiKeyHash, name)
	if err != nil {
		return fmt.Errorf("insert provider with uuid: %w", err)
	}
	return nil
}
