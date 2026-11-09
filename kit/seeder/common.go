package main

import (
	"os"
	"strconv"
)

type seedConfig struct {
	users              int
	projects           int
	maxMembers         int
	invitations        int
	tasks              int
	maxTasksPerProject int
	dependencies       int
	materials          int
	deliveries         int
	lines              int
	reset              bool
}

func loadConfig() seedConfig {
	return seedConfig{
		users:              envInt("SEED_USERS", 5000),
		projects:           envInt("SEED_PROJECTS", 2000),
		maxMembers:         envInt("SEED_MAX_MEMBERS", 50),
		invitations:        envInt("SEED_INVITATIONS", 50000),
		tasks:              envInt("SEED_TASKS", 500000),
		maxTasksPerProject: envInt("SEED_MAX_TASKS_PER_PROJECT", 3000),
		dependencies:       envInt("SEED_DEPENDENCIES", 1000000),
		materials:          envInt("SEED_MATERIALS", 20000),
		deliveries:         envInt("SEED_DELIVERIES", 400000),
		lines:              envInt("SEED_LINES", 3000000),
		reset:              envBool("SEED_RESET", false),
	}
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}
