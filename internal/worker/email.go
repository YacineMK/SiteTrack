package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

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
			h.fail(ctx, payload, "user not found")
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
			h.fail(ctx, payload, err.Error())
			return nil
		}
		if retry, _ := asynq.GetRetryCount(ctx); retry >= maxRetry(ctx) {
			h.fail(ctx, payload, err.Error())
			return nil
		}
		return err
	}

	if _, err := h.queries.CompleteOutbox(ctx, payload.OutboxID); err != nil {
		log.Printf("outbox complete %s: %v", payload.OutboxID, err)
	}
	return nil
}

func (h *EmailHandler) fail(ctx context.Context, payload EmailPayload, message string) {
	if _, err := h.queries.FailOutbox(ctx, sqlc.FailOutboxParams{
		ID:            payload.OutboxID,
		Status:        "failed",
		NextAttemptAt: time.Now(),
		LastError:     message,
	}); err != nil {
		log.Printf("outbox fail %s: %v", payload.OutboxID, err)
	}
}

func maxRetry(ctx context.Context) int {
	max, ok := asynq.GetMaxRetry(ctx)
	if !ok {
		return 0
	}
	return max
}
