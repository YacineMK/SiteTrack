package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/kit/hash"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Session struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	User             UserResponse
}

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (UserResponse, error)
	Login(ctx context.Context, req LoginRequest) (Session, error)
	Refresh(ctx context.Context, refreshToken string) (Session, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	Me(ctx context.Context, userID uuid.UUID) (UserResponse, error)
	EmailAvailable(ctx context.Context, email string) (bool, error)
	ValidateAccessToken(ctx context.Context, accessToken string) (*kitjwt.Claims, error)
}

type service struct {
	repo      Repository
	jwt       *kitjwt.Manager
	blacklist Blacklist
	utils     Utils
}

func NewService(repo Repository, jwtManager *kitjwt.Manager, blacklist Blacklist) Service {
	return &service{
		repo:      repo,
		jwt:       jwtManager,
		blacklist: blacklist,
		utils:     NewUtils(),
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (UserResponse, error) {
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

	return s.utils.ToUserResponse(user), nil
}

func (s *service) Login(ctx context.Context, req LoginRequest) (Session, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := req.Validate(); err != nil {
		return Session{}, ErrInvalidInput
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			_, _ = hash.Verify(req.Password, s.utils.DummyHash())
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

	return s.utils.NewSession(pair, user), nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
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

	claimed, err := s.utils.Claim(ctx, s.blacklist, claims)
	if err != nil {
		return Session{}, err
	}
	if !claimed {
		return Session{}, ErrUnauthorized
	}

	return s.utils.NewSession(pair, user), nil
}

func (s *service) Logout(ctx context.Context, accessToken, refreshToken string) error {
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
		if err := s.utils.Revoke(ctx, s.blacklist, accessClaims); err != nil {
			return err
		}
	}

	if refreshClaims != nil {
		if err := s.utils.Revoke(ctx, s.blacklist, refreshClaims); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) Me(ctx context.Context, userID uuid.UUID) (UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return UserResponse{}, ErrUnauthorized
		}
		return UserResponse{}, err
	}
	return s.utils.ToUserResponse(user), nil
}

func (s *service) EmailAvailable(ctx context.Context, email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := validation.Validate(email, validation.Required, is.Email); err != nil {
		return false, ErrInvalidInput
	}

	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (s *service) ValidateAccessToken(ctx context.Context, accessToken string) (*kitjwt.Claims, error) {
	claims, err := s.jwt.ParseAccess(accessToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	user, err := s.sessionUser(ctx, claims)
	if err != nil {
		return nil, err
	}
	claims.AppRole = user.AppRole
	return claims, nil
}

func (s *service) sessionUser(ctx context.Context, claims *kitjwt.Claims) (User, error) {
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
