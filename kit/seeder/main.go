package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/kit/config"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

func main() {
	log.SetFlags(0)
	cfg := loadConfig()
	appCfg := config.Load()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, appCfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	var existing int64
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&existing); err != nil {
		log.Fatal(err)
	}
	if existing > 0 && !cfg.reset {
		log.Printf("database already seeded (%d users); set SEED_RESET=true to truncate first", existing)
		os.Exit(0)
	}
	if cfg.reset {
		for _, table := range []string{"delivery_lines", "deliveries", "task_dependencies", "tasks", "notifications", "project_members", "projects", "materials", "users"} {
			if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE"); err != nil {
				log.Fatal(err)
			}
		}
	}

	if cfg.tasks > cfg.projects*cfg.maxTasksPerProject {
		cfg.tasks = cfg.projects * cfg.maxTasksPerProject
	}

	steps := []struct {
		name string
		run  func() error
	}{
		{"users", func() error { return seedUsers(ctx, pool, cfg.users) }},
		{"materials", func() error { return seedMaterials(ctx, pool, cfg.materials) }},
		{"projects", func() error { return seedProjects(ctx, pool, cfg.projects) }},
		{"project_members", func() error { return seedProjectMembers(ctx, pool, cfg.maxMembers) }},
		{"invitations", func() error { return seedInvitations(ctx, pool, cfg.invitations) }},
		{"tasks", func() error { return seedTasks(ctx, pool, cfg.tasks) }},
		{"task_dependencies", func() error { return seedTaskDependencies(ctx, pool, cfg.dependencies) }},
		{"deliveries", func() error { return seedDeliveries(ctx, pool, cfg.deliveries) }},
		{"delivery_lines", func() error { return seedDeliveryLines(ctx, pool, cfg.lines, cfg.deliveries) }},
	}

	for _, step := range steps {
		started := time.Now()
		if err := step.run(); err != nil {
			log.Fatalf("%s: %v", step.name, err)
		}
		log.Printf("%s done in %s", step.name, time.Since(started).Round(time.Millisecond))
	}

	redisClient, err := kitredis.New(ctx, kitredis.Config{
		Addr:     appCfg.RedisAddr,
		Password: appCfg.RedisPassword,
		DB:       appCfg.RedisDB,
	})
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	if err := syncSequences(ctx, pool, redisClient); err != nil {
		log.Fatalf("sync sequences: %v", err)
	}

	log.Printf("seed complete: %d users, %d projects, %d tasks, %d dependencies, %d materials, %d deliveries",
		cfg.users, cfg.projects, cfg.tasks, cfg.dependencies, cfg.materials, cfg.deliveries)
}
