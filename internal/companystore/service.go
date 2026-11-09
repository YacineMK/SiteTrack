package companystore

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	List(ctx context.Context, query string, limit, offset int32) (ListStockResponse, error)
	Get(ctx context.Context, materialID string) (StockResponse, error)
	Add(ctx context.Context, req AddStockRequest) (StockResponse, error)
	SetReorderPoint(ctx context.Context, materialID string, req SetReorderPointRequest) (StockResponse, error)
}

type service struct {
	repo  Repository
	utils Utils
}

func NewService(repo Repository) Service {
	return &service{repo: repo, utils: NewUtils()}
}

func (s *service) List(ctx context.Context, query string, limit, offset int32) (ListStockResponse, error) {
	limit, offset = s.utils.NormalizePagination(limit, offset)

	stock, total, err := s.repo.List(ctx, strings.TrimSpace(query), limit, offset)
	if err != nil {
		return ListStockResponse{}, err
	}

	return s.utils.ListResponse(stock, total, limit, offset), nil
}

func (s *service) Get(ctx context.Context, materialID string) (StockResponse, error) {
	id, err := s.utils.ParseID(materialID)
	if err != nil {
		return StockResponse{}, err
	}

	stock, err := s.repo.Get(ctx, id)
	if err != nil {
		return StockResponse{}, err
	}

	return s.utils.ToResponse(stock), nil
}

func (s *service) Add(ctx context.Context, req AddStockRequest) (StockResponse, error) {
	if err := req.Validate(); err != nil {
		return StockResponse{}, ErrInvalidInput
	}

	id, err := s.utils.ParseID(req.MaterialID)
	if err != nil {
		return StockResponse{}, err
	}

	if err := s.repo.Add(ctx, id, req.Quantity); err != nil {
		return StockResponse{}, err
	}

	return s.Get(ctx, req.MaterialID)
}

func (s *service) SetReorderPoint(ctx context.Context, materialID string, req SetReorderPointRequest) (StockResponse, error) {
	if err := req.Validate(); err != nil {
		return StockResponse{}, ErrInvalidInput
	}

	id, err := s.utils.ParseID(materialID)
	if err != nil {
		return StockResponse{}, err
	}

	if err := s.repo.SetReorderPoint(ctx, id, req.ReorderPoint); err != nil {
		return StockResponse{}, err
	}

	return s.Get(ctx, materialID)
}
