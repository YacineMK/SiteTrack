package roles

import (
	"errors"
	"net/http"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Utils interface {
	ToRoleResponse(role Role) RoleResponse
	WriteServiceError(w http.ResponseWriter, err error)
}

type utils struct{}

func NewUtils() Utils {
	return utils{}
}

func (utils) ToRoleResponse(role Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID.String(),
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
	}
}

func (utils) WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		jsonutil.Error(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrRoleExists):
		jsonutil.Error(w, http.StatusConflict, "role already exists")
	case errors.Is(err, ErrRoleNotFound):
		jsonutil.Error(w, http.StatusBadRequest, "role not found")
	case errors.Is(err, ErrUserNotFound):
		jsonutil.Error(w, http.StatusNotFound, "user not found")
	case errors.Is(err, ErrPolicyExists):
		jsonutil.Error(w, http.StatusConflict, "policy already exists")
	case errors.Is(err, ErrPolicyNotFound):
		jsonutil.Error(w, http.StatusNotFound, "policy not found")
	default:
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
