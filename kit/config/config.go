package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	CookieSecure         bool
	JWTSecret            string
	JWTAccessTTL         time.Duration
	JWTRefreshTTL        time.Duration
	ResendAPIKey         string
	ResendFrom           string
	ResendBaseURL        string
	InviteTTL            time.Duration
	AppBaseURL           string
	DatabaseURL          string
	DBMaxConns           int32
	DBMinConns           int32
	DBMaxConnLifetime    time.Duration
	DBMaxConnIdleTime    time.Duration
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	RedisPoolSize        int
	RedisMinIdleConns    int
	RedisConnMaxLifetime time.Duration
	RedisConnMaxIdleTime time.Duration
	WorkerConcurrency    int
	OutboxBatchSize      int
	OutboxPollInterval   time.Duration
	OutboxLockTimeout    time.Duration
	OutboxMaxAttempts    int
	CORSAllowedOrigins   []string
	RateLimitGlobal      int
	RateLimitAuth        int
	RateLimitWindow      time.Duration
}


func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:                 getEnv("PORT", ":8080"),
		CookieSecure:         getEnvBool("COOKIE_SECURE", false),
		JWTSecret:            getEnv("JWT_SECRET", "placeholder"),
		JWTAccessTTL:         getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:        getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		ResendAPIKey:         getEnv("RESEND_API_KEY", ""),
		ResendFrom:           getEnv("RESEND_FROM", "onboarding@resend.dev"),
		ResendBaseURL:        getEnv("RESEND_BASE_URL", "https://api.resend.com"),
		InviteTTL:            min(getEnvDuration("INVITE_TTL", time.Hour), time.Hour),
		AppBaseURL:           getEnv("APP_BASE_URL", "http://localhost:8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://sitetrack:sitetrack@localhost:5432/sitetrack?sslmode=disable"),
		DBMaxConns:           int32(getEnvInt("DB_MAX_CONNS", 10)),
		DBMinConns:           int32(getEnvInt("DB_MIN_CONNS", 2)),
		DBMaxConnLifetime:    getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		DBMaxConnIdleTime:    getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		RedisAddr:            getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", "sitetrack"),
		RedisDB:              getEnvInt("REDIS_DB", 0),
		RedisPoolSize:        getEnvInt("REDIS_POOL_SIZE", 10),
		RedisMinIdleConns:    getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
		RedisConnMaxLifetime: getEnvDuration("REDIS_CONN_MAX_LIFETIME", 5*time.Minute),
		RedisConnMaxIdleTime: getEnvDuration("REDIS_CONN_MAX_IDLE_TIME", 30*time.Minute),
		WorkerConcurrency:    getEnvInt("WORKER_CONCURRENCY", 5),
		OutboxBatchSize:      getEnvInt("OUTBOX_BATCH_SIZE", 20),
		OutboxPollInterval:   getEnvDuration("OUTBOX_POLL_INTERVAL", 2*time.Second),
		OutboxLockTimeout:    getEnvDuration("OUTBOX_LOCK_TIMEOUT", time.Minute),
		OutboxMaxAttempts:    getEnvInt("OUTBOX_MAX_ATTEMPTS", 5),
		CORSAllowedOrigins:   strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		RateLimitGlobal:      getEnvInt("RATE_LIMIT_GLOBAL", 60),
		RateLimitAuth:        getEnvInt("RATE_LIMIT_AUTH", 5),
		RateLimitWindow:      getEnvDuration("RATE_LIMIT_WINDOW", 10*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
