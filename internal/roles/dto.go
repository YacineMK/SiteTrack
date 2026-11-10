package roles

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

type CreateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r CreateRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 64)),
		validation.Field(&r.Description, validation.Required, validation.Length(1, 255)),
	)
}

type RoleResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AssignRoleRequest struct {
	Role string `json:"role"`
}

func (r AssignRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role, validation.Required, validation.Length(1, 64)),
	)
}

type AssignedUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AppRole   string    `json:"app_role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatePolicyRequest struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func (r CreatePolicyRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Resource, validation.Required, validation.Length(1, 64)),
		validation.Field(&r.Action, validation.Required, validation.Length(1, 64)),
	)
}

type PolicyResponse struct {
	RoleID   string `json:"role_id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}
