package materials

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var (
	ErrNotFound  = errors.New("material not found")
	ErrDuplicate = errors.New("material already exists")
	ErrInUse     = errors.New("material is used by deliveries")
)

type Material struct {
	ID        uuid.UUID
	Name      string
	Category  string
	Unit      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, material Material) (Material, error)
	Get(ctx context.Context, id uuid.UUID) (Material, error)
	List(ctx context.Context, query string, limit, offset int32) ([]Material, int64, error)
	Update(ctx context.Context, id uuid.UUID, name, category, unit string) (Material, error)
	Delete(ctx context.Context, id uuid.UUID) (int64, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) Create(ctx context.Context, material Material) (Material, error) {
	row, err := r.queries.CreateMaterial(ctx, sqlc.CreateMaterialParams{
		ID:       material.ID,
		Name:     material.Name,
		Category: material.Category,
		Unit:     material.Unit,
	})
	if err != nil {
		return Material{}, mapDuplicate(err)
	}
	return toMaterial(row), nil
}

func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (Material, error) {
	row, err := r.queries.GetMaterial(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Material{}, ErrNotFound
		}
		return Material{}, err
	}
	return toMaterial(row), nil
}

func (r *PostgresRepository) List(ctx context.Context, query string, limit, offset int32) ([]Material, int64, error) {
	total, err := r.queries.CountMaterials(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListMaterials(ctx, sqlc.ListMaterialsParams{
		Q:          query,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	materials := make([]Material, 0, len(rows))
	for _, row := range rows {
		materials = append(materials, toMaterial(row))
	}
	return materials, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id uuid.UUID, name, category, unit string) (Material, error) {
	row, err := r.queries.UpdateMaterial(ctx, sqlc.UpdateMaterialParams{
		ID:       id,
		Name:     name,
		Category: category,
		Unit:     unit,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Material{}, ErrNotFound
		}
		return Material{}, mapDuplicate(err)
	}
	return toMaterial(row), nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	rows, err := r.queries.DeleteMaterial(ctx, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, ErrInUse
		}
		return 0, err
	}
	return rows, nil
}

func mapDuplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

func toMaterial(row sqlc.Material) Material {
	return Material{
		ID:        row.ID,
		Name:      row.Name,
		Category:  row.Category,
		Unit:      row.Unit,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
