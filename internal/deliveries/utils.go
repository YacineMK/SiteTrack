package deliveries

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/projects"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(delivery Delivery) DeliveryResponse
	ListResponse(deliveries []Delivery, total int64, limit, offset int32) ListDeliveriesResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseID(value string) (uuid.UUID, error)
	NormalizePagination(limit, offset int32) (int32, int32)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (u utils) ToResponse(delivery Delivery) DeliveryResponse {
	lines := make([]DeliveryLineResponse, 0, len(delivery.Lines))
	for _, line := range delivery.Lines {
		lines = append(lines, DeliveryLineResponse{
			MaterialID: line.MaterialID,
			Name:       line.Name,
			Unit:       line.Unit,
			Quantity:   line.Quantity,
		})
	}

	return DeliveryResponse{
		ID:           delivery.ID,
		SenderType:   delivery.SenderType,
		SenderID:     delivery.SenderID,
		ReceiverType: delivery.ReceiverType,
		ReceiverID:   delivery.ReceiverID,
		Status:       delivery.Status,
		SentAt:       delivery.SentAt,
		ReceivedAt:   delivery.ReceivedAt,
		Lines:        lines,
	}
}

func (u utils) ListResponse(deliveries []Delivery, total int64, limit, offset int32) ListDeliveriesResponse {
	items := make([]DeliveryResponse, 0, len(deliveries))
	for _, delivery := range deliveries {
		items = append(items, u.ToResponse(delivery))
	}
	return ListDeliveriesResponse{
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
		jsonutil.Error(w, http.StatusNotFound, "delivery not found")
	case errors.Is(err, projects.ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "project not found")
	case errors.Is(err, ErrInsufficientStock):
		jsonutil.Error(w, http.StatusConflict, "insufficient stock")
	case errors.Is(err, ErrInvalidTransition):
		jsonutil.Error(w, http.StatusConflict, "invalid status transition")
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
