package companystore

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
)

type AddStockRequest struct {
	MaterialID string  `json:"material_id"`
	Quantity   float64 `json:"quantity"`
}

func (r AddStockRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.MaterialID, validation.Required, is.UUID),
		validation.Field(&r.Quantity, validation.Required, validation.Min(0.001)),
	)
}

type SetReorderPointRequest struct {
	ReorderPoint float64 `json:"reorder_point"`
}

func (r SetReorderPointRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ReorderPoint, validation.Min(0.0)),
	)
}

type StockResponse struct {
	MaterialID   uuid.UUID `json:"material_id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	Unit         string    `json:"unit"`
	Quantity     float64   `json:"quantity"`
	ReorderPoint float64   `json:"reorder_point"`
	InTransit    float64   `json:"in_transit"`
	Status       string    `json:"status"`
}

type ListStockResponse struct {
	Items  []StockResponse `json:"items"`
	Total  int64           `json:"total"`
	Limit  int32           `json:"limit"`
	Offset int32           `json:"offset"`
}
