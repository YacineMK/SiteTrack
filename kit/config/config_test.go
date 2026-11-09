package config

import (
	"testing"
	"time"
)

var allKeys = []string{
	"PORT",
	"COOKIE_SECURE",
	"JWT_SECRET",
	"JWT_ACCESS_TTL",
	"JWT_REFRESH_TTL",
	"RESEND_API_KEY",
	"RESEND_FROM",
	"RESEND_BASE_URL",
	"INVITE_TTL",
	"APP_BASE_URL",
	"DATABASE_URL",
	"DB_MAX_CONNS",
	"DB_MIN_CONNS",
	"DB_MAX_CONN_LIFETIME",
	"DB_MAX_CONN_IDLE_TIME",
	"REDIS_ADDR",
	"REDIS_PASSWORD",
	"REDIS_DB",
	"REDIS_POOL_SIZE",
	"REDIS_MIN_IDLE_CONNS",
	"REDIS_CONN_MAX_LIFETIME",
	"REDIS_CONN_MAX_IDLE_TIME",
	"WORKER_CONCURRENCY",
	"OUTBOX_BATCH_SIZE",
	"OUTBOX_POLL_INTERVAL",
	"OUTBOX_LOCK_TIMEOUT",
	"OUTBOX_MAX_ATTEMPTS",
}

func setEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range allKeys {
		t.Setenv(key, values[key])
	}
}

