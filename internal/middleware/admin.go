package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wispberry-tech/go-common"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type AdminTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
}

var adminTokenStore = &AdminTokenStore{
	tokens: make(map[string]time.Time),
}

func (s *AdminTokenStore) StoreToken(token string, expiry time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = time.Now().Add(expiry)
}

func (s *AdminTokenStore) ValidateToken(token string) bool {
	s.mu.RLock()
	expiry, exists := s.tokens[token]
	s.mu.RUnlock()
	if !exists {
		return false
	}
	if time.Now().After(expiry) {
		s.mu.Lock()
		delete(s.tokens, token)
		s.mu.Unlock()
		return false
	}
	return true
}

func (s *AdminTokenStore) RevokeToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			common.WriteJSONError(w, http.StatusUnauthorized, "MISSING_ADMIN_TOKEN", "Missing admin token", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_AUTH_FORMAT", "Invalid authorization format. Use: Bearer <token>", nil)
			return
		}

		token := parts[1]
		if !adminTokenStore.ValidateToken(token) {
			common.LogError("AdminAuthMiddleware: invalid or expired token", "token_prefix", token[:min(len(token), 8)])
			common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_ADMIN_TOKEN", "Invalid or expired admin token", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func StoreAdminToken(token string, expiry time.Duration) {
	adminTokenStore.StoreToken(token, expiry)
}

func RevokeAdminToken(token string) {
	adminTokenStore.RevokeToken(token)
}
