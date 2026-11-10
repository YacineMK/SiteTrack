package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID    string    `json:"user_id"`
	AppRole   string    `json:"app_role"`
	TokenType TokenType `json:"token_type"`
	jwtlib.RegisteredClaims
}

type Pair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func New(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (m *Manager) GeneratePair(userID, appRole string) (Pair, error) {
	accessToken, accessExpiresAt, err := m.generate(userID, appRole, AccessToken, m.accessTTL)
	if err != nil {
		return Pair{}, err
	}
	refreshToken, refreshExpiresAt, err := m.generate(userID, appRole, RefreshToken, m.refreshTTL)
	if err != nil {
		return Pair{}, err
	}
	return Pair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (m *Manager) ParseAccess(tokenString string) (*Claims, error) {
	return m.parse(tokenString, AccessToken)
}

func (m *Manager) ParseRefresh(tokenString string) (*Claims, error) {
	return m.parse(tokenString, RefreshToken)
}

func (m *Manager) generate(userID, appRole string, tokenType TokenType, ttl time.Duration) (string, time.Time, error) {
	jti, err := uuid.NewV7()
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now()
	expiresAt := now.Add(ttl)
	claims := Claims{
		UserID:    userID,
		AppRole:   appRole,
		TokenType: tokenType,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ID:        jti.String(),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
	}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (m *Manager) parse(tokenString string, expected TokenType) (*Claims, error) {
	claims := &Claims{}
	token, err := jwtlib.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwtlib.Token) (any, error) {
			return m.secret, nil
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}),
		jwtlib.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != expected {
		return nil, fmt.Errorf("%w: unexpected token type %q", ErrInvalidToken, claims.TokenType)
	}
	return claims, nil
}
