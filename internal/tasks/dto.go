package tasks

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const dateLayout = "2006-01-02"

type CreateTaskRequest struct {
	Name             string `json:"name"`
	PlannedStartDate string `json:"planned_start_date"`
	PlannedEndDate   string `json:"planned_end_date"`
}

func (r CreateTaskRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 200)),
		validation.Field(&r.PlannedStartDate, validation.Required),
		validation.Field(&r.PlannedEndDate, validation.Required),
	)
}

type UpdateTaskRequest struct {
	Name             string `json:"name"`
	PlannedStartDate string `json:"planned_start_date"`
	PlannedEndDate   string `json:"planned_end_date"`
}

func (r UpdateTaskRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 200)),
		validation.Field(&r.PlannedStartDate, validation.Required),
		validation.Field(&r.PlannedEndDate, validation.Required),
	)
}

type TaskResponse struct {
	DisplayID        string    `json:"display_id"`
	ProjectDisplayID string    `json:"project_display_id"`
	Name             string    `json:"name"`
	PlannedStartDate string    `json:"planned_start_date"`
	PlannedEndDate   string    `json:"planned_end_date"`
	Status           string    `json:"status"`
	Late             bool      `json:"late"`
	Blocked          bool      `json:"blocked"`
	DependsOn        []string  `json:"depends_on"`
	DependentsCount  int64     `json:"dependents_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ListTasksResponse struct {
	Items []TaskResponse `json:"items"`
	Total int64          `json:"total"`
	Limit int32          `json:"limit"`
}

type DependencyResponse struct {
	DisplayID string `json:"display_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Steps     int32  `json:"steps"`
}

type TaskDetailResponse struct {
	TaskResponse
	Upstream   []DependencyResponse `json:"upstream"`
	Downstream []DependencyResponse `json:"downstream"`
}

type AddDependencyRequest struct {
	DependsOn string `json:"depends_on"`
}

func (r AddDependencyRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.DependsOn, validation.Required),
	)
}
