package companystore

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToResponse(stock Stock) StockResponse
	ListResponse(stock []Stock, total int64, limit, offset int32) ListStockResponse
	WriteServiceError(w http.ResponseWriter, err error)
	ParseID(value string) (uuid.UUID, error)
	NormalizePagination(limit, offset int32) (int32, int32)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (utils) ToResponse(stock Stock) StockResponse {
	return StockResponse{
		MaterialID:   stock.MaterialID,
		Name:         stock.Name,
		Category:     stock.Category,
		Unit:         stock.Unit,
		Quantity:     stock.Quantity,
		ReorderPoint: stock.ReorderPoint,
		InTransit:    stock.InTransit,
		Status:       stock.Status,
	}
}

func (u utils) ListResponse(stock []Stock, total int64, limit, offset int32) ListStockResponse {
	items := make([]StockResponse, 0, len(stock))
	for _, item := range stock {
		items = append(items, u.ToResponse(item))
	}
	return ListStockResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

func (utils) WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		jsonutil.Error(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrNotFound):
		jsonutil.Error(w, http.StatusNotFound, "material not found")
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (utils) ParseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, ErrInvalidInput
	}
	return id, nil
}

func (utils) NormalizePagination(limit, offset int32) (int32, int32) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
