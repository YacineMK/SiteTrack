package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var (
	ErrNotFound         = errors.New("task not found")
	ErrDependencyExists = errors.New("dependency already exists")
)

type Dependency struct {
	DisplayID string
	Name      string
	Status    string
	Steps     int32
}

type Task struct {
	ID               uuid.UUID
	DisplayID        string
	ProjectID        uuid.UUID
	Name             string
	PlannedStartDate time.Time
	PlannedEndDate   time.Time
	Status           string
	DependsOn        []string
	DependentsCount  int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListParams struct {
	ProjectID   uuid.UUID
	Query       string
	Status      string
	FromDate    string
	ToDate      string
	SortBy      string
	SortDir     string
	LateOnly    bool
	BlockedOnly bool
	BlockedIDs  []uuid.UUID
	Limit       int32
	Offset      int32
}

type Repository interface {
	Create(ctx context.Context, task Task) (Task, error)
	Get(ctx context.Context, projectID uuid.UUID, displayID string) (Task, error)
	List(ctx context.Context, params ListParams) ([]Task, int64, error)
	Update(ctx context.Context, projectID uuid.UUID, displayID, name string, start, end time.Time) (Task, error)
	UpdateStatus(ctx context.Context, projectID uuid.UUID, displayID, status string) (Task, error)
	Delete(ctx context.Context, projectID uuid.UUID, displayID string) (int64, error)
	AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) (int64, error)
	RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) (int64, error)
	HasBlockingUpstream(ctx context.Context, taskID uuid.UUID) (bool, error)
	HasStartedDownstream(ctx context.Context, taskID uuid.UUID) (bool, error)
	Upstream(ctx context.Context, taskID uuid.UUID) ([]Dependency, error)
	Downstream(ctx context.Context, taskID uuid.UUID) ([]Dependency, error)
	BlockedIDs(ctx context.Context, projectID uuid.UUID) (map[uuid.UUID]bool, error)
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

func (r *PostgresRepository) List(ctx context.Context, params ListParams) ([]Task, int64, error) {
	total, err := r.queries.CountTasks(ctx, sqlc.CountTasksParams{
		ProjectID:   params.ProjectID,
		Query:       params.Query,
		Status:      params.Status,
		FromDate:    params.FromDate,
		ToDate:      params.ToDate,
		LateOnly:    params.LateOnly,
		BlockedOnly: params.BlockedOnly,
		BlockedIds:  params.BlockedIDs,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListTasks(ctx, sqlc.ListTasksParams{
		ProjectID:   params.ProjectID,
		Query:       params.Query,
		Status:      params.Status,
		FromDate:    params.FromDate,
		ToDate:      params.ToDate,
		LateOnly:    params.LateOnly,
		BlockedOnly: params.BlockedOnly,
		BlockedIds:  params.BlockedIDs,
		SortBy:      params.SortBy,
		SortDir:     params.SortDir,
		PageLimit:   params.Limit,
		PageOffset:  params.Offset,
	})
	if err != nil {
		return nil, 0, err
	}

	tasks := make([]Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, toListTask(row))
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

func (r *PostgresRepository) UpdateStatus(ctx context.Context, projectID uuid.UUID, displayID, status string) (Task, error) {
	row, err := r.queries.UpdateTaskStatus(ctx, sqlc.UpdateTaskStatusParams{
		ProjectID: projectID,
		DisplayID: displayID,
		Status:    status,
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

func (r *PostgresRepository) AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) (int64, error) {
	rows, err := r.queries.AddTaskDependency(ctx, sqlc.AddTaskDependencyParams{
		NewTaskID:   taskID,
		DependsOnID: dependsOnID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrDependencyExists
		}
		return 0, err
	}
	return rows, nil
}

func (r *PostgresRepository) RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) (int64, error) {
	return r.queries.RemoveTaskDependency(ctx, sqlc.RemoveTaskDependencyParams{
		TaskID:    taskID,
		DependsOn: dependsOnID,
	})
}

func (r *PostgresRepository) HasBlockingUpstream(ctx context.Context, taskID uuid.UUID) (bool, error) {
	return r.queries.TaskHasBlockingUpstream(ctx, taskID)
}

func (r *PostgresRepository) HasStartedDownstream(ctx context.Context, taskID uuid.UUID) (bool, error) {
	return r.queries.TaskHasStartedDownstream(ctx, taskID)
}

func (r *PostgresRepository) Upstream(ctx context.Context, taskID uuid.UUID) ([]Dependency, error) {
	rows, err := r.queries.ListUpstreamDependencies(ctx, taskID)
	if err != nil {
		return nil, err
	}
	deps := make([]Dependency, 0, len(rows))
	for _, row := range rows {
		deps = append(deps, Dependency{DisplayID: row.DisplayID, Name: row.Name, Status: row.Status, Steps: row.Steps})
	}
	return deps, nil
}

func (r *PostgresRepository) Downstream(ctx context.Context, taskID uuid.UUID) ([]Dependency, error) {
	rows, err := r.queries.ListDownstreamDependencies(ctx, taskID)
	if err != nil {
		return nil, err
	}
	deps := make([]Dependency, 0, len(rows))
	for _, row := range rows {
		deps = append(deps, Dependency{DisplayID: row.DisplayID, Name: row.Name, Status: row.Status, Steps: row.Steps})
	}
	return deps, nil
}

func (r *PostgresRepository) BlockedIDs(ctx context.Context, projectID uuid.UUID) (map[uuid.UUID]bool, error) {
	ids, err := r.queries.ListBlockedTasks(ctx, projectID)
	if err != nil {
		return nil, err
	}

	blocked := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		blocked[id] = true
	}
	return blocked, nil
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

func toListTask(row sqlc.ListTasksRow) Task {
	var dependsOn []string
	_ = json.Unmarshal([]byte(row.DependsOn), &dependsOn)
	return Task{
		ID:               row.ID,
		DisplayID:        row.DisplayID,
		ProjectID:        row.ProjectID,
		Name:             row.Name,
		PlannedStartDate: row.PlannedStartDate,
		PlannedEndDate:   row.PlannedEndDate,
		Status:           row.Status,
		DependsOn:        dependsOn,
		DependentsCount:  row.DependentsCount,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