func TestLoad(t *testing.T) {
	setEnv(t, map[string]string{
		"PORT":                     ":9090",
		"COOKIE_SECURE":            "true",
		"JWT_SECRET":               "secret",
		"JWT_ACCESS_TTL":           "5m",
		"JWT_REFRESH_TTL":          "48h",
		"RESEND_API_KEY":           "re_test",
		"RESEND_FROM":              "noreply@example.com",
		"RESEND_BASE_URL":          "https://resend.test",
		"DATABASE_URL":             "postgres://user:pass@db:5432/app",
		"DB_MAX_CONNS":             "25",
		"DB_MIN_CONNS":             "5",
		"DB_MAX_CONN_LIFETIME":     "2h",
		"DB_MAX_CONN_IDLE_TIME":    "10m",
		"REDIS_ADDR":               "redis:6379",
		"REDIS_PASSWORD":           "redis-pass",
		"REDIS_DB":                 "3",
		"REDIS_POOL_SIZE":          "20",
		"REDIS_MIN_IDLE_CONNS":     "4",
		"REDIS_CONN_MAX_LIFETIME":  "7m",
		"REDIS_CONN_MAX_IDLE_TIME": "15m",
		"WORKER_CONCURRENCY":       "12",
		"OUTBOX_BATCH_SIZE":        "50",
		"OUTBOX_POLL_INTERVAL":     "3s",
		"OUTBOX_LOCK_TIMEOUT":      "2m",
		"OUTBOX_MAX_ATTEMPTS":      "8",
	})

	cfg := Load()

	if cfg.Port != ":9090" {
		t.Fatalf("Port = %q, want :9090", cfg.Port)
	}
	if !cfg.CookieSecure {
		t.Fatal("CookieSecure = false, want true")
	}
	if cfg.JWTSecret != "secret" {
		t.Fatalf("JWTSecret = %q, want secret", cfg.JWTSecret)
	}
	if cfg.JWTAccessTTL != 5*time.Minute {
		t.Fatalf("JWTAccessTTL = %v, want 5m", cfg.JWTAccessTTL)
	}
	if cfg.JWTRefreshTTL != 48*time.Hour {
		t.Fatalf("JWTRefreshTTL = %v, want 48h", cfg.JWTRefreshTTL)
	}
	if cfg.ResendAPIKey != "re_test" || cfg.ResendFrom != "noreply@example.com" || cfg.ResendBaseURL != "https://resend.test" {
		t.Fatalf("Resend config = %q/%q/%q", cfg.ResendAPIKey, cfg.ResendFrom, cfg.ResendBaseURL)
	}
	if cfg.DatabaseURL != "postgres://user:pass@db:5432/app" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.DBMaxConns != 25 || cfg.DBMinConns != 5 {
		t.Fatalf("DB conns = %d/%d, want 25/5", cfg.DBMaxConns, cfg.DBMinConns)
	}
	if cfg.DBMaxConnLifetime != 2*time.Hour || cfg.DBMaxConnIdleTime != 10*time.Minute {
		t.Fatalf("DB lifetimes = %v/%v", cfg.DBMaxConnLifetime, cfg.DBMaxConnIdleTime)
	}
	if cfg.RedisAddr != "redis:6379" || cfg.RedisPassword != "redis-pass" || cfg.RedisDB != 3 {
		t.Fatalf("Redis connection = %q/%q/%d", cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	}
	if cfg.RedisPoolSize != 20 || cfg.RedisMinIdleConns != 4 {
		t.Fatalf("Redis pool = %d/%d, want 20/4", cfg.RedisPoolSize, cfg.RedisMinIdleConns)
	}
	if cfg.RedisConnMaxLifetime != 7*time.Minute || cfg.RedisConnMaxIdleTime != 15*time.Minute {
		t.Fatalf("Redis lifetimes = %v/%v", cfg.RedisConnMaxLifetime, cfg.RedisConnMaxIdleTime)
	}
	if cfg.WorkerConcurrency != 12 || cfg.OutboxBatchSize != 50 || cfg.OutboxMaxAttempts != 8 {
		t.Fatalf("worker config = %d/%d/%d", cfg.WorkerConcurrency, cfg.OutboxBatchSize, cfg.OutboxMaxAttempts)
	}
	if cfg.OutboxPollInterval != 3*time.Second || cfg.OutboxLockTimeout != 2*time.Minute {
		t.Fatalf("outbox timings = %v/%v", cfg.OutboxPollInterval, cfg.OutboxLockTimeout)
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)

	cfg := Load()

	if cfg.Port != ":8080" {
		t.Fatalf("Port = %q, want :8080", cfg.Port)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure = true, want false")
	}
	if cfg.JWTSecret != "placeholder" {
		t.Fatalf("JWTSecret = %q, want placeholder", cfg.JWTSecret)
	}
	if cfg.JWTAccessTTL != 15*time.Minute {
		t.Fatalf("JWTAccessTTL = %v, want 15m", cfg.JWTAccessTTL)
	}
	if cfg.JWTRefreshTTL != 7*24*time.Hour {
		t.Fatalf("JWTRefreshTTL = %v, want 168h", cfg.JWTRefreshTTL)
	}
	if cfg.ResendAPIKey != "" || cfg.ResendFrom != "onboarding@resend.dev" || cfg.ResendBaseURL != "https://api.resend.com" {
		t.Fatalf("Resend defaults = %q/%q/%q", cfg.ResendAPIKey, cfg.ResendFrom, cfg.ResendBaseURL)
	}
	if cfg.DatabaseURL != "postgres://sitetrack:sitetrack@localhost:5432/sitetrack?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.DBMaxConns != 10 || cfg.DBMinConns != 2 {
		t.Fatalf("DB conns = %d/%d, want 10/2", cfg.DBMaxConns, cfg.DBMinConns)
	}
	if cfg.DBMaxConnLifetime != time.Hour || cfg.DBMaxConnIdleTime != 30*time.Minute {
		t.Fatalf("DB lifetimes = %v/%v", cfg.DBMaxConnLifetime, cfg.DBMaxConnIdleTime)
	}
	if cfg.RedisAddr != "127.0.0.1:6379" || cfg.RedisDB != 0 {
		t.Fatalf("Redis = %q/%d", cfg.RedisAddr, cfg.RedisDB)
	}
	if cfg.RedisPoolSize != 10 || cfg.RedisMinIdleConns != 2 {
		t.Fatalf("Redis pool = %d/%d, want 10/2", cfg.RedisPoolSize, cfg.RedisMinIdleConns)
	}
	if cfg.RedisConnMaxLifetime != 5*time.Minute || cfg.RedisConnMaxIdleTime != 30*time.Minute {
		t.Fatalf("Redis lifetimes = %v/%v", cfg.RedisConnMaxLifetime, cfg.RedisConnMaxIdleTime)
	}
	if cfg.WorkerConcurrency != 5 || cfg.OutboxBatchSize != 20 || cfg.OutboxMaxAttempts != 5 {
		t.Fatalf("worker defaults = %d/%d/%d", cfg.WorkerConcurrency, cfg.OutboxBatchSize, cfg.OutboxMaxAttempts)
	}
	if cfg.OutboxPollInterval != 2*time.Second || cfg.OutboxLockTimeout != time.Minute {
		t.Fatalf("outbox defaults = %v/%v", cfg.OutboxPollInterval, cfg.OutboxLockTimeout)
	}
}

func TestLoadInvalidValuesFallBackToDefaults(t *testing.T) {
	setEnv(t, map[string]string{
		"DB_MAX_CONNS":    "not-a-number",
		"REDIS_DB":        "not-a-number",
		"JWT_ACCESS_TTL":  "not-a-duration",
		"COOKIE_SECURE":   "not-a-bool",
		"REDIS_POOL_SIZE": "",
	})

	cfg := Load()

	if cfg.DBMaxConns != 10 {
		t.Fatalf("DBMaxConns = %d, want default 10", cfg.DBMaxConns)
	}
	if cfg.RedisDB != 0 {
		t.Fatalf("RedisDB = %d, want default 0", cfg.RedisDB)
	}
	if cfg.JWTAccessTTL != 15*time.Minute {
		t.Fatalf("JWTAccessTTL = %v, want default 15m", cfg.JWTAccessTTL)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure = true, want default false")
	}
	if cfg.RedisPoolSize != 10 {
		t.Fatalf("RedisPoolSize = %d, want default 10", cfg.RedisPoolSize)
	}
}
