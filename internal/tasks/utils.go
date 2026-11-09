package tasks

import (
	"errors"
	"net/http"
	"time"

	"github.com/YacineMK/SiteTrack/internal/projects"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(task Task, projectDisplayID string) TaskResponse
	ListResponse(tasks []Task, projectDisplayID string, total int64, limit, offset int32) ListTasksResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseDates(start, end string) (time.Time, time.Time, error)
	NormalizePagination(limit, offset int32) (int32, int32)
	Late(task Task) bool
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (u utils) ToResponse(task Task, projectDisplayID string) TaskResponse {
	return TaskResponse{
		DisplayID:        task.DisplayID,
		ProjectDisplayID: projectDisplayID,
		Name:             task.Name,
		PlannedStartDate: task.PlannedStartDate.Format(dateLayout),
		PlannedEndDate:   task.PlannedEndDate.Format(dateLayout),
		Status:           task.Status,
		Late:             u.Late(task),
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
	}
}

func (u utils) ListResponse(tasks []Task, projectDisplayID string, total int64, limit, offset int32) ListTasksResponse {
	items := make([]TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, u.ToResponse(task, projectDisplayID))
	}
	return ListTasksResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

func (utils) WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		jsonutil.Error(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "task not found")
	case errors.Is(err, projects.ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "project not found")
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
	}
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

func (utils) NormalizePagination(limit, offset int32) (int32, int32) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
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
