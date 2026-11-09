package jwt

import (
	"errors"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestGeneratePairAndParse(t *testing.T) {
	manager := New("test-secret", 15*time.Minute, 24*time.Hour)

	pair, err := manager.GeneratePair("user-1", "admin", 3)
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("GeneratePair() returned an empty token")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Fatal("access and refresh tokens must differ")
	}

	access, err := manager.ParseAccess(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if access.UserID != "user-1" || access.AppRole != "admin" || access.TokenVersion != 3 {
		t.Fatalf("unexpected access claims: %+v", access)
	}
	if access.TokenType != AccessToken {
		t.Fatalf("access token type = %q, want %q", access.TokenType, AccessToken)
	}
	if access.ID == "" {
		t.Fatal("access token is missing a jti")
	}
	if access.ExpiresAt == nil || !access.ExpiresAt.Time.Equal(pair.AccessExpiresAt.UTC().Truncate(time.Second)) {
		t.Fatal("access expiration does not match the generated value")
	}

	refresh, err := manager.ParseRefresh(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}
	if refresh.UserID != "user-1" || refresh.AppRole != "admin" || refresh.TokenVersion != 3 {
		t.Fatalf("unexpected refresh claims: %+v", refresh)
	}
	if refresh.TokenType != RefreshToken {
		t.Fatalf("refresh token type = %q, want %q", refresh.TokenType, RefreshToken)
	}
	if refresh.ID == "" || refresh.ID == access.ID {
		t.Fatal("refresh token must have its own jti")
	}
	if refresh.ExpiresAt == nil || !refresh.ExpiresAt.Time.Equal(pair.RefreshExpiresAt.UTC().Truncate(time.Second)) {
		t.Fatal("refresh expiration does not match the generated value")
	}
	if !pair.RefreshExpiresAt.After(pair.AccessExpiresAt) {
		t.Fatal("refresh token must outlive the access token")
	}
}

func TestParseRejectsWrongTokenType(t *testing.T) {
	manager := New("test-secret", 15*time.Minute, 24*time.Hour)

	pair, err := manager.GeneratePair("user-1", "user", 1)
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}

	if _, err := manager.ParseRefresh(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseRefresh(access) error = %v, want ErrInvalidToken", err)
	}
	if _, err := manager.ParseAccess(pair.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseAccess(refresh) error = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	issuer := New("secret-one", 15*time.Minute, 24*time.Hour)
	verifier := New("secret-two", 15*time.Minute, 24*time.Hour)

	pair, err := issuer.GeneratePair("user-1", "user", 1)
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}

	if _, err := verifier.ParseAccess(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseAccess() error = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsExpiredTokens(t *testing.T) {
	expiredAccess := New("test-secret", -time.Minute, 24*time.Hour)
	pair, err := expiredAccess.GeneratePair("user-1", "user", 1)
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}
	if _, err := expiredAccess.ParseAccess(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseAccess() error = %v, want ErrInvalidToken", err)
	}

	expiredRefresh := New("test-secret", 15*time.Minute, -time.Minute)
	pair, err = expiredRefresh.GeneratePair("user-1", "user", 1)
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}
	if _, err := expiredRefresh.ParseRefresh(pair.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseRefresh() error = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsTokenWithoutExpiration(t *testing.T) {
	manager := New("test-secret", 15*time.Minute, 24*time.Hour)
	claims := Claims{
		UserID:       "user-1",
		AppRole:      "user",
		TokenVersion: 1,
		TokenType:    AccessToken,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ID: "some-jti",
		},
	}

	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	if _, err := manager.ParseAccess(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ParseAccess() error = %v, want ErrInvalidToken", err)
	}
}
