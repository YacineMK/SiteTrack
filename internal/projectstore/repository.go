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

type Repository interface {
	List(ctx context.Context, projectID uuid.UUID, query, category, sortBy, sortDir string, limit, offset int32) ([]Stock, int64, error)
	Get(ctx context.Context, projectID, materialID uuid.UUID) (Stock, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) List(ctx context.Context, projectID uuid.UUID, query, category, sortBy, sortDir string, limit, offset int32) ([]Stock, int64, error) {
	total, err := r.queries.CountProjectStock(ctx, sqlc.CountProjectStockParams{
		ProjectID: projectID,
		Query:     query,
		Category:  category,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListProjectStock(ctx, sqlc.ListProjectStockParams{
		ProjectID:  projectID,
		Query:      query,
		Category:   category,
		SortBy:     sortBy,
		SortDir:    sortDir,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	stock := make([]Stock, 0, len(rows))
	for _, row := range rows {
		item := Stock{
			MaterialID: row.MaterialID,
			Name:       row.Name,
			Category:   row.Category,
			Unit:       row.Unit,
			Quantity:   row.Quantity,
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
		Name:       row.Name,
		Category:   row.Category,
		Unit:       row.Unit,
		Quantity:   row.Quantity,
	}, nil
}
