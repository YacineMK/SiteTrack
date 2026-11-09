package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/YacineMK/SiteTrack/kit/hash"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type Utils interface {
	DummyHash() string
	NewSession(pair kitjwt.Pair, user User) Session
	ToUserResponse(user User) UserResponse
	AuthResponse(session Session) AuthResponse
	Revoke(ctx context.Context, blacklist Blacklist, claims *kitjwt.Claims) error
	Claim(ctx context.Context, blacklist Blacklist, claims *kitjwt.Claims) (bool, error)
	WriteUnauthorized(w http.ResponseWriter)
	SetSessionCookies(w http.ResponseWriter, session Session, secure bool)
	ClearSessionCookies(w http.ResponseWriter, secure bool)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

var dummyHash = func() string {
	encoded, _ := hash.Hash("invalid-password-placeholder")
	return encoded
}()

func (utils) DummyHash() string {
	return dummyHash
}

func (u utils) NewSession(pair kitjwt.Pair, user User) Session {
	return Session{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
		User:             u.ToUserResponse(user),
	}
}

func (utils) ToUserResponse(user User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		AppRole:   user.AppRole,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (utils) AuthResponse(session Session) AuthResponse {
	return AuthResponse{
		ExpiresIn: int64(time.Until(session.AccessExpiresAt).Seconds()),
		User:      session.User,
	}
}

func (utils) Revoke(ctx context.Context, blacklist Blacklist, claims *kitjwt.Claims) error {
	if claims.ExpiresAt == nil {
		return nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	return blacklist.Add(ctx, claims.ID, ttl)
}

func (utils) Claim(ctx context.Context, blacklist Blacklist, claims *kitjwt.Claims) (bool, error) {
	if claims.ExpiresAt == nil {
		return false, nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return false, nil
	}
	return blacklist.Claim(ctx, claims.ID, ttl)
}

func (utils) WriteUnauthorized(w http.ResponseWriter) {
	jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
}

func (utils) SetSessionCookies(w http.ResponseWriter, session Session, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    session.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.AccessExpiresAt,
		MaxAge:   int(time.Until(session.AccessExpiresAt).Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    session.RefreshToken,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.RefreshExpiresAt,
		MaxAge:   int(time.Until(session.RefreshExpiresAt).Seconds()),
	})
}

func (utils) ClearSessionCookies(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}

type Blacklist interface {
	Add(ctx context.Context, jti string, ttl time.Duration) error
	Has(ctx context.Context, jti string) (bool, error)
	Claim(ctx context.Context, jti string, ttl time.Duration) (bool, error)
}

type RedisBlacklist struct {
	redis *kitredis.Redis
}

func NewRedisBlacklist(redis *kitredis.Redis) *RedisBlacklist {
	return &RedisBlacklist{redis: redis}
}

func (b *RedisBlacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	return b.redis.Client.Set(ctx, blacklistKey(jti), "1", ttl).Err()
}

func (b *RedisBlacklist) Has(ctx context.Context, jti string) (bool, error) {
	count, err := b.redis.Client.Exists(ctx, blacklistKey(jti)).Result()
	return count > 0, err
}

func (b *RedisBlacklist) Claim(ctx context.Context, jti string, ttl time.Duration) (bool, error) {
	return b.redis.Client.SetNX(ctx, blacklistKey(jti), "1", ttl).Result()
}

func blacklistKey(jti string) string {
	return "auth:blacklist:" + jti
}
