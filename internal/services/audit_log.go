package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

type AuditLogService struct {
	pool *pgxpool.Pool
}

func NewAuditLogService(pool *pgxpool.Pool) *AuditLogService {
	return &AuditLogService{pool: pool}
}

type CreateAuditLogRequest struct {
	Action     string
	EntityType string
	EntityID   uuid.UUID
	Details    map[string]interface{}
	AdminUser  string
	IPAddress  string
	UserAgent  string
}

func (s *AuditLogService) LogAction(ctx context.Context, req CreateAuditLogRequest) error {
	params := db.CreateAuditLogParams{
		Action:     req.Action,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Details:    req.Details,
		AdminUser:  req.AdminUser,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
	}
	_, err := db.CreateAuditLog(ctx, s.pool, params)
	if err != nil {
		return fmt.Errorf("log audit action %s on %s: %w", req.Action, req.EntityType, err)
	}
	return nil
}

func (s *AuditLogService) GetAuditLogs(ctx context.Context, params db.PaginationParams, filter db.AuditLogFilter) (db.PaginatedResult[db.AuditLog], error) {
	return db.GetAuditLogs(ctx, s.pool, params, filter)
}

func (s *AuditLogService) LogCurrencyCreated(ctx context.Context, currency *db.Currency, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{
		"code":        currency.Code,
		"name":        currency.Name,
		"description": currency.Description,
	}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "currency_created",
		EntityType: "currency",
		EntityID:   currency.UUID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogCurrencyIssued(ctx context.Context, walletID, currencyID uuid.UUID, amount string, transactionID uuid.UUID, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{
		"wallet_id":      walletID.String(),
		"currency_id":    currencyID.String(),
		"amount":         amount,
		"transaction_id": transactionID.String(),
	}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "currency_issued",
		EntityType: "wallet",
		EntityID:   walletID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogProviderCreated(ctx context.Context, provider *db.Provider, apiKey, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{
		"name":         provider.Name,
		"api_key_hash": provider.APIKeyHash,
	}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "provider_created",
		EntityType: "provider",
		EntityID:   provider.UUID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogProviderDeleted(ctx context.Context, providerID uuid.UUID, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "provider_deleted",
		EntityType: "provider",
		EntityID:   providerID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogWalletCreated(ctx context.Context, wallet *db.Wallet, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{
		"uuid": wallet.UUID.String(),
	}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "wallet_created",
		EntityType: "wallet",
		EntityID:   wallet.UUID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogWalletDeleted(ctx context.Context, walletID uuid.UUID, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "wallet_deleted",
		EntityType: "wallet",
		EntityID:   walletID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogWalletPassphraseRegenerated(ctx context.Context, walletID uuid.UUID, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "wallet_passphrase_regenerated",
		EntityType: "wallet",
		EntityID:   walletID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogProviderApiKeyUpdated(ctx context.Context, providerID uuid.UUID, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "provider_api_key_updated",
		EntityType: "provider",
		EntityID:   providerID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

func (s *AuditLogService) LogCurrencyDeleted(ctx context.Context, currencyID uuid.UUID, adminUser, ipAddress, userAgent string) error {
	details := map[string]interface{}{}
	return s.LogAction(ctx, CreateAuditLogRequest{
		Action:     "currency_deleted",
		EntityType: "currency",
		EntityID:   currencyID,
		Details:    details,
		AdminUser:  adminUser,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}
