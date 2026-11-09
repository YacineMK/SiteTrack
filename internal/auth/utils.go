package auth

import (
	"context"
	"net/http"
	"time"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

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

func writeUnauthorized(w http.ResponseWriter) {
	jsonutil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
