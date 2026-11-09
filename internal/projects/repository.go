package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

var (
	ErrNotFound        = errors.New("project not found")
	ErrMemberNotFound  = errors.New("project member not found")
	ErrLastAdmin       = errors.New("project must keep at least one admin")
	ErrInviteExists    = errors.New("invitation already exists")
	ErrInviteNotFound  = errors.New("invitation not found")
	ErrInviteExpired   = errors.New("invitation expired")
	ErrInviteeNotFound = errors.New("user not found")
	ErrAlreadyMember   = errors.New("user is already a member")
)

type Project struct {
	ID               uuid.UUID
	DisplayID        string
	Name             string
	Description      string
	PlannedStartDate time.Time
	PlannedEndDate   time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	IsAdmin          bool
	MemberCount      int64
	NotStarted       int64
	InProgress       int64
	Done             int64
	LateCount        int64
	Progress         float64
	BlockedCount     int64
}

type Member struct {
	UserID    uuid.UUID
	Name      string
	Email     string
	IsAdmin   bool
	CreatedAt time.Time
}

type Invitee struct {
	ID    uuid.UUID
	Name  string
	Email string
}

type Invitation struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Title     string
	Body      string
	Data      []byte
	ReadAt    *time.Time
	CreatedAt time.Time
}

type InvitationNotification struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Type   string
	Title  string
	Body   string
	Data   []byte
}

type OutboxEvent struct {
	ID      uuid.UUID
	Kind    string
	Payload []byte
}

type Repository interface {
	Create(ctx context.Context, project Project, adminID uuid.UUID) (Project, error)
	GetByDisplayID(ctx context.Context, displayID string) (Project, error)
	List(ctx context.Context, userID uuid.UUID, superAdmin, lateOnly bool, sortBy, sortDir string, limit int32, after string) ([]Project, int64, error)
	Search(ctx context.Context, userID uuid.UUID, superAdmin, lateOnly bool, query string, limit int32, after string) ([]Project, int64, error)
	Update(ctx context.Context, displayID, name, description string, start, end time.Time) (Project, error)
	Delete(ctx context.Context, displayID string) (int64, error)
	ProjectAccess(ctx context.Context, displayID string, userID uuid.UUID) (uuid.UUID, bool, bool, bool, error)
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]Member, error)
	SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, isAdmin bool) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	Membership(ctx context.Context, projectID, userID uuid.UUID) (bool, bool, error)
	FindUserByEmail(ctx context.Context, email string) (Invitee, error)
	CreateInvitationNotification(ctx context.Context, notification InvitationNotification, events []OutboxEvent) error
	GetInvitationNotification(ctx context.Context, id uuid.UUID) (Invitation, error)
	LockInvitationNotification(ctx context.Context, id uuid.UUID) (Invitation, error)
	DeleteInvitationNotification(ctx context.Context, id uuid.UUID) (int64, error)
	MarkInvitationRead(ctx context.Context, id, userID uuid.UUID) (int64, error)
	AcceptInvitation(ctx context.Context, id, projectID, userID uuid.UUID, isAdmin bool) error
	RejectInvitation(ctx context.Context, id, userID uuid.UUID) (int64, error)
}

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, queries: sqlc.New(pool)}
}

func (r *PostgresRepository) Create(ctx context.Context, project Project, adminID uuid.UUID) (Project, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		row, err := queries.CreateProject(ctx, sqlc.CreateProjectParams{
			ID:               project.ID,
			DisplayID:        project.DisplayID,
			Name:             project.Name,
			Description:      project.Description,
			PlannedStartDate: project.PlannedStartDate,
			PlannedEndDate:   project.PlannedEndDate,
		})
		if err != nil {
			return err
		}

		project = toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt,
			row.MemberCount, row.TaskCountNotStarted, row.TaskCountInProgress, row.TaskCountDone, row.LateCount, row.BlockedCount, row.Progress.Float64, true)
		return queries.AddProjectMember(ctx, sqlc.AddProjectMemberParams{
			ProjectID: project.ID,
			UserID:    adminID,
			IsAdmin:   true,
		})
	})
	if err != nil {
		return Project{}, err
	}
	return project, nil
}

func (r *PostgresRepository) GetByDisplayID(ctx context.Context, displayID string) (Project, error) {
	row, err := r.queries.GetProjectByDisplayID(ctx, displayID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt,
		row.MemberCount, row.TaskCountNotStarted, row.TaskCountInProgress, row.TaskCountDone, row.LateCount, row.BlockedCount, row.Progress.Float64, false), nil
}

