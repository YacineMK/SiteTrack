package tasks

import (
	"errors"
	"net/http"
	"time"

	"github.com/YacineMK/SiteTrack/internal/projects"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(task Task, projectDisplayID string, blocked bool) TaskResponse
	ToDetailResponse(task Task, projectDisplayID string, upstream, downstream []Dependency, blocked bool) TaskDetailResponse
	ListResponse(tasks []Task, projectDisplayID string, total int64, limit int32) ListTasksResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseDates(start, end string) (time.Time, time.Time, error)
	ParseDateRange(from, to string) (*time.Time, *time.Time, error)
	NormalizeLimit(limit int32) int32
	Late(task Task) bool
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (u utils) ToResponse(task Task, projectDisplayID string, blocked bool) TaskResponse {
	return TaskResponse{
		DisplayID:        task.DisplayID,
		ProjectDisplayID: projectDisplayID,
		Name:             task.Name,
		PlannedStartDate: task.PlannedStartDate.Format(dateLayout),
		PlannedEndDate:   task.PlannedEndDate.Format(dateLayout),
		Status:           task.Status,
		Late:             u.Late(task),
		Blocked:          blocked,
		DependsOn:        task.DependsOn,
		DependentsCount:  task.DependentsCount,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
	}
}

func (u utils) ToDetailResponse(task Task, projectDisplayID string, upstream, downstream []Dependency, blocked bool) TaskDetailResponse {
	return TaskDetailResponse{
		TaskResponse: u.ToResponse(task, projectDisplayID, blocked),
		Upstream:     u.dependencies(upstream),
		Downstream:   u.dependencies(downstream),
	}
}

func (utils) dependencies(deps []Dependency) []DependencyResponse {
	items := make([]DependencyResponse, 0, len(deps))
	for _, dep := range deps {
		items = append(items, DependencyResponse{
			DisplayID: dep.DisplayID,
			Name:      dep.Name,
			Status:    dep.Status,
			Steps:     dep.Steps,
		})
	}
	return items
}

func (u utils) ListResponse(tasks []Task, projectDisplayID string, total int64, limit int32) ListTasksResponse {
	items := make([]TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, u.ToResponse(task, projectDisplayID, task.Status != StatusDone && task.Blocked))
	}
	return ListTasksResponse{
		Items: items,
		Total: total,
		Limit: limit,
	}
}

func (utils) WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		jsonutil.Error(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "task not found")
	case errors.Is(err, ErrDependencyNotFound):
		jsonutil.Error(w, http.StatusNotFound, "dependency not found")
	case errors.Is(err, projects.ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "project not found")
	case errors.Is(err, ErrInvalidTransition):
		jsonutil.Error(w, http.StatusConflict, "invalid status transition")
	case errors.Is(err, ErrBlocked):
		jsonutil.Error(w, http.StatusConflict, "task is blocked by incomplete dependencies")
	case errors.Is(err, ErrCycle):
		jsonutil.Error(w, http.StatusConflict, "dependency cycle detected")
	case errors.Is(err, ErrDependencyExists):
		jsonutil.Error(w, http.StatusConflict, "dependency already exists")
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (utils) ParseDateRange(from, to string) (*time.Time, *time.Time, error) {
	var start, end *time.Time
	if from != "" {
		parsed, err := time.Parse(dateLayout, from)
		if err != nil {
			return nil, nil, ErrInvalidInput
		}
		start = &parsed
	}
	if to != "" {
		parsed, err := time.Parse(dateLayout, to)
		if err != nil {
			return nil, nil, ErrInvalidInput
		}
		end = &parsed
	}
	if start != nil && end != nil && end.Before(*start) {
		return nil, nil, ErrInvalidInput
	}
	return start, end, nil
}

func (utils) ParseDates(start, end string) (time.Time, time.Time, error) {
	startDate, err := time.Parse(dateLayout, start)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}

	endDate, err := time.Parse(dateLayout, end)
	if err != nil || endDate.Before(startDate) {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}

	return startDate, endDate, nil
}

func (utils) NormalizeLimit(limit int32) int32 {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (utils) Late(task Task) bool {
	today := time.Now().Format(dateLayout)

	switch task.Status {
	case StatusNotStarted:
		return task.PlannedStartDate.Format(dateLayout) < today
	case StatusInProgress:
		return task.PlannedEndDate.Format(dateLayout) < today
	default:
		return false
	}
}
