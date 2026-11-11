package projectstore

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/projects"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(stock Stock) ProjectStockResponse
	ListResponse(stock []Stock, total int64, limit, offset int32) ListProjectStockResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseID(value string) (uuid.UUID, error)
	NormalizePagination(limit, offset int32) (int32, int32)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (utils) ToResponse(stock Stock) ProjectStockResponse {
	return ProjectStockResponse(stock)
}

func (u utils) ListResponse(stock []Stock, total int64, limit, offset int32) ListProjectStockResponse {
	items := make([]ProjectStockResponse, 0, len(stock))
	for _, item := range stock {
		items = append(items, u.ToResponse(item))
	}
	return ListProjectStockResponse{
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
		jsonutil.Error(w, http.StatusNotFound, "stock not found")
	case errors.Is(err, projects.ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "project not found")
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (utils) ParseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, ErrInvalidInput
	}
	return id, nil
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
