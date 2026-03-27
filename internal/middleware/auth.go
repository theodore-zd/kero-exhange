package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/crypto"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/middleware/context"
)

func APIKeyMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				common.WriteJSONError(w, http.StatusUnauthorized, "MISSING_API_KEY", "Missing X-API-Key header", nil)
				return
			}

			apiKeyHash := crypto.HashPassphrase(apiKey)
			provider, err := db.GetProviderByAPIKeyHash(r.Context(), pool, apiKeyHash)
			if err != nil {
				common.LogError("APIKeyMiddleware: failed to get provider", "error", err)
				common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_API_KEY", "Invalid API key", nil)
				return
			}
			if provider == nil {
				common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_API_KEY", "Invalid API key", nil)
				return
			}

			ctx := context.WithProviderUUID(r.Context(), provider.UUID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AccessTokenMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				common.WriteJSONError(w, http.StatusUnauthorized, "MISSING_AUTH_TOKEN", "Missing Authorization header", nil)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_AUTH_FORMAT", "Invalid authorization format. Use: Bearer <token>", nil)
				return
			}

			accessToken := parts[1]

			token, err := db.GetAccessTokenByToken(r.Context(), pool, accessToken)
			if err != nil {
				common.LogError("AccessTokenMiddleware: failed to get token", "error", err)
				common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_ACCESS_TOKEN", "Invalid or expired access token", nil)
				return
			}
			if token == nil {
				common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_ACCESS_TOKEN", "Invalid or expired access token", nil)
				return
			}

			if time.Now().After(token.ExpiresAt) {
				common.WriteJSONError(w, http.StatusUnauthorized, "EXPIRED_ACCESS_TOKEN", "Access token has expired", nil)
				return
			}

			if err := db.UpdateAccessTokenLastUsedAt(r.Context(), pool, token.UUID); err != nil {
				common.LogError("AccessTokenMiddleware: failed to update last used", "token_id", token.UUID, "error", err)
			}

			ctx := context.WithWalletUUID(r.Context(), token.WalletID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
