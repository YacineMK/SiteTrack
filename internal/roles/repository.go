package roles

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
	"github.com/YacineMK/SiteTrack/kit/rbac"
)

var (
	ErrRoleExists   = errors.New("role already exists")
	ErrRoleNotFound = errors.New("role not found")
	ErrPolicyExists = errors.New("policy already exists")
	ErrUserNotFound = errors.New("user not found")
)

type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
}

type UserRole struct {
	ID        uuid.UUID
	Name      string
	Email     string
	AppRole   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Policy struct {
	RoleID   uuid.UUID
	Resource string
	Action   string
}

type Repository interface {
	CreateRole(ctx context.Context, name, description string) (Role, error)
	ListRoles(ctx context.Context) ([]Role, error)
	GetRole(ctx context.Context, id uuid.UUID) (Role, error)
	GetRoleByName(ctx context.Context, name string) (Role, error)
	AddPolicy(ctx context.Context, roleID uuid.UUID, resource, action string) error
	ListPolicies(ctx context.Context, roleID uuid.UUID) ([]Policy, error)
	DeletePolicy(ctx context.Context, roleID uuid.UUID, resource, action string) (int64, error)
	AssignRole(ctx context.Context, userID uuid.UUID, roleName string) (UserRole, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
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

func (r *PostgresRepository) GetRoleByName(ctx context.Context, name string) (Role, error) {
	row, err := r.queries.GetRoleByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, ErrRoleNotFound
		}
		return Role{}, err
	}
	return toRole(row), nil
}

func (r *PostgresRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleName string) (UserRole, error) {
	if _, err := r.queries.GetRoleByName(ctx, roleName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserRole{}, ErrRoleNotFound
		}
		return UserRole{}, err
	}

	row, err := r.queries.AssignUserRole(ctx, sqlc.AssignUserRoleParams{
		UserID: userID,
		Role:   roleName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserRole{}, ErrUserNotFound
		}
		return UserRole{}, err
	}

	return UserRole{
		ID:        row.ID,
		Name:      row.Name,
		Email:     row.Email,
		AppRole:   row.AppRole,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *PostgresRepository) toRoleFromRow(row sqlc.AppRole) Role {
	return Role{ID: row.ID, Name: row.Name, Description: row.Description}
}

func (r *PostgresRepository) HasPolicy(ctx context.Context, role, resource, action string) (bool, error) {
	policies, err := r.queries.GetPoliciesByRoleName(ctx, role)
	if err != nil {
		return false, err
	}

	result := make([]rbac.Policy, 0, len(policies))
	for _, policy := range policies {
		result = append(result, rbac.Policy{Resource: policy.Resource, Action: policy.Action})
	}
	return rbac.Allows(result, resource, action), nil
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
