package auth

import (
	"errors"
	"testing"
	"time"
)

func TestTokenServiceSignsAndValidatesHS256JWT(t *testing.T) {
	tokens := NewTokenService("secret", "tests", time.Hour)
	tokens.now = func() time.Time {
		return time.Unix(1000, 0)
	}

	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	claims, err := tokens.Validate(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if claims.Subject != "admin" {
		t.Fatalf("subject = %q, want admin", claims.Subject)
	}
	if claims.ExpiresAt != time.Unix(1000, 0).Add(time.Hour).Unix() {
		t.Fatalf("unexpected expiry: %d", claims.ExpiresAt)
	}
}

func TestTokenServiceRejectsExpiredTokens(t *testing.T) {
	now := time.Unix(1000, 0)
	tokens := NewTokenService("secret", "tests", time.Second)
	tokens.now = func() time.Time {
		return now
	}

	token, err := tokens.Sign("admin", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	tokens.now = func() time.Time {
		return now.Add(2 * time.Second)
	}

	_, err = tokens.Validate(token)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("err = %v, want %v", err, ErrExpiredToken)
	}
}

func TestTokenServiceRejectsWrongSecret(t *testing.T) {
	tokens := NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	other := NewTokenService("other-secret", "tests", time.Hour)
	_, err = other.Validate(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidToken)
	}
}
