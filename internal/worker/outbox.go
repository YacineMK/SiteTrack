package worker

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

func failOutbox(ctx context.Context, queries *sqlc.Queries, id uuid.UUID, message string) {
	queries.FailOutbox(ctx, sqlc.FailOutboxParams{
		ID:            id,
		Status:        "failed",
		NextAttemptAt: time.Now(),
		LastError:     message,
	})
}
