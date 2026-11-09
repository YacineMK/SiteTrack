package materials

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(material Material) MaterialResponse
	ListResponse(materials []Material, total int64, limit, offset int32) ListMaterialsResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseID(value string) (uuid.UUID, error)
	NormalizePagination(limit, offset int32) (int32, int32)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (utils) ToResponse(material Material) MaterialResponse {
	return MaterialResponse{
		ID:        material.ID,
		Name:      material.Name,
		Category:  material.Category,
		Unit:      material.Unit,
		CreatedAt: material.CreatedAt,
		UpdatedAt: material.UpdatedAt,
	}
}

func (u utils) ListResponse(materials []Material, total int64, limit, offset int32) ListMaterialsResponse {
	items := make([]MaterialResponse, 0, len(materials))
	for _, material := range materials {
		items = append(items, u.ToResponse(material))
	}
	return ListMaterialsResponse{
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
		jsonutil.Error(w, http.StatusNotFound, "material not found")
	case errors.Is(err, ErrDuplicate):
		jsonutil.Error(w, http.StatusConflict, "material already exists")
	case errors.Is(err, ErrInUse):
		jsonutil.Error(w, http.StatusConflict, "material is used by deliveries")
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
