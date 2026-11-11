package projects

import (
	"errors"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
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
	DisplayID        string     `json:"display_id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	PlannedStartDate string     `json:"planned_start_date"`
	PlannedEndDate   string     `json:"planned_end_date"`
	IsAdmin          bool       `json:"is_admin"`
	MemberCount      int64      `json:"member_count"`
	TaskCounts       TaskCounts `json:"task_counts"`
	Progress         float64    `json:"progress"`
	LateCount        int64      `json:"late_count"`
	BlockedCount     int64      `json:"blocked_count"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type TaskCounts struct {
	NotStarted int64 `json:"not_started"`
	InProgress int64 `json:"in_progress"`
	Done       int64 `json:"done"`
}

type MemberResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

type ListMembersResponse struct {
	Items []MemberResponse `json:"items"`
	Total int64            `json:"total"`
}

type UpdateMemberRequest struct {
	IsAdmin *bool `json:"is_admin"`
}

func (r UpdateMemberRequest) Validate() error {
	if r.IsAdmin == nil {
		return errors.New("is_admin is required")
	}
	return nil
}

type InviteRequest struct {
	Email  string    `json:"email"`
	RoleID uuid.UUID `json:"role_id"` // must belong to this project; omit for default Member
}

func (r InviteRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required, is.Email),
	)
}

type InvitationResponse struct {
	ID               uuid.UUID `json:"id"`
	ProjectDisplayID string    `json:"project_display_id"`
	InviteeID        uuid.UUID `json:"invitee_id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	RoleID           uuid.UUID `json:"role_id"`
	RoleName         string    `json:"role_name"`
	Status           string    `json:"status"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type ListProjectsResponse struct {
	Items []ProjectResponse `json:"items"`
	Total int64             `json:"total"`
	Limit int32             `json:"limit"`
}
