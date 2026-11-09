package projects

import (
	"errors"
	"net/http"
	"time"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(project Project) ProjectResponse
	ListResponse(projects []Project, total int64, limit, offset int32) ListProjectsResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseDates(start, end string) (time.Time, time.Time, error)
	NormalizePagination(limit, offset int32) (int32, int32)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (utils) ToResponse(project Project) ProjectResponse {
	return ProjectResponse{
		DisplayID:        project.DisplayID,
		Name:             project.Name,
		Description:      project.Description,
		PlannedStartDate: project.PlannedStartDate.Format(dateLayout),
		PlannedEndDate:   project.PlannedEndDate.Format(dateLayout),
		CreatedAt:        project.CreatedAt,
		UpdatedAt:        project.UpdatedAt,
	}
}

func (u utils) ListResponse(projects []Project, total int64, limit, offset int32) ListProjectsResponse {
	items := make([]ProjectResponse, 0, len(projects))
	for _, project := range projects {
		items = append(items, u.ToResponse(project))
	}
	return ListProjectsResponse{
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
		jsonutil.Error(w, http.StatusNotFound, "project not found")
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
	}
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
