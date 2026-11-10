package companystore

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var ErrNotFound = errors.New("stock not found")

type Stock struct {
	MaterialID   uuid.UUID
	Name         string
	Category     string
	Unit         string
	Quantity     float64
	ReorderPoint float64
	Status       string
	InTransit    float64
}

type ListParams struct {
	Query    string
	Category string
	Status   string
	Sort     string
	Limit    int32
	Offset   int32
}

type Repository interface {
	List(ctx context.Context, p ListParams) ([]Stock, int64, error)
	Get(ctx context.Context, materialID uuid.UUID) (Stock, error)
	Add(ctx context.Context, materialID uuid.UUID, quantity float64) error
	SetReorderPoint(ctx context.Context, materialID uuid.UUID, reorderPoint float64) error
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) List(ctx context.Context, p ListParams) ([]Stock, int64, error) {
	total, err := r.queries.CountCompanyStock(ctx, sqlc.CountCompanyStockParams{
		Q: p.Query, Category: p.Category, Status: p.Status,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListCompanyStock(ctx, sqlc.ListCompanyStockParams{
		Q: p.Query, Category: p.Category, Status: p.Status, Sort: p.Sort,
		PageLimit: p.Limit, PageOffset: p.Offset,
	})
	if err != nil {
		return nil, 0, err
	}

	stock := make([]Stock, 0, len(rows))
	for _, row := range rows {
		stock = append(stock, Stock{
			MaterialID: row.MaterialID, Name: row.Name, Category: row.Category, Unit: row.Unit,
			Quantity: row.Quantity, ReorderPoint: row.ReorderPoint,
			Status: row.Status, InTransit: row.InTransit,
		})
	}
	return stock, total, nil
}

func (r *PostgresRepository) Get(ctx context.Context, materialID uuid.UUID) (Stock, error) {
	row, err := r.queries.GetCompanyStock(ctx, materialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, ErrNotFound
		}
		return Stock{}, err
	}
	return Stock{
		MaterialID: row.MaterialID, Name: row.Name, Category: row.Category, Unit: row.Unit,
		Quantity: row.Quantity, ReorderPoint: row.ReorderPoint,
		Status: row.Status, InTransit: row.InTransit,
	}, nil
}

func (r *PostgresRepository) Add(ctx context.Context, materialID uuid.UUID, quantity float64) error {
	_, err := r.queries.AddCompanyStock(ctx, sqlc.AddCompanyStockParams{
		MaterialID: materialID, Quantity: quantity,
	})
	return mapNotFound(err)
}

func (r *PostgresRepository) SetReorderPoint(ctx context.Context, materialID uuid.UUID, reorderPoint float64) error {
	_, err := r.queries.SetCompanyReorderPoint(ctx, sqlc.SetCompanyReorderPointParams{
		MaterialID: materialID, ReorderPoint: reorderPoint,
	})
	return mapNotFound(err)
}

func mapNotFound(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrNotFound
	}
	return err
}
