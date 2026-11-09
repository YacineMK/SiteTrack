package deliveries

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

const (
	PartyCompany = "company"
	PartyProject = "project"

	StatusInTransit = "in_transit"
	StatusDelivered = "delivered"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
)

var (
	ErrNotFound          = errors.New("delivery not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type Line struct {
	MaterialID uuid.UUID
	Name       string
	Unit       string
	Quantity   float64
}

type Delivery struct {
	ID             uuid.UUID
	SenderType     string
	SenderID       *uuid.UUID
	ReceiverType   string
	ReceiverID     *uuid.UUID
	Status         string
	SentAt         time.Time
	ReceivedAt     *time.Time
	SentBy         *uuid.UUID
	ReceivedBy     *uuid.UUID
	SentByName     string
	ReceivedByName string
	Lines          []Line
}

type Repository interface {
	Create(ctx context.Context, delivery Delivery) (Delivery, error)
	Get(ctx context.Context, id uuid.UUID) (Delivery, error)
	ListCompany(ctx context.Context, status, query, direction, fromDate, toDate string, projectID uuid.UUID, limit, offset int32) ([]Delivery, int64, error)
	ListProject(ctx context.Context, projectID uuid.UUID, status, query, direction, fromDate, toDate string, limit, offset int32) ([]Delivery, int64, error)
	Transition(ctx context.Context, id uuid.UUID, status string, receivedBy *uuid.UUID) (Delivery, error)
}

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, queries: sqlc.New(pool)}
}

func (r *PostgresRepository) Create(ctx context.Context, delivery Delivery) (Delivery, error) {
	var id uuid.UUID

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)

		for _, line := range delivery.Lines {
			var rows int64
			var err error
			if delivery.SenderType == PartyCompany {
				rows, err = q.DeductCompanyStock(ctx, sqlc.DeductCompanyStockParams{
					MaterialID: line.MaterialID,
					Quantity:   line.Quantity,
				})
			} else {
				rows, err = q.DeductProjectStock(ctx, sqlc.DeductProjectStockParams{
					ProjectID:  *delivery.SenderID,
					MaterialID: line.MaterialID,
					Quantity:   line.Quantity,
				})
			}
			if err != nil {
				return err
			}
			if rows == 0 {
				return ErrInsufficientStock
			}
		}

		createdID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		id = createdID

		if _, err := q.CreateDelivery(ctx, sqlc.CreateDeliveryParams{
			ID:           id,
			SenderType:   sqlc.DeliveryParty(delivery.SenderType),
			SenderID:     toPgUUID(delivery.SenderID),
			ReceiverType: sqlc.DeliveryParty(delivery.ReceiverType),
			ReceiverID:   toPgUUID(delivery.ReceiverID),
			SentBy:       toPgUUID(delivery.SentBy),
		}); err != nil {
			return err
		}

		for _, line := range delivery.Lines {
			lineID, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := q.CreateDeliveryLine(ctx, sqlc.CreateDeliveryLineParams{
				ID:         lineID,
				DeliveryID: id,
				MaterialID: line.MaterialID,
				Quantity:   line.Quantity,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return Delivery{}, err
	}

	return r.Get(ctx, id)
}

func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (Delivery, error) {
	row, err := r.queries.GetDelivery(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Delivery{}, ErrNotFound
		}
		return Delivery{}, err
	}

	rows, err := r.queries.GetDeliveryLines(ctx, id)
	if err != nil {
		return Delivery{}, err
	}

	delivery := toDelivery(row)
	delivery.Lines = make([]Line, 0, len(rows))
	for _, line := range rows {
		delivery.Lines = append(delivery.Lines, Line{
			MaterialID: line.MaterialID,
			Name:       line.Name,
			Unit:       line.Unit,
			Quantity:   line.Quantity,
		})
	}
	return delivery, nil
}

