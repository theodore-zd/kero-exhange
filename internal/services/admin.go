package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/crypto"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type AdminService struct {
	pool         *pgxpool.Pool
	password     string
	passwordHash string
	auditService *AuditLogService
	balanceSvc   *BalanceService
}

func NewAdminService(pool *pgxpool.Pool, password, passwordHash string) *AdminService {
	return &AdminService{
		pool:         pool,
		password:     password,
		passwordHash: passwordHash,
		auditService: NewAuditLogService(pool),
		balanceSvc:   NewBalanceService(pool),
	}
}

type CreateCurrencyParams struct {
	Code        string
	Name        string
	Description *string
}

type IssueCurrencyAdminParams struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Amount     string
	Reference  *string
}

func (s *AdminService) CheckAdminPassword(password string) bool {
	if s.passwordHash != "" {
		return bcrypt.CompareHashAndPassword([]byte(s.passwordHash), []byte(password)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(s.password), []byte(password)) == 1
}

func (s *AdminService) ListProviders(ctx context.Context, params db.PaginationParams) (db.PaginatedResult[db.Provider], error) {
	return db.ListProviders(ctx, s.pool, params)
}

func (s *AdminService) CreateProvider(ctx context.Context, name string, adminUser, ipAddress, userAgent string) (*db.Provider, string, error) {
	apiKey := s.generateAPIKey()
	if apiKey == "" {
		return nil, "", fmt.Errorf("generate provider api key")
	}

	apiKeyHash := crypto.HashPassphrase(apiKey)
	provider, err := db.CreateProvider(ctx, s.pool, apiKeyHash, name)
	if err != nil {
		return nil, "", fmt.Errorf("create provider: %w", err)
	}

	if err := s.auditService.LogProviderCreated(ctx, provider, apiKey, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log provider creation", "error", err)
	}

	return provider, apiKey, nil
}

func (s *AdminService) UpdateProviderAPIKey(ctx context.Context, id uuid.UUID, apiKey string, adminUser, ipAddress, userAgent string) error {
	apiKeyHash := crypto.HashPassphrase(apiKey)
	if err := db.UpdateProviderAPIKey(ctx, s.pool, id, apiKeyHash); err != nil {
		return fmt.Errorf("update provider api key (provider_id=%s): %w", id, err)
	}

	if err := s.auditService.LogProviderApiKeyUpdated(ctx, id, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log provider API key update", "error", err)
	}

	return nil
}

func (s *AdminService) DeleteProvider(ctx context.Context, id uuid.UUID, adminUser, ipAddress, userAgent string) error {
	if err := db.DeleteProvider(ctx, s.pool, id); err != nil {
		return fmt.Errorf("delete provider (provider_id=%s): %w", id, err)
	}

	if err := s.auditService.LogProviderDeleted(ctx, id, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log provider deletion", "error", err)
	}

	return nil
}

func (s *AdminService) ListWallets(ctx context.Context, params db.PaginationParams) (db.PaginatedResult[db.Wallet], error) {
	return db.GetWallets(ctx, s.pool, params)
}

func (s *AdminService) GetAllCurrencies(ctx context.Context, params db.PaginationParams) (db.PaginatedResult[db.Currency], error) {
	return db.GetCurrencies(ctx, s.pool, params)
}

func (s *AdminService) CreateWallet(ctx context.Context, adminUser, ipAddress, userAgent string) (*db.Wallet, string, string, error) {
	passphrase, err := crypto.GenerateSecurePassphrase()
	if err != nil {
		return nil, "", "", fmt.Errorf("generate passphrase: %w", err)
	}

	passphraseHash := crypto.HashPassphrase(passphrase)
	accessToken := generateAccessToken()
	accessTokenHash := crypto.HashAccessToken(passphrase, accessToken)

	wallet, err := db.CreateWallet(ctx, s.pool, db.CreateWalletParams{
		PassphraseHash:  passphraseHash,
		AccessTokenHash: accessTokenHash,
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("create wallet: %w", err)
	}

	if err := s.auditService.LogWalletCreated(ctx, wallet, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log wallet creation", "error", err)
	}

	return wallet, passphrase, accessToken, nil
}

func (s *AdminService) RegenerateWalletPassphrase(ctx context.Context, id uuid.UUID, adminUser, ipAddress, userAgent string) (*db.Wallet, string, string, error) {
	wallet, err := db.GetWalletByUUID(ctx, s.pool, id)
	if err != nil {
		return nil, "", "", fmt.Errorf("get wallet (wallet_id=%s): %w", id, err)
	}
	if wallet == nil {
		return nil, "", "", ErrWalletNotFound
	}

	passphrase, err := crypto.GenerateSecurePassphrase()
	if err != nil {
		return nil, "", "", fmt.Errorf("generate passphrase: %w", err)
	}

	passphraseHash := crypto.HashPassphrase(passphrase)
	accessToken := generateAccessToken()
	accessTokenHash := crypto.HashAccessToken(passphrase, accessToken)

	err = db.UpdateWalletPassphraseHash(ctx, s.pool, id, passphraseHash)
	if err != nil {
		return nil, "", "", fmt.Errorf("update wallet passphrase hash (wallet_id=%s): %w", id, err)
	}

	err = db.UpdateWalletAccessTokenHash(ctx, s.pool, id, accessTokenHash)
	if err != nil {
		return nil, "", "", fmt.Errorf("update wallet access token hash (wallet_id=%s): %w", id, err)
	}

	wallet.PassphraseHash = passphraseHash
	wallet.AccessTokenHash = accessTokenHash

	if err := s.auditService.LogWalletPassphraseRegenerated(ctx, id, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log wallet passphrase regeneration", "error", err)
	}

	return wallet, passphrase, accessToken, nil
}

func (s *AdminService) DeleteWallet(ctx context.Context, id uuid.UUID, adminUser, ipAddress, userAgent string) error {
	if err := db.DeleteWallet(ctx, s.pool, id); err != nil {
		return fmt.Errorf("delete wallet (wallet_id=%s): %w", id, err)
	}

	if err := s.auditService.LogWalletDeleted(ctx, id, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log wallet deletion", "error", err)
	}

	return nil
}

func (s *AdminService) GenerateAdminToken() string {
	return generateSecureHex(32)
}

func (s *AdminService) GetWalletBalances(ctx context.Context, walletID uuid.UUID, params db.PaginationParams) (db.PaginatedResult[db.Balance], error) {
	filter := db.BalanceFilter{WalletID: walletID}
	return db.GetBalances(ctx, s.pool, params, filter)
}

func (s *AdminService) generateAPIKey() string {
	token := generateSecureHex(32)
	if token == "" {
		return ""
	}
	return "kero_" + token
}

func generateSecureHex(length int) string {
	if length <= 0 {
		return ""
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

func (s *AdminService) CreateCurrency(ctx context.Context, params CreateCurrencyParams, adminUser, ipAddress, userAgent string) (*db.Currency, error) {
	currency, err := db.CreateCurrency(ctx, s.pool, db.CreateCurrencyParams{
		Code:        params.Code,
		Name:        params.Name,
		Description: params.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("create currency: %w", err)
	}

	if err := s.auditService.LogCurrencyCreated(ctx, currency, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log currency creation", "error", err)
	}

	return currency, nil
}

func (s *AdminService) DeleteCurrency(ctx context.Context, id uuid.UUID, adminUser, ipAddress, userAgent string) error {
	if err := db.DeleteCurrency(ctx, s.pool, id); err != nil {
		common.LogError("DeleteCurrency service failed", "currency_id", id, "admin_user", adminUser, "error", err)
		return fmt.Errorf("delete currency (currency_id=%s): %w", id, err)
	}

	if err := s.auditService.LogCurrencyDeleted(ctx, id, adminUser, ipAddress, userAgent); err != nil {
		common.LogError("Failed to log currency deletion", "currency_id", id, "error", err)
	}

	common.LogInfo("Currency deleted", "currency_id", id, "admin_user", adminUser)
	return nil
}

func (s *AdminService) IssueCurrency(ctx context.Context, params IssueCurrencyAdminParams, adminUser, ipAddress, userAgent string) (*IssueCurrencyResult, error) {
	amount, err := parseDecimal(params.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	result, err := s.balanceSvc.IssueCurrency(ctx, IssueCurrencyParams{
		WalletID:   params.WalletID,
		CurrencyID: params.CurrencyID,
		Amount:     amount,
		Reference:  params.Reference,
	})
	if err != nil {
		return nil, fmt.Errorf("issue currency: %w", err)
	}

	if result.Transaction != nil {
		if err := s.auditService.LogCurrencyIssued(ctx, params.WalletID, params.CurrencyID, params.Amount, result.Transaction.UUID, adminUser, ipAddress, userAgent); err != nil {
			common.LogError("Failed to log currency issuance", "error", err)
		}
	}

	return result, nil
}

func parseDecimal(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	return decimal.NewFromString(s)
}
