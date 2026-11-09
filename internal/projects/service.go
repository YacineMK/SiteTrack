package projects

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	ids "github.com/YacineMK/SiteTrack/kit/idgen"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	Create(ctx context.Context, userID uuid.UUID, req CreateProjectRequest) (ProjectResponse, error)
	Get(ctx context.Context, displayID string, isAdmin bool) (ProjectResponse, error)
	List(ctx context.Context, userID uuid.UUID, superAdmin bool, limit, offset int32) (ListProjectsResponse, error)
	Search(ctx context.Context, userID uuid.UUID, superAdmin bool, query string, limit, offset int32) (ListProjectsResponse, error)
	Update(ctx context.Context, displayID string, isAdmin bool, req UpdateProjectRequest) (ProjectResponse, error)
	Delete(ctx context.Context, displayID string) error
	ListMembers(ctx context.Context, projectID uuid.UUID) (ListMembersResponse, error)
	SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, req UpdateMemberRequest) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
}

type service struct {
	repo  Repository
	ids   *ids.Generator
	utils Utils
}

func NewService(repo Repository, generator *ids.Generator) Service {
	return &service{repo: repo, ids: generator, utils: NewUtils()}
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, req CreateProjectRequest) (ProjectResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if err := req.Validate(); err != nil {
		return ProjectResponse{}, ErrInvalidInput
	}

	start, end, err := s.utils.ParseDates(req.PlannedStartDate, req.PlannedEndDate)
	if err != nil {
		return ProjectResponse{}, err
	}

	generated, err := s.ids.NewProject(ctx)
	if err != nil {
		return ProjectResponse{}, err
	}

	project, err := s.repo.Create(ctx, Project{
		ID:               generated.UUID,
		DisplayID:        generated.DisplayID,
		Name:             req.Name,
		Description:      req.Description,
		PlannedStartDate: start,
		PlannedEndDate:   end,
	}, userID)
	if err != nil {
		return ProjectResponse{}, err
	}

	return s.utils.ToResponse(project), nil
}

func (s *service) Get(ctx context.Context, displayID string, isAdmin bool) (ProjectResponse, error) {
	project, err := s.repo.GetByDisplayID(ctx, displayID)
	if err != nil {
		return ProjectResponse{}, err
	}
	project.IsAdmin = isAdmin
	return s.utils.ToResponse(project), nil
}

func (s *service) List(ctx context.Context, userID uuid.UUID, superAdmin bool, limit, offset int32) (ListProjectsResponse, error) {
	limit, offset = s.utils.NormalizePagination(limit, offset)

	items, total, err := s.repo.List(ctx, userID, superAdmin, limit, offset)
	if err != nil {
		return ListProjectsResponse{}, err
	}

	return s.utils.ListResponse(items, total, limit, offset), nil
}

func (s *service) Search(ctx context.Context, userID uuid.UUID, superAdmin bool, query string, limit, offset int32) (ListProjectsResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return ListProjectsResponse{}, ErrInvalidInput
	}
	limit, offset = s.utils.NormalizePagination(limit, offset)

	items, total, err := s.repo.Search(ctx, userID, superAdmin, query, limit, offset)
	if err != nil {
		return ListProjectsResponse{}, err
	}

	return s.utils.ListResponse(items, total, limit, offset), nil
}

func (s *service) Update(ctx context.Context, displayID string, isAdmin bool, req UpdateProjectRequest) (ProjectResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if err := req.Validate(); err != nil {
		return ProjectResponse{}, ErrInvalidInput
	}

	start, end, err := s.utils.ParseDates(req.PlannedStartDate, req.PlannedEndDate)
	if err != nil {
		return ProjectResponse{}, err
	}

	project, err := s.repo.Update(ctx, displayID, req.Name, req.Description, start, end)
	if err != nil {
		return ProjectResponse{}, err
	}
	project.IsAdmin = isAdmin

	return s.utils.ToResponse(project), nil
}

func (s *service) Delete(ctx context.Context, displayID string) error {
	rows, err := s.repo.Delete(ctx, displayID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *service) ListMembers(ctx context.Context, projectID uuid.UUID) (ListMembersResponse, error) {
	members, err := s.repo.ListMembers(ctx, projectID)
	if err != nil {
		return ListMembersResponse{}, err
	}
	return s.utils.MembersResponse(members), nil
}

func (s *service) SetMemberAdmin(ctx context.Context, projectID, userID uuid.UUID, req UpdateMemberRequest) error {
	if err := req.Validate(); err != nil {
		return ErrInvalidInput
	}
	return s.repo.SetMemberAdmin(ctx, projectID, userID, *req.IsAdmin)
}

func (s *service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return s.repo.RemoveMember(ctx, projectID, userID)
}

func parseDates(start, end string) (time.Time, time.Time, error) {
	startDate, err := time.Parse(dateLayout, start)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}

	endDate, err := time.Parse(dateLayout, end)
	if err != nil || endDate.Before(startDate) {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}

	return startDate, endDate, nil
}
