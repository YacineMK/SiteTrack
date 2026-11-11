package roles

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
	"github.com/YacineMK/SiteTrack/kit/rbac"
)

var (
	ErrRoleExists    = errors.New("role already exists")
	ErrRoleNotFound  = errors.New("role not found")
	ErrPolicyExists  = errors.New("policy already exists")
	ErrUserNotFound  = errors.New("user not found")
	ErrSystemRole    = errors.New("cannot modify or delete a system role")
)

type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsAdmin     bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

// nil projectID = app-scope row, non-nil = project-scope row (same table).
type Repository interface {
	ListRoles(ctx context.Context, projectID *uuid.UUID) ([]Role, error)
	GetRole(ctx context.Context, projectID *uuid.UUID, id uuid.UUID) (Role, error)
	GetRoleByName(ctx context.Context, projectID *uuid.UUID, name string) (Role, error)
	CreateRole(ctx context.Context, projectID *uuid.UUID, name, description string, isAdmin bool) (Role, error)
	UpdateRole(ctx context.Context, projectID *uuid.UUID, id uuid.UUID, name, description string, isAdmin bool) (Role, error)
	DeleteRole(ctx context.Context, projectID *uuid.UUID, id uuid.UUID) error

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

func pgScope(projectID *uuid.UUID) pgtype.UUID {
	if projectID == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *projectID, Valid: true}
}

func (r *PostgresRepository) ListRoles(ctx context.Context, projectID *uuid.UUID) ([]Role, error) {
	rows, err := r.queries.ListRoles(ctx, pgScope(projectID))
	if err != nil {
		return nil, err
	}
	roles := make([]Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, Role{
			ID: row.ID, Name: row.Name, Description: row.Description,
			IsAdmin: row.IsAdmin, IsSystem: row.IsSystem,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return roles, nil
}

func (r *PostgresRepository) GetRole(ctx context.Context, projectID *uuid.UUID, id uuid.UUID) (Role, error) {
	row, err := r.queries.GetRoleByID(ctx, sqlc.GetRoleByIDParams{ID: id, ProjectID: pgScope(projectID)})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, ErrRoleNotFound
		}
		return Role{}, err
	}
	return Role{
		ID: row.ID, Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin, IsSystem: row.IsSystem,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *PostgresRepository) GetRoleByName(ctx context.Context, projectID *uuid.UUID, name string) (Role, error) {
	row, err := r.queries.GetRoleByName(ctx, sqlc.GetRoleByNameParams{Name: name, ProjectID: pgScope(projectID)})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, ErrRoleNotFound
		}
		return Role{}, err
	}
	return Role{
		ID: row.ID, Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin, IsSystem: row.IsSystem,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *PostgresRepository) CreateRole(ctx context.Context, projectID *uuid.UUID, name, description string, isAdmin bool) (Role, error) {
	row, err := r.queries.CreateRole(ctx, sqlc.CreateRoleParams{
		ProjectID: pgScope(projectID), Name: name, Description: description, IsAdmin: isAdmin,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Role{}, ErrRoleExists
		}
		return Role{}, err
	}
	return Role{
		ID: row.ID, Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin, IsSystem: row.IsSystem,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *PostgresRepository) UpdateRole(ctx context.Context, projectID *uuid.UUID, id uuid.UUID, name, description string, isAdmin bool) (Role, error) {
	row, err := r.queries.UpdateRole(ctx, sqlc.UpdateRoleParams{
		ID: id, ProjectID: pgScope(projectID),
		Name: name, Description: description, IsAdmin: isAdmin,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Either the role doesn't exist, lives in a different scope, or is_system=true.
			existing, getErr := r.GetRole(ctx, projectID, id)
			if getErr != nil {
				return Role{}, ErrRoleNotFound
			}
			if existing.IsSystem {
				return Role{}, ErrSystemRole
			}
			return Role{}, ErrRoleNotFound
		}
		if isUniqueViolation(err) {
			return Role{}, ErrRoleExists
		}
		return Role{}, err
	}
	return Role{
		ID: row.ID, Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin, IsSystem: row.IsSystem,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *PostgresRepository) DeleteRole(ctx context.Context, projectID *uuid.UUID, id uuid.UUID) error {
	rows, err := r.queries.DeleteRole(ctx, sqlc.DeleteRoleParams{ID: id, ProjectID: pgScope(projectID)})
	if err != nil {
		return err
	}
	if rows == 0 {
		existing, getErr := r.GetRole(ctx, projectID, id)
		if getErr != nil {
			return ErrRoleNotFound
		}
		if existing.IsSystem {
			return ErrSystemRole
		}
		return ErrRoleNotFound
	}
	return nil
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
		policies = append(policies, Policy{RoleID: row.RoleID, Resource: row.Resource, Action: row.Action})
	}
	return policies, nil
}

func (r *PostgresRepository) DeletePolicy(ctx context.Context, roleID uuid.UUID, resource, action string) (int64, error) {
	return r.queries.DeleteRolePolicy(ctx, sqlc.DeleteRolePolicyParams{
		RoleID: roleID, Resource: resource, Action: action,
	})
}

func (r *PostgresRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleName string) (UserRole, error) {
	if _, err := r.GetRoleByName(ctx, nil, roleName); err != nil {
		return UserRole{}, err
	}
	row, err := r.queries.AssignUserRole(ctx, sqlc.AssignUserRoleParams{
		UserID: userID, Role: roleName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserRole{}, ErrUserNotFound
		}
		return UserRole{}, err
	}
	return UserRole{
		ID: row.ID, Name: row.Name, Email: row.Email, AppRole: row.AppRole,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *PostgresRepository) HasPolicy(ctx context.Context, role, resource, action string) (bool, error) {
	policies, err := r.queries.GetPoliciesByRoleName(ctx, role)
	if err != nil {
		return false, err
	}
	result := make([]rbac.Policy, 0, len(policies))
	for _, p := range policies {
		result = append(result, rbac.Policy{Resource: p.Resource, Action: p.Action})
	}
	return rbac.Allows(result, resource, action), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
