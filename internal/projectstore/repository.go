package projectstore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var ErrNotFound = errors.New("stock not found")

type Stock struct {
	MaterialID   uuid.UUID
	Name         string
	Category     string
	Unit         string
	Quantity     float64
	LastDelivery *time.Time
}

type ListParams struct {
	ProjectID uuid.UUID
	Query     string
	Category  string
	Sort      string
	Limit     int32
	Offset    int32
}

type Repository interface {
	List(ctx context.Context, p ListParams) ([]Stock, int64, error)
	Get(ctx context.Context, projectID, materialID uuid.UUID) (Stock, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) List(ctx context.Context, p ListParams) ([]Stock, int64, error) {
	total, err := r.queries.CountProjectStock(ctx, sqlc.CountProjectStockParams{
		ProjectID: p.ProjectID, Q: p.Query, Category: p.Category,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListProjectStock(ctx, sqlc.ListProjectStockParams{
		ProjectID: p.ProjectID, Q: p.Query, Category: p.Category, Sort: p.Sort,
		PageLimit: p.Limit, PageOffset: p.Offset,
	})
	if err != nil {
		return nil, 0, err
	}

	stock := make([]Stock, 0, len(rows))
	for _, row := range rows {
		item := Stock{
			MaterialID: row.MaterialID,
			Name:       row.Name, Category: row.Category, Unit: row.Unit,
			Quantity: row.Quantity,
		}
		if !row.LastDelivery.IsZero() {
			last := row.LastDelivery
			item.LastDelivery = &last
		}
		stock = append(stock, item)
	}
	return stock, total, nil
}

func (r *PostgresRepository) Get(ctx context.Context, projectID, materialID uuid.UUID) (Stock, error) {
	row, err := r.queries.GetProjectStock(ctx, sqlc.GetProjectStockParams{
		ProjectID:  projectID,
		MaterialID: materialID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, ErrNotFound
		}
		return Stock{}, err
	}
	return Stock{
		MaterialID: row.MaterialID,
		Name:       row.Name, Category: row.Category, Unit: row.Unit,
		Quantity: row.Quantity,
	}, nil
}
