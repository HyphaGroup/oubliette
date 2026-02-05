package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/HyphaGroup/oubliette/internal/logger"
)

// Middleware creates HTTP middleware for authentication
// Only Bearer token authentication is supported for external HTTP access.
// Internal gogol connections use unix sockets (no HTTP auth needed).
func Middleware(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")

			if !strings.HasPrefix(auth, "Bearer ") {
				jsonError(w, "Authentication required (Bearer token)", http.StatusUnauthorized)
				return
			}

			tokenID := strings.TrimPrefix(auth, "Bearer ")
			token, err := store.ValidateToken(tokenID)
			if err != nil {
				logger.Info("Token validation failed: %v", err)
				jsonError(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			authContext := &AuthContext{
				Type:  AuthTypeToken,
				Token: token,
			}
			logger.Info("Authenticated with token: %s (scope: %s)", maskToken(tokenID), token.Scope)

			ctx := WithContext(r.Context(), authContext)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32001,
			"message": message,
		},
		"id": nil,
	})
}

func maskToken(tokenID string) string {
	if len(tokenID) <= 12 {
		return "***"
	}
	return tokenID[:8] + "..." + tokenID[len(tokenID)-4:]
}
