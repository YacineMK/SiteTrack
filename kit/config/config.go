package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:          getEnv("PORT", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://sitetrack:sitetrack@localhost:5432/sitetrack?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", "sitetrack"),
		RedisDB:       getEnvInt("REDIS_DB", 0),
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
