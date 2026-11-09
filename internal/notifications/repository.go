package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var ErrNotFound = errors.New("notification not found")

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Title     string
	Body      string
	Data      []byte
	ReadAt    *time.Time
	CreatedAt time.Time
}

type Repository interface {
	List(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int32) ([]Notification, int64, error)
	Count(ctx context.Context, userID uuid.UUID, unreadOnly bool) (int64, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) (Notification, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int32) ([]Notification, int64, error) {
	total, err := r.queries.CountNotifications(ctx, sqlc.CountNotificationsParams{
		UserID:     userID,
		UnreadOnly: unreadOnly,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListNotifications(ctx, sqlc.ListNotificationsParams{
		UserID:     userID,
		UnreadOnly: unreadOnly,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]Notification, 0, len(rows))
	for _, row := range rows {
		items = append(items, toNotification(row))
	}
	return items, total, nil
}

func (r *PostgresRepository) Count(ctx context.Context, userID uuid.UUID, unreadOnly bool) (int64, error) {
	return r.queries.CountNotifications(ctx, sqlc.CountNotificationsParams{
		UserID:     userID,
		UnreadOnly: unreadOnly,
	})
}

func (r *PostgresRepository) MarkRead(ctx context.Context, id, userID uuid.UUID) (Notification, error) {
	row, err := r.queries.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotFound
		}
		return Notification{}, err
	}
	return toNotification(row), nil
}

func toNotification(row sqlc.Notification) Notification {
	var readAt *time.Time
	if row.ReadAt.Valid {
		value := row.ReadAt.Time
		readAt = &value
	}

	return Notification{
		ID:        row.ID,
		UserID:    row.UserID,
		Type:      row.Type,
		Title:     row.Title,
		Body:      row.Body,
		Data:      row.Data,
		ReadAt:    readAt,
		CreatedAt: row.CreatedAt,
	}
}
