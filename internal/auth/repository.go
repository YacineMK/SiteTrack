package auth

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
	ErrNotFound    = errors.New("user not found")
	ErrEmailExists = errors.New("email already exists")
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	AppRole      string
	TokenVersion int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	Create(ctx context.Context, user User) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(db)}
}

func (r *PostgresRepository) Create(ctx context.Context, user User) (User, error) {
	row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		AppRole:      user.AppRole,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailExists
		}
		return User{}, err
	}
	return toUser(row.ID, row.Name, row.Email, row.PasswordHash, row.AppRole, row.TokenVersion, row.CreatedAt, row.UpdatedAt), nil
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return toUser(row.ID, row.Name, row.Email, row.PasswordHash, row.AppRole, row.TokenVersion, row.CreatedAt, row.UpdatedAt), nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return toUser(row.ID, row.Name, row.Email, row.PasswordHash, row.AppRole, row.TokenVersion, row.CreatedAt, row.UpdatedAt), nil
}

func (r *PostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.queries.EmailExists(ctx, email)
}

func toUser(id uuid.UUID, name, email, passwordHash, appRole string, tokenVersion int32, createdAt, updatedAt time.Time) User {
	return User{
		ID:           id,
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		AppRole:      appRole,
		TokenVersion: tokenVersion,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
