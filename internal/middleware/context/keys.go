package context

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	WalletUUIDKey      contextKey = "wallet_uuid"
	ProviderUUIDKey    contextKey = "provider_uuid"
	AccessTokenHashKey contextKey = "access_token_hash"
)

func WithWalletUUID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, WalletUUIDKey, id)
}

func GetWalletUUID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(WalletUUIDKey).(uuid.UUID)
	return id, ok
}

func WithProviderUUID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ProviderUUIDKey, id)
}

func GetProviderUUID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ProviderUUIDKey).(uuid.UUID)
	return id, ok
}

func WithAccessTokenHash(ctx context.Context, hash string) context.Context {
	return context.WithValue(ctx, AccessTokenHashKey, hash)
}

func GetAccessTokenHash(ctx context.Context) (string, bool) {
	hash, ok := ctx.Value(AccessTokenHashKey).(string)
	return hash, ok
}
