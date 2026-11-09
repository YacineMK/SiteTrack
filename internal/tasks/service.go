package tasks

import (
	"context"
	"errors"
	"strings"

	"github.com/YacineMK/SiteTrack/internal/projects"
	ids "github.com/YacineMK/SiteTrack/kit/idgen"
)

const (
	StatusNotStarted = "not_started"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	Create(ctx context.Context, projectDisplayID string, req CreateTaskRequest) (TaskResponse, error)
	Get(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error)
	List(ctx context.Context, projectDisplayID string, limit, offset int32) (ListTasksResponse, error)
	Update(ctx context.Context, projectDisplayID, taskDisplayID string, req UpdateTaskRequest) (TaskResponse, error)
	Delete(ctx context.Context, projectDisplayID, taskDisplayID string) error
}

type service struct {
	repo     Repository
	projects projects.Repository
	ids      *ids.Generator
	utils    Utils
}

func NewService(repo Repository, projectRepo projects.Repository, generator *ids.Generator) Service {
	return &service{repo: repo, projects: projectRepo, ids: generator, utils: NewUtils()}
}

func (s *service) Create(ctx context.Context, projectDisplayID string, req CreateTaskRequest) (TaskResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if err := req.Validate(); err != nil {
		return TaskResponse{}, ErrInvalidInput
	}

	start, end, err := s.utils.ParseDates(req.PlannedStartDate, req.PlannedEndDate)
	if err != nil {
		return TaskResponse{}, err
	}

	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return TaskResponse{}, err
	}

	generated, err := s.ids.NewTask(ctx, project.ID, project.DisplayID)
	if err != nil {
		return TaskResponse{}, err
	}

	task, err := s.repo.Create(ctx, Task{
		ID:               generated.UUID,
		DisplayID:        generated.DisplayID,
		ProjectID:        project.ID,
		Name:             req.Name,
		PlannedStartDate: start,
		PlannedEndDate:   end,
	})
	if err != nil {
		return TaskResponse{}, err
	}

	return s.utils.ToResponse(task, project.DisplayID), nil
}

func (s *service) Get(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error) {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return TaskResponse{}, err
	}

	task, err := s.repo.Get(ctx, project.ID, taskDisplayID)
	if err != nil {
		return TaskResponse{}, err
	}

	return s.utils.ToResponse(task, project.DisplayID), nil
}

func (s *service) List(ctx context.Context, projectDisplayID string, limit, offset int32) (ListTasksResponse, error) {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return ListTasksResponse{}, err
	}

	limit, offset = s.utils.NormalizePagination(limit, offset)

	tasks, total, err := s.repo.List(ctx, project.ID, limit, offset)
	if err != nil {
		return ListTasksResponse{}, err
	}

	return s.utils.ListResponse(tasks, project.DisplayID, total, limit, offset), nil
}

func (s *service) Update(ctx context.Context, projectDisplayID, taskDisplayID string, req UpdateTaskRequest) (TaskResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if err := req.Validate(); err != nil {
		return TaskResponse{}, ErrInvalidInput
	}

	start, end, err := s.utils.ParseDates(req.PlannedStartDate, req.PlannedEndDate)
	if err != nil {
		return TaskResponse{}, err
	}

	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return TaskResponse{}, err
	}

	task, err := s.repo.Update(ctx, project.ID, taskDisplayID, req.Name, start, end)
	if err != nil {
		return TaskResponse{}, err
	}

	return s.utils.ToResponse(task, project.DisplayID), nil
}

func (s *service) Delete(ctx context.Context, projectDisplayID, taskDisplayID string) error {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return err
	}

	rows, err := s.repo.Delete(ctx, project.ID, taskDisplayID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
