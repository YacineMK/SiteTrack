package projectstore

import (
	"time"

	"github.com/google/uuid"
)

type ProjectStockResponse struct {
	MaterialID   uuid.UUID  `json:"material_id"`
	Name         string     `json:"name"`
	Category     string     `json:"category"`
	Unit         string     `json:"unit"`
	Quantity     float64    `json:"quantity"`
	LastDelivery *time.Time `json:"last_delivery_at"`
}

type ListProjectStockResponse struct {
	Items  []ProjectStockResponse `json:"items"`
	Total  int64                  `json:"total"`
	Limit  int32                  `json:"limit"`
	Offset int32                  `json:"offset"`
}
