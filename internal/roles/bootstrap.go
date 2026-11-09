package roles

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

type defaultRole struct {
	name        string
	description string
}

var defaultRoles = []defaultRole{
	{name: "user", description: "Can only access projects they are a member of"},
	{name: "company_store_manager", description: "User permissions plus full access to company store"},
	{name: "super_admin", description: "Full access to all projects and company store"},
}

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

	for _, role := range defaultRoles {
		if err := queries.UpsertRole(ctx, sqlc.UpsertRoleParams{
			Name:        role.name,
			Description: role.description,
		}); err != nil {
			return err
		}
	}

	for _, policy := range defaultPolicies {
		if err := queries.UpsertRolePolicy(ctx, sqlc.UpsertRolePolicyParams{
			Name:     policy.role,
			Resource: policy.resource,
			Action:   policy.action,
		}); err != nil {
			return err
		}
	}

	return nil
}
