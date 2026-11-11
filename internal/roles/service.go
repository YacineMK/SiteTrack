package roles

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/kit/rbac"
)

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrPolicyNotFound = errors.New("policy not found")
)

type Service interface {
	// scope: nil projectID = app scope (super_admin), non-nil = project scope.
	CreateRole(ctx context.Context, projectID *uuid.UUID, req CreateRoleRequest) (RoleResponse, error)
	UpdateRole(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID, req UpdateRoleRequest) (RoleResponse, error)
	DeleteRole(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID) error
	ListRoles(ctx context.Context, projectID *uuid.UUID) ([]RoleResponse, error)

	CreatePolicy(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID, req CreatePolicyRequest) (PolicyResponse, error)
	ListPolicies(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID) ([]PolicyResponse, error)
	DeletePolicy(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID, resource, action string) error

	AssignRole(ctx context.Context, userID uuid.UUID, req AssignRoleRequest) (AssignedUserResponse, error)
}

type service struct {
	repo  Repository
	utils Utils
}

func NewService(repo Repository) Service {
	return &service{repo: repo, utils: NewUtils()}
}

func (s *service) CreateRole(ctx context.Context, projectID *uuid.UUID, req CreateRoleRequest) (RoleResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if err := req.Validate(); err != nil {
		return RoleResponse{}, ErrInvalidInput
	}

	role, err := s.repo.CreateRole(ctx, projectID, req.Name, req.Description, req.IsAdmin)
	if err != nil {
		return RoleResponse{}, err
	}
	return s.utils.ToRoleResponse(role), nil
}

func (s *service) UpdateRole(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID, req UpdateRoleRequest) (RoleResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if err := req.Validate(); err != nil {
		return RoleResponse{}, ErrInvalidInput
	}
	role, err := s.repo.UpdateRole(ctx, projectID, roleID, req.Name, req.Description, req.IsAdmin)
	if err != nil {
		return RoleResponse{}, err
	}
	return s.utils.ToRoleResponse(role), nil
}

func (s *service) DeleteRole(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID) error {
	return s.repo.DeleteRole(ctx, projectID, roleID)
}

func (s *service) ListRoles(ctx context.Context, projectID *uuid.UUID) ([]RoleResponse, error) {
	roles, err := s.repo.ListRoles(ctx, projectID)
	if err != nil {
		return nil, err
	}

	response := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		response = append(response, s.utils.ToRoleResponse(role))
	}
	return response, nil
}

func (s *service) CreatePolicy(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID, req CreatePolicyRequest) (PolicyResponse, error) {
	req.Resource = strings.TrimSpace(req.Resource)
	req.Action = strings.TrimSpace(req.Action)
	if err := req.Validate(); err != nil {
		return PolicyResponse{}, ErrInvalidInput
	}
	if !rbac.Valid(req.Resource, req.Action) {
		return PolicyResponse{}, ErrInvalidInput
	}

	if _, err := s.repo.GetRole(ctx, projectID, roleID); err != nil {
		return PolicyResponse{}, err
	}
	if err := s.repo.AddPolicy(ctx, roleID, req.Resource, req.Action); err != nil {
		return PolicyResponse{}, err
	}

	return PolicyResponse{
		RoleID:   roleID.String(),
		Resource: req.Resource,
		Action:   req.Action,
	}, nil
}

func (s *service) ListPolicies(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID) ([]PolicyResponse, error) {
	if _, err := s.repo.GetRole(ctx, projectID, roleID); err != nil {
		return nil, err
	}

	policies, err := s.repo.ListPolicies(ctx, roleID)
	if err != nil {
		return nil, err
	}

	response := make([]PolicyResponse, 0, len(policies))
	for _, policy := range policies {
		response = append(response, PolicyResponse{
			RoleID:   policy.RoleID.String(),
			Resource: policy.Resource,
			Action:   policy.Action,
		})
	}
	return response, nil
}

func (s *service) DeletePolicy(ctx context.Context, projectID *uuid.UUID, roleID uuid.UUID, resource, action string) error {
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if resource == "" || action == "" {
		return ErrInvalidInput
	}

	if _, err := s.repo.GetRole(ctx, projectID, roleID); err != nil {
		return err
	}

	rows, err := s.repo.DeletePolicy(ctx, roleID, resource, action)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPolicyNotFound
	}
	return nil
}

func (s *service) AssignRole(ctx context.Context, userID uuid.UUID, req AssignRoleRequest) (AssignedUserResponse, error) {
	req.Role = strings.TrimSpace(req.Role)
	if err := req.Validate(); err != nil {
		return AssignedUserResponse{}, ErrInvalidInput
	}

	user, err := s.repo.AssignRole(ctx, userID, req.Role)
	if err != nil {
		return AssignedUserResponse{}, err
	}

	return AssignedUserResponse(user), nil
}
