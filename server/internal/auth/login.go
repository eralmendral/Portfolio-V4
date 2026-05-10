package auth

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int64  `json:"expires_in"`
}

func LoginHandler(tokens *TokenService, username string, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if username == "" || password == "" {
			writeAuthError(w, http.StatusServiceUnavailable, "admin credentials are not configured")
			return
		}

		var input loginRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid login request")
			return
		}

		if subtle.ConstantTimeCompare([]byte(input.Username), []byte(username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(input.Password), []byte(password)) != 1 {
			writeAuthError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}

		token, err := tokens.Sign(input.Username, []string{"admin"})
		if err != nil {
			writeAuthError(w, http.StatusInternalServerError, "could not issue token")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(loginResponse{
			Token:     token,
			TokenType: "Bearer",
			ExpiresIn: int64(tokens.TTL().Seconds()),
		})
	})
}
