package notifications

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	List(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int32) (ListNotificationsResponse, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkRead(ctx context.Context, userID uuid.UUID, id string) (NotificationResponse, error)
}

type service struct {
	repo  Repository
	utils Utils
}

func NewService(repo Repository) Service {
	return &service{repo: repo, utils: NewUtils()}
}

func (s *service) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int32) (ListNotificationsResponse, error) {
	limit, offset = s.utils.NormalizePagination(limit, offset)

	items, total, err := s.repo.List(ctx, userID, unreadOnly, limit, offset)
	if err != nil {
		return ListNotificationsResponse{}, err
	}

	return s.utils.ListResponse(items, total, limit, offset), nil
}

func (s *service) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.Count(ctx, userID, true)
}

func (s *service) MarkRead(ctx context.Context, userID uuid.UUID, id string) (NotificationResponse, error) {
	notificationID, err := s.utils.ParseID(id)
	if err != nil {
		return NotificationResponse{}, err
	}

	notification, err := s.repo.MarkRead(ctx, notificationID, userID)
	if err != nil {
		return NotificationResponse{}, err
	}

	return s.utils.ToResponse(notification), nil
}
