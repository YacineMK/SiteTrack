package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
	"github.com/YacineMK/SiteTrack/internal/notifications"
)

type AppNotificationPayload struct {
	OutboxID       uuid.UUID `json:"outbox_id"`
	NotificationID uuid.UUID `json:"notification_id"`
}

type AppNotificationHandler struct {
	queries *sqlc.Queries
	hub     *notifications.Hub
}

func NewAppNotificationHandler(db *pgxpool.Pool, hub *notifications.Hub) *AppNotificationHandler {
	return &AppNotificationHandler{queries: sqlc.New(db), hub: hub}
}

func (h *AppNotificationHandler) Handle(ctx context.Context, task *asynq.Task) error {
	var payload AppNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	row, err := h.queries.GetNotification(ctx, payload.NotificationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			failOutbox(ctx, h.queries, payload.OutboxID, "notification not found")
			return nil
		}
		return err
	}

	h.hub.Publish(row.UserID, notificationResponse(row))

	h.queries.CompleteOutbox(ctx, payload.OutboxID)
	return nil
}

func notificationResponse(row sqlc.Notification) notifications.NotificationResponse {
	var readAt *time.Time
	if row.ReadAt.Valid {
		value := row.ReadAt.Time
		readAt = &value
	}

	return notifications.NotificationResponse{
		ID:        row.ID,
		Type:      row.Type,
		Title:     row.Title,
		Body:      row.Body,
		Data:      json.RawMessage(row.Data),
		ReadAt:    readAt,
		CreatedAt: row.CreatedAt,
	}
}