func (r *PostgresRepository) List(ctx context.Context, userID uuid.UUID, superAdmin, lateOnly bool, sortBy, sortDir string, limit int32, after string) ([]Project, int64, error) {
	total, err := r.queries.CountProjects(ctx, sqlc.CountProjectsParams{
		UserID:     userID,
		SuperAdmin: superAdmin,
		LateOnly:   lateOnly,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListProjects(ctx, sqlc.ListProjectsParams{
		UserID:     userID,
		SuperAdmin: superAdmin,
		LateOnly:   lateOnly,
		AfterID:    after,
		SortBy:     sortBy,
		SortDir:    sortDir,
		PageLimit:  limit,
	})
	if err != nil {
		return nil, 0, err
	}

	projects := make([]Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toListProject(row))
	}
	return projects, total, nil
}

func (r *PostgresRepository) Search(ctx context.Context, userID uuid.UUID, superAdmin, lateOnly bool, query string, limit int32, after string) ([]Project, int64, error) {
	total, err := r.queries.CountSearchProjects(ctx, sqlc.CountSearchProjectsParams{
		Query:      query,
		UserID:     userID,
		SuperAdmin: superAdmin,
		LateOnly:   lateOnly,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.SearchProjects(ctx, sqlc.SearchProjectsParams{
		Query:      query,
		UserID:     userID,
		SuperAdmin: superAdmin,
		LateOnly:   lateOnly,
		AfterID:    after,
		PageLimit:  limit,
	})
	if err != nil {
		return nil, 0, err
	}

	projects := make([]Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toSearchProject(row))
	}
	return projects, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, displayID, name, description string, start, end time.Time) (Project, error) {
	row, err := r.queries.UpdateProject(ctx, sqlc.UpdateProjectParams{
		DisplayID:        displayID,
		Name:             name,
		Description:      description,
		PlannedStartDate: start,
		PlannedEndDate:   end,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt,
		row.MemberCount, row.TaskCountNotStarted, row.TaskCountInProgress, row.TaskCountDone, row.LateCount, row.BlockedCount, row.Progress.Float64, false), nil
}

func (r *PostgresRepository) Delete(ctx context.Context, displayID string) (int64, error) {
	return r.queries.DeleteProject(ctx, displayID)
}

func (r *PostgresRepository) ProjectAccess(ctx context.Context, displayID string, userID uuid.UUID) (uuid.UUID, bool, bool, bool, error) {
	project, err := r.GetByDisplayID(ctx, displayID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return uuid.UUID{}, false, false, false, nil
		}
		return uuid.UUID{}, false, false, false, err
	}

	isAdmin, err := r.queries.GetProjectMembership(ctx, sqlc.GetProjectMembershipParams{
		ProjectID: project.ID,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return project.ID, true, false, false, nil
		}
		return uuid.UUID{}, false, false, false, err
	}
	return project.ID, true, true, isAdmin, nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, projectID uuid.UUID) ([]Member, error) {
	rows, err := r.queries.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	members := make([]Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, Member{
			UserID:    row.UserID,
			Name:      row.Name,
			Email:     row.Email,
			IsAdmin:   row.IsAdmin,
			CreatedAt: row.CreatedAt,
		})
	}
	return members, nil
}

func (r *PostgresRepository) SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, isAdmin bool) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		if !isAdmin {
			if err := r.ensureAnotherAdmin(ctx, queries, projectID, userID); err != nil {
				return err
			}
		}

		rows, err := queries.SetProjectMemberAdmin(ctx, sqlc.SetProjectMemberAdminParams{
			ProjectID: projectID,
			UserID:    userID,
			IsAdmin:   isAdmin,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrMemberNotFound
		}
		return nil
	})
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		if err := r.ensureAnotherAdmin(ctx, queries, projectID, userID); err != nil {
			return err
		}

		rows, err := queries.RemoveProjectMember(ctx, sqlc.RemoveProjectMemberParams{
			ProjectID: projectID,
			UserID:    userID,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrMemberNotFound
		}
		return nil
	})
}

func (r *PostgresRepository) ensureAnotherAdmin(ctx context.Context, queries *sqlc.Queries, projectID, userID uuid.UUID) error {
	current, err := queries.GetProjectMembership(ctx, sqlc.GetProjectMembershipParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMemberNotFound
		}
		return err
	}
	if !current {
		return nil
	}

	admins, err := queries.LockProjectAdmins(ctx, projectID)
	if err != nil {
		return err
	}
	if len(admins) <= 1 {
		return ErrLastAdmin
	}
	return nil
}

