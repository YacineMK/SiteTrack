package notifications

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(notification Notification) NotificationResponse
	ListResponse(items []Notification, total int64, limit, offset int32) ListNotificationsResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseID(value string) (uuid.UUID, error)
	NormalizePagination(limit, offset int32) (int32, int32)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (utils) ToResponse(notification Notification) NotificationResponse {
	return NotificationResponse{
		ID:        notification.ID,
		Type:      notification.Type,
		Title:     notification.Title,
		Body:      notification.Body,
		Data:      json.RawMessage(notification.Data),
		ReadAt:    notification.ReadAt,
		CreatedAt: notification.CreatedAt,
	}
}

func (u utils) ListResponse(items []Notification, total int64, limit, offset int32) ListNotificationsResponse {
	response := make([]NotificationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, u.ToResponse(item))
	}
	return ListNotificationsResponse{
		Items:  response,
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
		jsonutil.Error(w, http.StatusNotFound, "notification not found")
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
