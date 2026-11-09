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

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := req.Validate(); err != nil {
		return AuthResponse{}, ErrInvalidInput
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			_, _ = hash.Verify(req.Password, dummyHash)
			return AuthResponse{}, ErrInvalidCredentials
		}
		return AuthResponse{}, err
	}

	match, err := hash.Verify(req.Password, user.PasswordHash)
	if err != nil || !match {
		return AuthResponse{}, ErrInvalidCredentials
	}

	pair, err := s.jwt.GeneratePair(user.ID.String(), user.AppRole, user.TokenVersion)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(time.Until(pair.AccessExpiresAt).Seconds()),
		User:         toUserResponse(user),
	}, nil
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

func (s *Service) Authenticate(ctx context.Context, accessToken string) (*kitjwt.Claims, error) {
	claims, err := s.jwt.ParseAccess(accessToken)
	if err != nil {
		return nil, ErrUnauthorized
	}

	revoked, err := s.blacklist.Has(ctx, claims.ID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, ErrUnauthorized
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, ErrUnauthorized
	}

	return claims, nil
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
