package roles

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrPolicyNotFound = errors.New("policy not found")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateRole(ctx context.Context, req CreateRoleRequest) (RoleResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if err := req.Validate(); err != nil {
		return RoleResponse{}, ErrInvalidInput
	}

	role, err := s.repo.CreateRole(ctx, req.Name, req.Description)
	if err != nil {
		return RoleResponse{}, err
	}
	return toRoleResponse(role), nil
}

func (s *Service) ListRoles(ctx context.Context) ([]RoleResponse, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		response = append(response, toRoleResponse(role))
	}
	return response, nil
}

func (s *Service) CreatePolicy(ctx context.Context, roleID uuid.UUID, req CreatePolicyRequest) (PolicyResponse, error) {
	req.Resource = strings.TrimSpace(req.Resource)
	req.Action = strings.TrimSpace(req.Action)
	if err := req.Validate(); err != nil {
		return PolicyResponse{}, ErrInvalidInput
	}

	if _, err := s.repo.GetRole(ctx, roleID); err != nil {
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

func (s *Service) ListPolicies(ctx context.Context, roleID uuid.UUID) ([]PolicyResponse, error) {
	if _, err := s.repo.GetRole(ctx, roleID); err != nil {
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

func (s *Service) DeletePolicy(ctx context.Context, roleID uuid.UUID, resource, action string) error {
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if resource == "" || action == "" {
		return ErrInvalidInput
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

func toRoleResponse(role Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID.String(),
		Name:        role.Name,
		Description: role.Description,
	}
}
