package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/kit/hash"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
)

var dummyHash = func() string {
	encoded, _ := hash.Hash("invalid-password-placeholder")
	return encoded
}()

type Session struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	User             UserResponse
}

type Service struct {
	repo      Repository
	jwt       *kitjwt.Manager
	blacklist Blacklist
}

func NewService(repo Repository, jwtManager *kitjwt.Manager, blacklist Blacklist) *Service {
	return &Service{repo: repo, jwt: jwtManager, blacklist: blacklist}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (UserResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.AppRole = strings.TrimSpace(req.AppRole)
	if err := req.Validate(); err != nil {
		return UserResponse{}, ErrInvalidInput
	}

	passwordHash, err := hash.Hash(req.Password)
	if err != nil {
		return UserResponse{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return UserResponse{}, err
	}

	user, err := s.repo.Create(ctx, User{
		ID:           id,
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: passwordHash,
		AppRole:      req.AppRole,
	})
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			return UserResponse{}, ErrEmailExists
		}
		return UserResponse{}, err
	}

	return toUserResponse(user), nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (Session, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := req.Validate(); err != nil {
		return Session{}, ErrInvalidInput
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			_, _ = hash.Verify(req.Password, dummyHash)
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	match, err := hash.Verify(req.Password, user.PasswordHash)
	if err != nil || !match {
		return Session{}, ErrInvalidCredentials
	}

	pair, err := s.jwt.GeneratePair(user.ID.String(), user.AppRole, user.TokenVersion)
	if err != nil {
		return Session{}, err
	}

	return newSession(pair, user), nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	if refreshToken == "" {
		return Session{}, ErrUnauthorized
	}

	claims, err := s.jwt.ParseRefresh(refreshToken)
	if err != nil {
		return Session{}, ErrUnauthorized
	}

	user, err := s.sessionUser(ctx, claims)
	if err != nil {
		return Session{}, err
	}

	pair, err := s.jwt.GeneratePair(user.ID.String(), user.AppRole, user.TokenVersion)
	if err != nil {
		return Session{}, err
	}

	claimed, err := s.claim(ctx, claims)
	if err != nil {
		return Session{}, err
	}
	if !claimed {
		return Session{}, ErrUnauthorized
	}

	return newSession(pair, user), nil
}

func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken == "" && refreshToken == "" {
		return ErrUnauthorized
	}

	var accessClaims *kitjwt.Claims
	if accessToken != "" {
		if claims, err := s.jwt.ParseAccess(accessToken); err == nil {
			accessClaims = claims
		}
	}

	var refreshClaims *kitjwt.Claims
	if refreshToken != "" {
		claims, err := s.jwt.ParseRefresh(refreshToken)
		if err != nil {
			return ErrUnauthorized
		}
		refreshClaims = claims
	}

	if accessClaims != nil && refreshClaims != nil && accessClaims.UserID != refreshClaims.UserID {
		accessClaims = nil
	}

	if accessClaims != nil {
		if err := s.revoke(ctx, accessClaims); err != nil {
			return err
		}
	}

	if refreshClaims != nil {
		if err := s.revoke(ctx, refreshClaims); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return UserResponse{}, ErrUnauthorized
		}
		return UserResponse{}, err
	}
	return toUserResponse(user), nil
}

func (s *Service) ValidateAccessToken(ctx context.Context, accessToken string) (*kitjwt.Claims, error) {
	claims, err := s.jwt.ParseAccess(accessToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if _, err := s.sessionUser(ctx, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (s *Service) sessionUser(ctx context.Context, claims *kitjwt.Claims) (User, error) {
	revoked, err := s.blacklist.Has(ctx, claims.ID)
	if err != nil {
		return User{}, err
	}
	if revoked {
		return User{}, ErrUnauthorized
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return User{}, ErrUnauthorized
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, ErrUnauthorized
		}
		return User{}, err
	}
	if user.TokenVersion != claims.TokenVersion {
		return User{}, ErrUnauthorized
	}

	return user, nil
}

func (s *Service) revoke(ctx context.Context, claims *kitjwt.Claims) error {
	if claims.ExpiresAt == nil {
		return nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	return s.blacklist.Add(ctx, claims.ID, ttl)
}

func (s *Service) claim(ctx context.Context, claims *kitjwt.Claims) (bool, error) {
	if claims.ExpiresAt == nil {
		return false, nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return false, nil
	}
	return s.blacklist.Claim(ctx, claims.ID, ttl)
}

func newSession(pair kitjwt.Pair, user User) Session {
	return Session{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
		User:             toUserResponse(user),
	}
}

func toUserResponse(user User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		AppRole:   user.AppRole,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
