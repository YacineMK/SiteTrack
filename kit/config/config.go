package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
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
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:                 getEnv("PORT", ":8080"),
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