func (r *PostgresRepository) Membership(ctx context.Context, projectID, userID uuid.UUID) (bool, bool, error) {
	isAdmin, err := r.queries.GetProjectMembership(ctx, sqlc.GetProjectMembershipParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, isAdmin, nil
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (Invitee, error) {
	row, err := r.queries.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Invitee{}, ErrInviteeNotFound
		}
		return Invitee{}, err
	}
	return Invitee{ID: row.ID, Name: row.Name, Email: row.Email}, nil
}

func (r *PostgresRepository) CreateInvitationNotification(ctx context.Context, notification InvitationNotification, events []OutboxEvent) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		if _, err := queries.CreateNotification(ctx, sqlc.CreateNotificationParams{
			ID:     notification.ID,
			UserID: notification.UserID,
			Type:   notification.Type,
			Title:  notification.Title,
			Body:   notification.Body,
			Data:   notification.Data,
		}); err != nil {
			return err
		}

		for _, event := range events {
			if err := queries.CreateOutboxEvent(ctx, sqlc.CreateOutboxEventParams{
				ID:      event.ID,
				Kind:    event.Kind,
				Payload: event.Payload,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostgresRepository) GetInvitationNotification(ctx context.Context, id uuid.UUID) (Invitation, error) {
	row, err := r.queries.GetNotification(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Invitation{}, ErrInviteNotFound
		}
		return Invitation{}, err
	}
	return toInvitation(row), nil
}

func (r *PostgresRepository) LockInvitationNotification(ctx context.Context, id uuid.UUID) (Invitation, error) {
	row, err := r.queries.LockNotification(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Invitation{}, ErrInviteNotFound
		}
		return Invitation{}, err
	}
	return toInvitation(row), nil
}

func (r *PostgresRepository) DeleteInvitationNotification(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queries.DeleteNotification(ctx, id)
}

func (r *PostgresRepository) MarkInvitationRead(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	_, err := r.queries.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return 1, nil
}

func (r *PostgresRepository) AcceptInvitation(ctx context.Context, id, projectID, userID uuid.UUID, isAdmin bool) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)

		notification, err := queries.LockNotification(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInviteNotFound
			}
			return err
		}
		if notification.ReadAt.Valid {
			return ErrInviteNotFound
		}

		if err := queries.AddProjectMemberIfAbsent(ctx, sqlc.AddProjectMemberIfAbsentParams{
			ProjectID: projectID,
			UserID:    userID,
			IsAdmin:   isAdmin,
		}); err != nil {
			return err
		}

		if _, err := queries.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{
			ID:     id,
			UserID: userID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInviteNotFound
			}
			return err
		}
		return nil
	})
}

func (r *PostgresRepository) RejectInvitation(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	return r.MarkInvitationRead(ctx, id, userID)
}

func toInvitation(row sqlc.Notification) Invitation {
	var readAt *time.Time
	if row.ReadAt.Valid {
		value := row.ReadAt.Time
		readAt = &value
	}
	return Invitation{
		ID:        row.ID,
		UserID:    row.UserID,
		Type:      row.Type,
		Title:     row.Title,
		Body:      row.Body,
		Data:      row.Data,
		ReadAt:    readAt,
		CreatedAt: row.CreatedAt,
	}
}

func toProject(id uuid.UUID, displayID, name, description string, start, end, createdAt, updatedAt time.Time, memberCount, notStarted, inProgress, done, late, blocked int32, progress float64, isAdmin bool) Project {
	return Project{
		ID:               id,
		DisplayID:        displayID,
		Name:             name,
		Description:      description,
		PlannedStartDate: start,
		PlannedEndDate:   end,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		IsAdmin:          isAdmin,
		MemberCount:      int64(memberCount),
		NotStarted:       int64(notStarted),
		InProgress:       int64(inProgress),
		Done:             int64(done),
		LateCount:        int64(late),
		BlockedCount:     int64(blocked),
		Progress:         progress,
	}
}

func toListProject(row sqlc.ListProjectsRow) Project {
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt,
		row.MemberCount, row.NotStarted, row.InProgress, row.Done, row.Late, row.BlockedCount, row.Progress.Float64, row.IsAdmin)
}

func toSearchProject(row sqlc.SearchProjectsRow) Project {
	return toProject(row.ID, row.DisplayID, row.Name, row.Description, row.PlannedStartDate, row.PlannedEndDate, row.CreatedAt, row.UpdatedAt,
		row.MemberCount, row.NotStarted, row.InProgress, row.Done, row.Late, row.BlockedCount, row.Progress.Float64, row.IsAdmin)
}
