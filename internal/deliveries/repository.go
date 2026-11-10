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
	ID                uuid.UUID
	SenderType        string
	SenderID          *uuid.UUID
	ReceiverType      string
	ReceiverID        *uuid.UUID
	Status            string
	SentAt            time.Time
	ReceivedAt        *time.Time
	SentBy            *uuid.UUID
	ReceivedBy        *uuid.UUID
	SenderDisplayID   string
	ReceiverDisplayID string
	SentByName        string
	ReceivedByName    string
	Lines             []Line
}

type ListParams struct {
	Status      string
	Direction   string
	FromDate    *time.Time
	ToDate      *time.Time
	ProjectID   *uuid.UUID
	AfterSentAt *time.Time
	AfterID     *uuid.UUID
	Limit       int32
}

type Repository interface {
	Create(ctx context.Context, delivery Delivery) (Delivery, error)
	Get(ctx context.Context, id uuid.UUID) (Delivery, error)
	ListCompany(ctx context.Context, p ListParams) ([]Delivery, error)
	ListProject(ctx context.Context, projectID uuid.UUID, p ListParams) ([]Delivery, error)
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
					MaterialID: line.MaterialID, Quantity: line.Quantity,
				})
			} else {
				rows, err = q.DeductProjectStock(ctx, sqlc.DeductProjectStockParams{
					ProjectID: *delivery.SenderID, MaterialID: line.MaterialID, Quantity: line.Quantity,
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
			SenderID:     pgUUID(delivery.SenderID),
			ReceiverType: sqlc.DeliveryParty(delivery.ReceiverType),
			ReceiverID:   pgUUID(delivery.ReceiverID),
			SentBy:       pgUUID(delivery.SentBy),
		}); err != nil {
			return err
		}

		for _, line := range delivery.Lines {
			lineID, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := q.CreateDeliveryLine(ctx, sqlc.CreateDeliveryLineParams{
				ID: lineID, DeliveryID: id, MaterialID: line.MaterialID, Quantity: line.Quantity,
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

	lines, err := r.queries.GetDeliveryLines(ctx, []uuid.UUID{id})
	if err != nil {
		return Delivery{}, err
	}

	delivery := Delivery{
		ID:           row.ID,
		SenderType:   string(row.SenderType),
		SenderID:     uuidPtr(row.SenderID),
		ReceiverType: string(row.ReceiverType),
		ReceiverID:   uuidPtr(row.ReceiverID),
		Status:       string(row.Status),
		SentAt:       row.SentAt,
		ReceivedAt:   timePtr(row.ReceivedAt),
		SentBy:       uuidPtr(row.SentBy),
		ReceivedBy:   uuidPtr(row.ReceivedBy),
		Lines:        make([]Line, 0, len(lines)),
	}
	for _, line := range lines {
		delivery.Lines = append(delivery.Lines, Line{
			MaterialID: line.MaterialID, Name: line.Name, Unit: line.Unit, Quantity: line.Quantity,
		})
	}
	return delivery, nil
}

func (r *PostgresRepository) ListCompany(ctx context.Context, p ListParams) ([]Delivery, error) {
	rows, err := r.queries.ListCompanyDeliveries(ctx, sqlc.ListCompanyDeliveriesParams{
		ProjectID:   pgUUID(p.ProjectID),
		Direction:   p.Direction,
		Status:      p.Status,
		FromDate:    pgDate(p.FromDate),
		ToDate:      pgDate(p.ToDate),
		AfterSentAt: pgTime(p.AfterSentAt),
		AfterID:     pgUUID(p.AfterID),
		PageLimit:   p.Limit,
	})
	if err != nil {
		return nil, err
	}

	deliveries := make([]Delivery, 0, len(rows))
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		deliveries = append(deliveries, Delivery{
			ID: row.ID, SenderType: string(row.SenderType), SenderID: uuidPtr(row.SenderID),
			ReceiverType: string(row.ReceiverType), ReceiverID: uuidPtr(row.ReceiverID),
			Status: string(row.Status), SentAt: row.SentAt, ReceivedAt: timePtr(row.ReceivedAt),
			SentBy: uuidPtr(row.SentBy), ReceivedBy: uuidPtr(row.ReceivedBy),
			SenderDisplayID: row.SenderDisplayID.String, ReceiverDisplayID: row.ReceiverDisplayID.String,
			SentByName: row.SentByName.String, ReceivedByName: row.ReceivedByName.String,
		})
		ids = append(ids, row.ID)
	}
	if err := r.attachLines(ctx, deliveries, ids); err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *PostgresRepository) ListProject(ctx context.Context, projectID uuid.UUID, p ListParams) ([]Delivery, error) {
	rows, err := r.queries.ListProjectDeliveries(ctx, sqlc.ListProjectDeliveriesParams{
		ProjectID:   projectID,
		Direction:   p.Direction,
		Status:      p.Status,
		FromDate:    pgDate(p.FromDate),
		ToDate:      pgDate(p.ToDate),
		AfterSentAt: pgTime(p.AfterSentAt),
		AfterID:     pgUUID(p.AfterID),
		PageLimit:   p.Limit,
	})
	if err != nil {
		return nil, err
	}

	deliveries := make([]Delivery, 0, len(rows))
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		deliveries = append(deliveries, Delivery{
			ID: row.ID, SenderType: string(row.SenderType), SenderID: uuidPtr(row.SenderID),
			ReceiverType: string(row.ReceiverType), ReceiverID: uuidPtr(row.ReceiverID),
			Status: string(row.Status), SentAt: row.SentAt, ReceivedAt: timePtr(row.ReceivedAt),
			SentBy: uuidPtr(row.SentBy), ReceivedBy: uuidPtr(row.ReceivedBy),
			SenderDisplayID: row.SenderDisplayID.String, ReceiverDisplayID: row.ReceiverDisplayID.String,
			SentByName: row.SentByName.String, ReceivedByName: row.ReceivedByName.String,
		})
		ids = append(ids, row.ID)
	}
	if err := r.attachLines(ctx, deliveries, ids); err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *PostgresRepository) attachLines(ctx context.Context, deliveries []Delivery, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.queries.GetDeliveryLines(ctx, ids)
	if err != nil {
		return err
	}
	byID := make(map[uuid.UUID][]Line, len(deliveries))
	for _, line := range rows {
		byID[line.DeliveryID] = append(byID[line.DeliveryID], Line{
			MaterialID: line.MaterialID, Name: line.Name, Unit: line.Unit, Quantity: line.Quantity,
		})
	}
	for i := range deliveries {
		deliveries[i].Lines = byID[deliveries[i].ID]
	}
	return nil
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
				ID: id, Status: sqlc.DeliveryStatus(status), ReceivedBy: pgUUID(receivedBy),
			})
		} else {
			_, updateErr = q.UpdateDeliveryStatus(ctx, sqlc.UpdateDeliveryStatusParams{
				ID: id, Status: sqlc.DeliveryStatus(status),
			})
		}
		if updateErr != nil {
			if errors.Is(updateErr, pgx.ErrNoRows) {
				return ErrInvalidTransition
			}
			return updateErr
		}

		lines, err := q.GetDeliveryLines(ctx, []uuid.UUID{id})
		if err != nil {
			return err
		}

		partyType := string(current.SenderType)
		partyID := uuidPtr(current.SenderID)
		if status == StatusDelivered {
			partyType = string(current.ReceiverType)
			partyID = uuidPtr(current.ReceiverID)
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
				MaterialID: line.MaterialID, Quantity: line.Quantity,
			})
		} else {
			_, err = q.AddProjectStock(ctx, sqlc.AddProjectStockParams{
				ProjectID: *partyID, MaterialID: line.MaterialID, Quantity: line.Quantity,
			})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func pgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func uuidPtr(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	value := uuid.UUID(id.Bytes)
	return &value
}

func pgTime(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func pgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}