func (r *PostgresRepository) ListCompany(ctx context.Context, status, query, direction, fromDate, toDate string, projectID uuid.UUID, limit, offset int32) ([]Delivery, int64, error) {
	total, err := r.queries.CountCompanyDeliveries(ctx, sqlc.CountCompanyDeliveriesParams{
		Direction:    direction,
		ProjectID:    projectID,
		StatusFilter: status,
		FromDate:     fromDate,
		ToDate:       toDate,
		Query:        query,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListCompanyDeliveries(ctx, sqlc.ListCompanyDeliveriesParams{
		Direction:    direction,
		ProjectID:    projectID,
		StatusFilter: status,
		FromDate:     fromDate,
		ToDate:       toDate,
		Query:        query,
		PageLimit:    limit,
		PageOffset:   offset,
	})
	if err != nil {
		return nil, 0, err
	}

	return toCompanyDeliveries(rows), total, nil
}

func (r *PostgresRepository) ListProject(ctx context.Context, projectID uuid.UUID, status, query, direction, fromDate, toDate string, limit, offset int32) ([]Delivery, int64, error) {
	total, err := r.queries.CountProjectDeliveries(ctx, sqlc.CountProjectDeliveriesParams{
		ProjectID:    projectID,
		Direction:    direction,
		StatusFilter: status,
		FromDate:     fromDate,
		ToDate:       toDate,
		Query:        query,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListProjectDeliveries(ctx, sqlc.ListProjectDeliveriesParams{
		ProjectID:    projectID,
		Direction:    direction,
		StatusFilter: status,
		FromDate:     fromDate,
		ToDate:       toDate,
		Query:        query,
		PageLimit:    limit,
		PageOffset:   offset,
	})
	if err != nil {
		return nil, 0, err
	}

	return toProjectDeliveries(rows), total, nil
}

func (r *PostgresRepository) Transition(ctx context.Context, id uuid.UUID, status string, receivedBy *uuid.UUID) (Delivery, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)

		current, err := q.GetDelivery(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		var updateErr error
		if receivedBy != nil {
			_, updateErr = q.CompleteDeliveryStatus(ctx, sqlc.CompleteDeliveryStatusParams{
				ID:         id,
				Status:     sqlc.DeliveryStatus(status),
				ReceivedBy: toPgUUID(receivedBy),
			})
		} else {
			_, updateErr = q.UpdateDeliveryStatus(ctx, sqlc.UpdateDeliveryStatusParams{
				ID:     id,
				Status: sqlc.DeliveryStatus(status),
			})
		}
		if updateErr != nil {
			if errors.Is(updateErr, pgx.ErrNoRows) {
				return ErrInvalidTransition
			}
			return updateErr
		}

		lines, err := q.GetDeliveryLines(ctx, id)
		if err != nil {
			return err
		}

		partyType := string(current.SenderType)
		partyID := fromPgUUID(current.SenderID)
		if status == StatusDelivered {
			partyType = string(current.ReceiverType)
			partyID = fromPgUUID(current.ReceiverID)
		}
		return creditStock(ctx, q, partyType, partyID, lines)
	})
	if err != nil {
		return Delivery{}, err
	}

	return r.Get(ctx, id)
}

func creditStock(ctx context.Context, q *sqlc.Queries, partyType string, partyID *uuid.UUID, lines []sqlc.GetDeliveryLinesRow) error {
	for _, line := range lines {
		var err error
		if partyType == PartyCompany {
			_, err = q.AddCompanyStock(ctx, sqlc.AddCompanyStockParams{
				MaterialID: line.MaterialID,
				Quantity:   line.Quantity,
			})
		} else {
			_, err = q.AddProjectStock(ctx, sqlc.AddProjectStockParams{
				ProjectID:  *partyID,
				MaterialID: line.MaterialID,
				Quantity:   line.Quantity,
			})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func toCompanyDeliveries(rows []sqlc.ListCompanyDeliveriesRow) []Delivery {
	deliveries := make([]Delivery, 0, len(rows))
	for _, row := range rows {
		deliveries = append(deliveries, Delivery{
			ID:             row.ID,
			SenderType:     string(row.SenderType),
			SenderID:       fromPgUUID(row.SenderID),
			ReceiverType:   string(row.ReceiverType),
			ReceiverID:     fromPgUUID(row.ReceiverID),
			Status:         string(row.Status),
			SentAt:         row.SentAt,
			ReceivedAt:     fromPgTime(row.ReceivedAt),
			SentBy:         fromPgUUID(row.SentBy),
			ReceivedBy:     fromPgUUID(row.ReceivedBy),
			SentByName:     row.SentByName,
			ReceivedByName: row.ReceivedByName,
		})
	}
	return deliveries
}

func toProjectDeliveries(rows []sqlc.ListProjectDeliveriesRow) []Delivery {
	deliveries := make([]Delivery, 0, len(rows))
	for _, row := range rows {
		deliveries = append(deliveries, Delivery{
			ID:             row.ID,
			SenderType:     string(row.SenderType),
			SenderID:       fromPgUUID(row.SenderID),
			ReceiverType:   string(row.ReceiverType),
			ReceiverID:     fromPgUUID(row.ReceiverID),
			Status:         string(row.Status),
			SentAt:         row.SentAt,
			ReceivedAt:     fromPgTime(row.ReceivedAt),
			SentBy:         fromPgUUID(row.SentBy),
			ReceivedBy:     fromPgUUID(row.ReceivedBy),
			SentByName:     row.SentByName,
			ReceivedByName: row.ReceivedByName,
		})
	}
	return deliveries
}

func toDelivery(row sqlc.Delivery) Delivery {
	return Delivery{
		ID:           row.ID,
		SenderType:   string(row.SenderType),
		SenderID:     fromPgUUID(row.SenderID),
		ReceiverType: string(row.ReceiverType),
		ReceiverID:   fromPgUUID(row.ReceiverID),
		Status:       string(row.Status),
		SentAt:       row.SentAt,
		ReceivedAt:   fromPgTime(row.ReceivedAt),
		SentBy:       fromPgUUID(row.SentBy),
		ReceivedBy:   fromPgUUID(row.ReceivedBy),
	}
}

func toPgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	value := uuid.UUID(id.Bytes)
	return &value
}

func fromPgTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
