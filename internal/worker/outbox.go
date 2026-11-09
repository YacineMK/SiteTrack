package worker

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

func failOutbox(ctx context.Context, queries *sqlc.Queries, id uuid.UUID, message string) {
	if _, err := queries.FailOutbox(ctx, sqlc.FailOutboxParams{
		ID:            id,
		Status:        "failed",
		NextAttemptAt: time.Now(),
		LastError:     message,
	}); err != nil {
		log.Printf("outbox fail %s: %v", id, err)
	}
}
