package roles

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

type defaultPolicy struct {
	role     string
	resource string
	action   string
}

var defaultPolicies = []defaultPolicy{
	{"user", "materials", "write"},
	{"user", "projects", "write"},
	{"user", "tasks", "write"},
	{"user", "project_store", "write"},
	{"user", "project_deliveries", "write"},

	{"company_store_manager", "materials", "write"},
	{"company_store_manager", "projects", "write"},
	{"company_store_manager", "tasks", "write"},
	{"company_store_manager", "project_store", "write"},
	{"company_store_manager", "project_deliveries", "write"},
	{"company_store_manager", "company_store", "write"},
	{"company_store_manager", "company_store_deliveries", "write"},

	{"super_admin", "materials", "write"},
	{"super_admin", "projects", "write"},
	{"super_admin", "tasks", "write"},
	{"super_admin", "project_store", "write"},
	{"super_admin", "project_deliveries", "write"},
	{"super_admin", "company_store", "write"},
	{"super_admin", "company_store_deliveries", "write"},
	{"super_admin", "roles", "write"},
	{"super_admin", "users", "write"},
}

func Bootstrap(ctx context.Context, db *pgxpool.Pool) error {
	queries := sqlc.New(db)

	for _, policy := range defaultPolicies {
		if err := queries.UpsertAppRolePolicyByName(ctx, sqlc.UpsertAppRolePolicyByNameParams{
			Name:     policy.role,
			Resource: policy.resource,
			Action:   policy.action,
		}); err != nil {
			return err
		}
	}

	return nil
}
