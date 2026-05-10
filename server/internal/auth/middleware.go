package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

func RequireJWT(tokens *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "missing or invalid bearer token")
				return
			}

			claims, err := tokens.Validate(token)
			if err != nil {
				message := "invalid bearer token"
				if errors.Is(err, ErrExpiredToken) {
					message = "expired bearer token"
				}
				writeAuthError(w, http.StatusUnauthorized, message)
				return
			}

			next.ServeHTTP(w, r.WithContext(contextWithClaims(r.Context(), claims)))
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
