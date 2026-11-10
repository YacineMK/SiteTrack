package companystore

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidInput = errors.New("invalid input")

type Service interface {
	List(ctx context.Context, p ListParams) (ListStockResponse, error)
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

func (s *service) List(ctx context.Context, p ListParams) (ListStockResponse, error) {
	switch p.Sort {
	case "", "name", "quantity", "status":
	default:
		return ListStockResponse{}, ErrInvalidInput
	}
	switch p.Status {
	case "", "out_of_stock", "low", "available":
	default:
		return ListStockResponse{}, ErrInvalidInput
	}
	p.Query = strings.TrimSpace(p.Query)
	p.Category = strings.TrimSpace(p.Category)
	p.Limit, p.Offset = s.utils.NormalizePagination(p.Limit, p.Offset)

	stock, total, err := s.repo.List(ctx, p)
	if err != nil {
		return ListStockResponse{}, err
	}

	return s.utils.ListResponse(stock, total, p.Limit, p.Offset), nil
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
