package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"iseelocal/internal/shared/contracts"
)

func BearerMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/api/tls-ask" && r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		header := r.Header.Get("Authorization")
		if token == "" || header != "Bearer "+token {
			writeAuthError(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ExtractBearerToken(header string) string {
	value := strings.TrimSpace(header)
	if !strings.HasPrefix(value, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
}

func writeAuthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(contracts.ErrorResponse{Error: "unauthorized"})
}
