package projects

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const dateLayout = "2006-01-02"

type CreateProjectRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	PlannedStartDate string `json:"planned_start_date"`
	PlannedEndDate   string `json:"planned_end_date"`
}

func (r CreateProjectRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 200)),
		validation.Field(&r.Description, validation.Length(0, 2000)),
		validation.Field(&r.PlannedStartDate, validation.Required),
		validation.Field(&r.PlannedEndDate, validation.Required),
	)
}

type UpdateProjectRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	PlannedStartDate string `json:"planned_start_date"`
	PlannedEndDate   string `json:"planned_end_date"`
}

func (r UpdateProjectRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 200)),
		validation.Field(&r.Description, validation.Length(0, 2000)),
		validation.Field(&r.PlannedStartDate, validation.Required),
		validation.Field(&r.PlannedEndDate, validation.Required),
	)
}

type ProjectResponse struct {
	DisplayID        string    `json:"display_id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	PlannedStartDate string    `json:"planned_start_date"`
	PlannedEndDate   string    `json:"planned_end_date"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ListProjectsResponse struct {
	Items  []ProjectResponse `json:"items"`
	Total  int64             `json:"total"`
	Limit  int32             `json:"limit"`
	Offset int32             `json:"offset"`
}
