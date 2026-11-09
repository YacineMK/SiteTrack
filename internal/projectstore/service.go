package projectstore

import (
	"context"
	"errors"
	"strings"

	"github.com/YacineMK/SiteTrack/internal/projects"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	List(ctx context.Context, projectDisplayID, query, category, sortBy, sortDir string, limit, offset int32) (ListProjectStockResponse, error)
	Get(ctx context.Context, projectDisplayID, materialID string) (ProjectStockResponse, error)
}

type service struct {
	repo     Repository
	projects projects.Repository
	utils    Utils
}

func NewService(repo Repository, projectRepo projects.Repository) Service {
	return &service{repo: repo, projects: projectRepo, utils: NewUtils()}
}

func (s *service) List(ctx context.Context, projectDisplayID, query, category, sortBy, sortDir string, limit, offset int32) (ListProjectStockResponse, error) {
	switch sortBy {
	case "", "name", "quantity", "last_delivery":
	default:
		return ListProjectStockResponse{}, ErrInvalidInput
	}
	switch sortDir {
	case "", "asc", "desc":
	default:
		return ListProjectStockResponse{}, ErrInvalidInput
	}

	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return ListProjectStockResponse{}, err
	}

	limit, offset = s.utils.NormalizePagination(limit, offset)

	stock, total, err := s.repo.List(ctx, project.ID, strings.TrimSpace(query), strings.TrimSpace(category), sortBy, sortDir, limit, offset)
	if err != nil {
		return ListProjectStockResponse{}, err
	}

	return s.utils.ListResponse(stock, total, limit, offset), nil
}

func (s *service) Get(ctx context.Context, projectDisplayID, materialID string) (ProjectStockResponse, error) {
	project, err := s.projects.GetByDisplayID(ctx, projectDisplayID)
	if err != nil {
		return ProjectStockResponse{}, err
	}

	id, err := s.utils.ParseID(materialID)
	if err != nil {
		return ProjectStockResponse{}, err
	}

	stock, err := s.repo.Get(ctx, project.ID, id)
	if err != nil {
		return ProjectStockResponse{}, err
	}

	return s.utils.ToResponse(stock), nil
}
