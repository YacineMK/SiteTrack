package materials

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

type CreateMaterialRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Unit     string `json:"unit"`
}

func (r CreateMaterialRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 200)),
		validation.Field(&r.Category, validation.Required, validation.Length(1, 100)),
		validation.Field(&r.Unit, validation.Required, validation.Length(1, 50)),
	)
}

type UpdateMaterialRequest struct {
	Name     *string `json:"name,omitempty"`
	Category *string `json:"category,omitempty"`
	Unit     *string `json:"unit,omitempty"`
}

func (r UpdateMaterialRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.When(r.Name != nil, validation.Required, validation.Length(1, 200))),
		validation.Field(&r.Category, validation.When(r.Category != nil, validation.Required, validation.Length(1, 100))),
		validation.Field(&r.Unit, validation.When(r.Unit != nil, validation.Required, validation.Length(1, 50))),
	)
}

type MaterialResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListMaterialsResponse struct {
	Items  []MaterialResponse `json:"items"`
	Total  int64              `json:"total"`
	Limit  int32              `json:"limit"`
	Offset int32              `json:"offset"`
}
