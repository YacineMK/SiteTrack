package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/auth"
	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
	"github.com/YacineMK/SiteTrack/kit/resend"
)

type EmailPayload struct {
	OutboxID uuid.UUID `json:"outbox_id"`
	UserID   uuid.UUID `json:"user_id"`
	Subject  string    `json:"subject"`
	HTML     string    `json:"html"`
	Text     string    `json:"text"`
}

type EmailHandler struct {
	queries *sqlc.Queries
	users   auth.Repository
	mail    *resend.Client
}

func NewEmailHandler(db *pgxpool.Pool, mail *resend.Client) *EmailHandler {
	return &EmailHandler{
		queries: sqlc.New(db),
		users:   auth.NewPostgresRepository(db),
		mail:    mail,
	}
}

func (h *EmailHandler) Handle(ctx context.Context, task *asynq.Task) error {
	var payload EmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	user, err := h.users.GetByID(ctx, payload.UserID)
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			failOutbox(ctx, h.queries, payload.OutboxID, "user not found")
			return nil
		}
		return err
	}

	_, err = h.mail.Send(ctx, resend.Email{
		To:      []string{user.Email},
		Subject: payload.Subject,
		HTML:    payload.HTML,
		Text:    payload.Text,
	})
	if err != nil {
		if errors.Is(err, resend.ErrInvalidEmail) || errors.Is(err, resend.ErrNotConfigured) {
			failOutbox(ctx, h.queries, payload.OutboxID, err.Error())
			return nil
		}
		if retry, _ := asynq.GetRetryCount(ctx); retry >= maxRetry(ctx) {
			failOutbox(ctx, h.queries, payload.OutboxID, err.Error())
			return nil
		}
		return err
	}

	h.queries.CompleteOutbox(ctx, payload.OutboxID)
	return nil
}

func maxRetry(ctx context.Context) int {
	max, ok := asynq.GetMaxRetry(ctx)
	if !ok {
		return 0
	}
	return max
}
