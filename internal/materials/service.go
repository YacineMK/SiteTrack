package materials

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	Create(ctx context.Context, req CreateMaterialRequest) (MaterialResponse, error)
	Get(ctx context.Context, id string) (MaterialResponse, error)
	List(ctx context.Context, query string, limit, offset int32) (ListMaterialsResponse, error)
	Update(ctx context.Context, id string, req UpdateMaterialRequest) (MaterialResponse, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo  Repository
	utils Utils
}

func NewService(repo Repository) Service {
	return &service{repo: repo, utils: NewUtils()}
}

func (s *service) Create(ctx context.Context, req CreateMaterialRequest) (MaterialResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	req.Unit = strings.TrimSpace(req.Unit)
	if err := req.Validate(); err != nil {
		return MaterialResponse{}, ErrInvalidInput
	}

	id, err := uuid.NewV7()
	if err != nil {
		return MaterialResponse{}, err
	}

	material, err := s.repo.Create(ctx, Material{
		ID:       id,
		Name:     req.Name,
		Category: req.Category,
		Unit:     req.Unit,
	})
	if err != nil {
		return MaterialResponse{}, err
	}

	return s.utils.ToResponse(material), nil
}

func (s *service) Get(ctx context.Context, id string) (MaterialResponse, error) {
	materialID, err := s.utils.ParseID(id)
	if err != nil {
		return MaterialResponse{}, err
	}

	material, err := s.repo.Get(ctx, materialID)
	if err != nil {
		return MaterialResponse{}, err
	}

	return s.utils.ToResponse(material), nil
}

func (s *service) List(ctx context.Context, query string, limit, offset int32) (ListMaterialsResponse, error) {
	limit, offset = s.utils.NormalizePagination(limit, offset)

	materials, total, err := s.repo.List(ctx, strings.TrimSpace(query), limit, offset)
	if err != nil {
		return ListMaterialsResponse{}, err
	}

	return s.utils.ListResponse(materials, total, limit, offset), nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateMaterialRequest) (MaterialResponse, error) {
	materialID, err := s.utils.ParseID(id)
	if err != nil {
		return MaterialResponse{}, err
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
	}
	if req.Category != nil {
		trimmed := strings.TrimSpace(*req.Category)
		req.Category = &trimmed
	}
	if req.Unit != nil {
		trimmed := strings.TrimSpace(*req.Unit)
		req.Unit = &trimmed
	}
	if err := req.Validate(); err != nil {
		return MaterialResponse{}, ErrInvalidInput
	}

	existing, err := s.repo.Get(ctx, materialID)
	if err != nil {
		return MaterialResponse{}, err
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	category := existing.Category
	if req.Category != nil {
		category = *req.Category
	}
	unit := existing.Unit
	if req.Unit != nil {
		unit = *req.Unit
	}

	material, err := s.repo.Update(ctx, materialID, name, category, unit)
	if err != nil {
		return MaterialResponse{}, err
	}

	return s.utils.ToResponse(material), nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	materialID, err := s.utils.ParseID(id)
	if err != nil {
		return err
	}

	rows, err := s.repo.Delete(ctx, materialID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
