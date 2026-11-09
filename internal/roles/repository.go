package roles

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var (
	ErrRoleExists   = errors.New("role already exists")
	ErrRoleNotFound = errors.New("role not found")
	ErrPolicyExists = errors.New("policy already exists")
)

type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
}

type Policy struct {
	RoleID   uuid.UUID
	Resource string
	Action   string
}

type Repository interface {
	HasPolicy(ctx context.Context, role, resource, action string) (bool, error)
	CreateRole(ctx context.Context, name, description string) (Role, error)
	ListRoles(ctx context.Context) ([]Role, error)
	GetRole(ctx context.Context, id uuid.UUID) (Role, error)
	AddPolicy(ctx context.Context, roleID uuid.UUID, resource, action string) error
	ListPolicies(ctx context.Context, roleID uuid.UUID) ([]Policy, error)
	DeletePolicy(ctx context.Context, roleID uuid.UUID, resource, action string) (int64, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) HasPolicy(ctx context.Context, role, resource, action string) (bool, error) {
	return r.queries.CheckRolePolicy(ctx, sqlc.CheckRolePolicyParams{
		RoleName: role,
		Resource: resource,
		Action:   action,
	})
}

func (r *PostgresRepository) CreateRole(ctx context.Context, name, description string) (Role, error) {
	row, err := r.queries.CreateRole(ctx, sqlc.CreateRoleParams{
		Name:        name,
		Description: description,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Role{}, ErrRoleExists
		}
		return Role{}, err
	}
	return toRole(row), nil
}

func (r *PostgresRepository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.queries.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, toRole(row))
	}
	return roles, nil
}

func (r *PostgresRepository) GetRole(ctx context.Context, id uuid.UUID) (Role, error) {
	row, err := r.queries.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, ErrRoleNotFound
		}
		return Role{}, err
	}
	return toRole(row), nil
}

func (r *PostgresRepository) AddPolicy(ctx context.Context, roleID uuid.UUID, resource, action string) error {
	err := r.queries.AddRolePolicy(ctx, sqlc.AddRolePolicyParams{
		RoleID:   roleID,
		Resource: resource,
		Action:   action,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPolicyExists
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) ListPolicies(ctx context.Context, roleID uuid.UUID) ([]Policy, error) {
	rows, err := r.queries.ListRolePolicies(ctx, roleID)
	if err != nil {
		return nil, err
	}
	policies := make([]Policy, 0, len(rows))
	for _, row := range rows {
		policies = append(policies, Policy{
			RoleID:   row.RoleID,
			Resource: row.Resource,
			Action:   row.Action,
		})
	}
	return policies, nil
}

func (r *PostgresRepository) DeletePolicy(ctx context.Context, roleID uuid.UUID, resource, action string) (int64, error) {
	return r.queries.DeleteRolePolicy(ctx, sqlc.DeleteRolePolicyParams{
		RoleID:   roleID,
		Resource: resource,
		Action:   action,
	})
}

func toRole(row sqlc.AppRole) Role {
	return Role{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
