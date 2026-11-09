package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var ErrNotFound = errors.New("project not found")

type Project struct {
	ID               uuid.UUID
	DisplayID        string
	Name             string
	Description      string
	PlannedStartDate time.Time
	PlannedEndDate   time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Repository interface {
	Create(ctx context.Context, project Project) (Project, error)
	GetByDisplayID(ctx context.Context, displayID string) (Project, error)
	List(ctx context.Context, limit, offset int32) ([]Project, int64, error)
	Search(ctx context.Context, query string, limit, offset int32) ([]Project, int64, error)
	Update(ctx context.Context, displayID, name, description string, start, end time.Time) (Project, error)
	Delete(ctx context.Context, displayID string) (int64, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) Create(ctx context.Context, project Project) (Project, error) {
	row, err := r.queries.CreateProject(ctx, sqlc.CreateProjectParams{
		ID:               project.ID,
		DisplayID:        project.DisplayID,
		Name:             project.Name,
		Description:      project.Description,
		PlannedStartDate: project.PlannedStartDate,
		PlannedEndDate:   project.PlannedEndDate,
	})
	if err != nil {
		return Project{}, err
	}
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt), nil
}

func (r *PostgresRepository) GetByDisplayID(ctx context.Context, displayID string) (Project, error) {
	row, err := r.queries.GetProjectByDisplayID(ctx, displayID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt), nil
}

func (r *PostgresRepository) List(ctx context.Context, limit, offset int32) ([]Project, int64, error) {
	total, err := r.queries.CountProjects(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListProjects(ctx, sqlc.ListProjectsParams{
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	projects := make([]Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt))
	}
	return projects, total, nil
}

func (r *PostgresRepository) Search(ctx context.Context, query string, limit, offset int32) ([]Project, int64, error) {
	total, err := r.queries.CountSearchProjects(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.SearchProjects(ctx, sqlc.SearchProjectsParams{
		Query:      query,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	projects := make([]Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt))
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
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt), nil
}

func (r *PostgresRepository) Delete(ctx context.Context, displayID string) (int64, error) {
	return r.queries.DeleteProject(ctx, displayID)
}

func toProject(id uuid.UUID, displayID, name, description string, start, end, createdAt, updatedAt time.Time) Project {
	return Project{
		ID:               id,
		DisplayID:        displayID,
		Name:             name,
		Description:      description,
		PlannedStartDate: start,
		PlannedEndDate:   end,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}
