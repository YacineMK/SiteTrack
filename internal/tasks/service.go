package tasks

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/projects"
	ids "github.com/YacineMK/SiteTrack/kit/idgen"
)

const (
	StatusNotStarted = "not_started"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrBlocked            = errors.New("task is blocked")
	ErrCycle              = errors.New("dependency cycle detected")
	ErrDependencyNotFound = errors.New("dependency not found")
)

type Service interface {
	Create(ctx context.Context, projectDisplayID string, req CreateTaskRequest) (TaskResponse, error)
	Get(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskDetailResponse, error)
	List(ctx context.Context, projectDisplayID, query string, limit, offset int32) (ListTasksResponse, error)
	Update(ctx context.Context, projectDisplayID, taskDisplayID string, req UpdateTaskRequest) (TaskResponse, error)
	Delete(ctx context.Context, projectDisplayID, taskDisplayID string) error
	Start(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error)
	Done(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error)
	Reopen(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error)
	AddDependency(ctx context.Context, projectDisplayID, taskDisplayID, dependsOnDisplayID string) error
	RemoveDependency(ctx context.Context, projectDisplayID, taskDisplayID, dependsOnDisplayID string) error
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

	return s.utils.ToResponse(task, project.DisplayID, false), nil
}

func (s *service) Get(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskDetailResponse, error) {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return TaskDetailResponse{}, err
	}

	task, err := s.repo.Get(ctx, project.ID, taskDisplayID)
	if err != nil {
		return TaskDetailResponse{}, err
	}

	upstream, err := s.repo.Upstream(ctx, task.ID)
	if err != nil {
		return TaskDetailResponse{}, err
	}

	downstream, err := s.repo.Downstream(ctx, task.ID)
	if err != nil {
		return TaskDetailResponse{}, err
	}

	blocked := false
	if task.Status != StatusDone {
		blocked, err = s.repo.HasBlockingUpstream(ctx, task.ID)
		if err != nil {
			return TaskDetailResponse{}, err
		}
	}

	return s.utils.ToDetailResponse(task, project.DisplayID, upstream, downstream, blocked), nil
}

func (s *service) List(ctx context.Context, projectDisplayID, query string, limit, offset int32) (ListTasksResponse, error) {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return ListTasksResponse{}, err
	}

	limit, offset = s.utils.NormalizePagination(limit, offset)

	tasks, total, err := s.repo.List(ctx, project.ID, query, limit, offset)
	if err != nil {
		return ListTasksResponse{}, err
	}

	blocked, err := s.repo.BlockedIDs(ctx, project.ID)
	if err != nil {
		return ListTasksResponse{}, err
	}

	return s.utils.ListResponse(tasks, project.DisplayID, blocked, total, limit, offset), nil
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

	blocked := false
	if task.Status != StatusDone {
		blocked, err = s.repo.HasBlockingUpstream(ctx, task.ID)
		if err != nil {
			return TaskResponse{}, err
		}
	}

	return s.utils.ToResponse(task, project.DisplayID, blocked), nil
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

func (s *service) Start(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error) {
	return s.transition(ctx, projectDisplayID, taskDisplayID, StatusNotStarted, StatusInProgress, s.repo.HasBlockingUpstream)
}

func (s *service) Done(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error) {
	return s.transition(ctx, projectDisplayID, taskDisplayID, StatusInProgress, StatusDone, nil)
}

func (s *service) Reopen(ctx context.Context, projectDisplayID, taskDisplayID string) (TaskResponse, error) {
	return s.transition(ctx, projectDisplayID, taskDisplayID, StatusDone, StatusInProgress, s.repo.HasStartedDownstream)
}

func (s *service) transition(ctx context.Context, projectDisplayID, taskDisplayID, from, to string, blockedCheck func(context.Context, uuid.UUID) (bool, error)) (TaskResponse, error) {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return TaskResponse{}, err
	}

	task, err := s.repo.Get(ctx, project.ID, taskDisplayID)
	if err != nil {
		return TaskResponse{}, err
	}
	if task.Status != from {
		return TaskResponse{}, ErrInvalidTransition
	}

	if blockedCheck != nil {
		blocked, err := blockedCheck(ctx, task.ID)
		if err != nil {
			return TaskResponse{}, err
		}
		if blocked {
			return TaskResponse{}, ErrBlocked
		}
	}

	updated, err := s.repo.UpdateStatus(ctx, project.ID, taskDisplayID, to)
	if err != nil {
		return TaskResponse{}, err
	}

	blocked := false
	if updated.Status != StatusDone {
		blocked, err = s.repo.HasBlockingUpstream(ctx, updated.ID)
		if err != nil {
			return TaskResponse{}, err
		}
	}

	return s.utils.ToResponse(updated, project.DisplayID, blocked), nil
}

func (s *service) AddDependency(ctx context.Context, projectDisplayID, taskDisplayID, dependsOnDisplayID string) error {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return err
	}

	task, err := s.repo.Get(ctx, project.ID, taskDisplayID)
	if err != nil {
		return err
	}

	dependsOn, err := s.repo.Get(ctx, project.ID, dependsOnDisplayID)
	if err != nil {
		return err
	}
	if task.ID == dependsOn.ID {
		return ErrInvalidInput
	}

	rows, err := s.repo.AddDependency(ctx, task.ID, dependsOn.ID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCycle
	}
	return nil
}

func (s *service) RemoveDependency(ctx context.Context, projectDisplayID, taskDisplayID, dependsOnDisplayID string) error {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return err
	}

	task, err := s.repo.Get(ctx, project.ID, taskDisplayID)
	if err != nil {
		return err
	}

	dependsOn, err := s.repo.Get(ctx, project.ID, dependsOnDisplayID)
	if err != nil {
		return err
	}

	rows, err := s.repo.RemoveDependency(ctx, task.ID, dependsOn.ID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDependencyNotFound
	}
	return nil
}
