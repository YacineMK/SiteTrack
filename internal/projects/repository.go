package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var (
	ErrNotFound       = errors.New("project not found")
	ErrMemberNotFound = errors.New("project member not found")
	ErrLastAdmin      = errors.New("project must keep at least one admin")
)

type Project struct {
	ID               uuid.UUID
	DisplayID        string
	Name             string
	Description      string
	PlannedStartDate time.Time
	PlannedEndDate   time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	IsAdmin          bool
}

type Member struct {
	UserID    uuid.UUID
	Name      string
	Email     string
	IsAdmin   bool
	CreatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, project Project, adminID uuid.UUID) (Project, error)
	GetByDisplayID(ctx context.Context, displayID string) (Project, error)
	List(ctx context.Context, userID uuid.UUID, superAdmin bool, limit, offset int32) ([]Project, int64, error)
	Search(ctx context.Context, userID uuid.UUID, superAdmin bool, query string, limit, offset int32) ([]Project, int64, error)
	Update(ctx context.Context, displayID, name, description string, start, end time.Time) (Project, error)
	Delete(ctx context.Context, displayID string) (int64, error)
	ProjectAccess(ctx context.Context, displayID string, userID uuid.UUID) (uuid.UUID, bool, bool, bool, error)
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]Member, error)
	SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, isAdmin bool) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
}

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, queries: sqlc.New(pool)}
}

func (r *PostgresRepository) Create(ctx context.Context, project Project, adminID uuid.UUID) (Project, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		row, err := queries.CreateProject(ctx, sqlc.CreateProjectParams{
			ID:               project.ID,
			DisplayID:        project.DisplayID,
			Name:             project.Name,
			Description:      project.Description,
			PlannedStartDate: project.PlannedStartDate,
			PlannedEndDate:   project.PlannedEndDate,
		})
		if err != nil {
			return err
		}

		project = toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt, true)
		return queries.AddProjectMember(ctx, sqlc.AddProjectMemberParams{
			ProjectID: project.ID,
			UserID:    adminID,
			IsAdmin:   true,
		})
	})
	if err != nil {
		return Project{}, err
	}
	return project, nil
}

func (r *PostgresRepository) GetByDisplayID(ctx context.Context, displayID string) (Project, error) {
	row, err := r.queries.GetProjectByDisplayID(ctx, displayID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt, false), nil
}

func (r *PostgresRepository) List(ctx context.Context, userID uuid.UUID, superAdmin bool, limit, offset int32) ([]Project, int64, error) {
	total, err := r.queries.CountProjects(ctx, sqlc.CountProjectsParams{
		UserID:     userID,
		SuperAdmin: superAdmin,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListProjects(ctx, sqlc.ListProjectsParams{
		UserID:     userID,
		SuperAdmin: superAdmin,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	projects := make([]Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt, row.IsAdmin))
	}
	return projects, total, nil
}

func (r *PostgresRepository) Search(ctx context.Context, userID uuid.UUID, superAdmin bool, query string, limit, offset int32) ([]Project, int64, error) {
	total, err := r.queries.CountSearchProjects(ctx, sqlc.CountSearchProjectsParams{
		Query:      query,
		UserID:     userID,
		SuperAdmin: superAdmin,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.SearchProjects(ctx, sqlc.SearchProjectsParams{
		Query:      query,
		UserID:     userID,
		SuperAdmin: superAdmin,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	projects := make([]Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt, row.IsAdmin))
	}
	return projects, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, displayID, name, description string, start, end time.Time) (Project, error) {
	row, err := r.queries.UpdateProject(ctx, sqlc.UpdateProjectParams{
		DisplayID:        displayID,
		Name:             name,
		Description:      description,
		PlannedStartDate: start,
		PlannedEndDate:   end,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt, false), nil
}

func (r *PostgresRepository) Delete(ctx context.Context, displayID string) (int64, error) {
	return r.queries.DeleteProject(ctx, displayID)
}

func (r *PostgresRepository) ProjectAccess(ctx context.Context, displayID string, userID uuid.UUID) (uuid.UUID, bool, bool, bool, error) {
	project, err := r.GetByDisplayID(ctx, displayID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return uuid.UUID{}, false, false, false, nil
		}
		return uuid.UUID{}, false, false, false, err
	}

	isAdmin, err := r.queries.GetProjectMembership(ctx, sqlc.GetProjectMembershipParams{
		ProjectID: project.ID,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return project.ID, true, false, false, nil
		}
		return uuid.UUID{}, false, false, false, err
	}
	return project.ID, true, true, isAdmin, nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, projectID uuid.UUID) ([]Member, error) {
	rows, err := r.queries.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	members := make([]Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, Member{
			UserID:    row.UserID,
			Name:      row.Name,
			Email:     row.Email,
			IsAdmin:   row.IsAdmin,
			CreatedAt: row.CreatedAt,
		})
	}
	return members, nil
}

func (r *PostgresRepository) SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, isAdmin bool) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		if !isAdmin {
			if err := r.ensureAnotherAdmin(ctx, queries, projectID, userID); err != nil {
				return err
			}
		}

		rows, err := queries.SetProjectMemberAdmin(ctx, sqlc.SetProjectMemberAdminParams{
			ProjectID: projectID,
			UserID:    userID,
			IsAdmin:   isAdmin,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrMemberNotFound
		}
		return nil
	})
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		if err := r.ensureAnotherAdmin(ctx, queries, projectID, userID); err != nil {
			return err
		}

		rows, err := queries.RemoveProjectMember(ctx, sqlc.RemoveProjectMemberParams{
			ProjectID: projectID,
			UserID:    userID,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrMemberNotFound
		}
		return nil
	})
}

func (r *PostgresRepository) ensureAnotherAdmin(ctx context.Context, queries *sqlc.Queries, projectID, userID uuid.UUID) error {
	current, err := queries.GetProjectMembership(ctx, sqlc.GetProjectMembershipParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMemberNotFound
		}
		return err
	}
	if !current {
		return nil
	}

	admins, err := queries.LockProjectAdmins(ctx, projectID)
	if err != nil {
		return err
	}
	if len(admins) <= 1 {
		return ErrLastAdmin
	}
	return nil
}

func toProject(id uuid.UUID, displayID, name, description string, start, end, createdAt, updatedAt time.Time, isAdmin bool) Project {
	return Project{
		ID:               id,
		DisplayID:        displayID,
		Name:             name,
		Description:      description,
		PlannedStartDate: start,
		PlannedEndDate:   end,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		IsAdmin:          isAdmin,
	}
}
