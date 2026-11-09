package projects

import (
	"context"
	"errors"
	"strings"

	ids "github.com/YacineMK/SiteTrack/kit/idgen"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	Create(ctx context.Context, req CreateProjectRequest) (ProjectResponse, error)
	Get(ctx context.Context, displayID string) (ProjectResponse, error)
	List(ctx context.Context, limit, offset int32) (ListProjectsResponse, error)
	Search(ctx context.Context, query string, limit, offset int32) (ListProjectsResponse, error)
	Update(ctx context.Context, displayID string, req UpdateProjectRequest) (ProjectResponse, error)
	Delete(ctx context.Context, displayID string) error
}

type service struct {
	repo  Repository
	ids   *ids.Generator
	utils Utils
}

func NewService(repo Repository, generator *ids.Generator) Service {
	return &service{repo: repo, ids: generator, utils: NewUtils()}
}

func (s *service) Create(ctx context.Context, req CreateProjectRequest) (ProjectResponse, error) {
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
	})
	if err != nil {
		return ProjectResponse{}, err
	}

	return s.utils.ToResponse(project), nil
}

func (s *service) Get(ctx context.Context, displayID string) (ProjectResponse, error) {
	project, err := s.repo.GetByDisplayID(ctx, displayID)
	if err != nil {
		return ProjectResponse{}, err
	}
	return s.utils.ToResponse(project), nil
}

func (s *service) List(ctx context.Context, limit, offset int32) (ListProjectsResponse, error) {
	limit, offset = s.utils.NormalizePagination(limit, offset)

	items, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return ListProjectsResponse{}, err
	}

	return s.utils.ListResponse(items, total, limit, offset), nil
}

func (s *service) Search(ctx context.Context, query string, limit, offset int32) (ListProjectsResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return ListProjectsResponse{}, ErrInvalidInput
	}
	limit, offset = s.utils.NormalizePagination(limit, offset)

	items, total, err := s.repo.Search(ctx, query, limit, offset)
	if err != nil {
		return ListProjectsResponse{}, err
	}

	return s.utils.ListResponse(items, total, limit, offset), nil
}

func (s *service) Update(ctx context.Context, displayID string, req UpdateProjectRequest) (ProjectResponse, error) {
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
