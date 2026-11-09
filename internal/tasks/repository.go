package tasks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID               uuid.UUID
	DisplayID        string
	ProjectID        uuid.UUID
	Name             string
	PlannedStartDate time.Time
	PlannedEndDate   time.Time
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Repository interface {
	Create(ctx context.Context, task Task) (Task, error)
	Get(ctx context.Context, projectID uuid.UUID, displayID string) (Task, error)
	List(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]Task, int64, error)
	Update(ctx context.Context, projectID uuid.UUID, displayID, name string, start, end time.Time) (Task, error)
	Delete(ctx context.Context, projectID uuid.UUID, displayID string) (int64, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) Create(ctx context.Context, task Task) (Task, error) {
	row, err := r.queries.CreateTask(ctx, sqlc.CreateTaskParams{
		ID:               task.ID,
		DisplayID:        task.DisplayID,
		ProjectID:        task.ProjectID,
		Name:             task.Name,
		PlannedStartDate: task.PlannedStartDate,
		PlannedEndDate:   task.PlannedEndDate,
	})
	if err != nil {
		return Task{}, err
	}
	return toTask(row), nil
}

func (r *PostgresRepository) Get(ctx context.Context, projectID uuid.UUID, displayID string) (Task, error) {
	row, err := r.queries.GetTask(ctx, sqlc.GetTaskParams{
		ProjectID: projectID,
		DisplayID: displayID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	return toTask(row), nil
}

func (r *PostgresRepository) List(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]Task, int64, error) {
	total, err := r.queries.CountTasks(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListTasks(ctx, sqlc.ListTasksParams{
		ProjectID:  projectID,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	tasks := make([]Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, toTask(row))
	}
	return tasks, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, projectID uuid.UUID, displayID, name string, start, end time.Time) (Task, error) {
	row, err := r.queries.UpdateTask(ctx, sqlc.UpdateTaskParams{
		ProjectID:        projectID,
		DisplayID:        displayID,
		Name:             name,
		PlannedStartDate: start,
		PlannedEndDate:   end,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	return toTask(row), nil
}

func (r *PostgresRepository) Delete(ctx context.Context, projectID uuid.UUID, displayID string) (int64, error) {
	return r.queries.DeleteTask(ctx, sqlc.DeleteTaskParams{
		ProjectID: projectID,
		DisplayID: displayID,
	})
}

func toTask(row sqlc.Task) Task {
	return Task{
		ID:               row.ID,
		DisplayID:        row.DisplayID,
		ProjectID:        row.ProjectID,
		Name:             row.Name,
		PlannedStartDate: row.PlannedStartDate,
		PlannedEndDate:   row.PlannedEndDate,
		Status:           row.Status,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
