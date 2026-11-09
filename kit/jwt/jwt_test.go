package jwt

import (
	"errors"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParse(t *testing.T) {
	manager := New("test-secret", time.Hour)

	token, expiresAt, err := manager.Generate("user-1", "admin", 3)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if token == "" {
		t.Fatal("Generate() returned an empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("Generate() returned an expiration in the past")
	}

	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.AppRole != "admin" {
		t.Fatalf("AppRole = %q, want %q", claims.AppRole, "admin")
	}
	if claims.TokenVersion != 3 {
		t.Fatalf("TokenVersion = %d, want %d", claims.TokenVersion, 3)
	}
	if claims.ID == "" {
		t.Fatal("token is missing a jti")
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(expiresAt.UTC().Truncate(time.Second)) {
		t.Fatal("parsed expiration does not match the generated expiration")
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	issuer := New("secret-one", time.Hour)
	verifier := New("secret-two", time.Hour)

	token, _, err := issuer.Generate("user-1", "user", 1)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, err := verifier.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	manager := New("test-secret", -time.Minute)

	token, _, err := manager.Generate("user-1", "user", 1)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, err := manager.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsTokenWithoutExpiration(t *testing.T) {
	manager := New("test-secret", time.Hour)
	claims := Claims{
		UserID:       "user-1",
		AppRole:      "user",
		TokenVersion: 1,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ID: "some-jti",
		},
	}

	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	if _, err := manager.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
	}
}
