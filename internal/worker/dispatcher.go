package worker

import (
	"context"
	"errors"
	"time"

	"github.com/hibiken/asynq"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

func (w *Worker) dispatch(ctx context.Context) {
	w.dispatchBatch(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.dispatchBatch(ctx)
		}
	}
}

func (w *Worker) dispatchBatch(ctx context.Context) {
	rows, err := w.queries.ClaimOutboxBatch(ctx, sqlc.ClaimOutboxBatchParams{
		WorkerID:    w.workerID,
		LockSeconds: int32(w.config.LockTimeout.Seconds()),
		BatchSize:   w.config.BatchSize,
	})
	if err != nil {
		return
	}

	for _, row := range rows {
		w.enqueue(ctx, row)
	}
}

func (w *Worker) enqueue(ctx context.Context, row sqlc.Outbox) {
	taskType, ok := taskType(row.Kind)
	if !ok {
		w.reschedule(ctx, row, "unknown outbox kind: "+row.Kind)
		return
	}

	_, err := w.client.Enqueue(asynq.NewTask(
		taskType,
		row.Payload,
		asynq.TaskID(row.ID.String()),
		asynq.Queue(QueueName),
		asynq.MaxRetry(w.config.MaxAttempts),
		asynq.Timeout(30*time.Second),
	))
	if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
		w.reschedule(ctx, row, err.Error())
		return
	}

	w.queries.MarkOutboxEnqueued(ctx, row.ID)
}

func (w *Worker) reschedule(ctx context.Context, row sqlc.Outbox, message string) {
	status := "pending"
	if int(row.Attempts) >= w.config.MaxAttempts {
		status = "failed"
	}

	w.queries.FailOutbox(ctx, sqlc.FailOutboxParams{
		ID:            row.ID,
		Status:        status,
		NextAttemptAt: time.Now().Add(backoff(row.Attempts)),
		LastError:     message,
	})
}

func backoff(attempts int32) time.Duration {
	seconds := 1 << min(int(attempts), 6)
	delay := time.Duration(seconds) * time.Second
	if delay > time.Minute {
		return time.Minute
	}
	return delay
}
