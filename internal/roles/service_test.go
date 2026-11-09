package roles

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type stubRepository struct {
	hasPolicyAllowed bool
	hasPolicyErr     error

	createRoleResult      Role
	createRoleErr         error
	createRoleCalls       int
	createRoleName        string
	createRoleDescription string

	roles        []Role
	listRolesErr error

	getRoleResult Role
	getRoleErr    error

	addPolicyErr   error
	addPolicyCalls int

	policies        []Policy
	listPoliciesErr error

	deleteRows  int64
	deleteErr   error
	deleteCalls int
}

func (s *stubRepository) HasPolicy(ctx context.Context, role, resource, action string) (bool, error) {
	return s.hasPolicyAllowed, s.hasPolicyErr
}

func (s *stubRepository) CreateRole(ctx context.Context, name, description string) (Role, error) {
	s.createRoleCalls++
	s.createRoleName = name
	s.createRoleDescription = description
	return s.createRoleResult, s.createRoleErr
}

func (s *stubRepository) ListRoles(ctx context.Context) ([]Role, error) {
	return s.roles, s.listRolesErr
}

func (s *stubRepository) GetRole(ctx context.Context, id uuid.UUID) (Role, error) {
	return s.getRoleResult, s.getRoleErr
}

func (s *stubRepository) AddPolicy(ctx context.Context, roleID uuid.UUID, resource, action string) error {
	s.addPolicyCalls++
	return s.addPolicyErr
}

func (s *stubRepository) ListPolicies(ctx context.Context, roleID uuid.UUID) ([]Policy, error) {
	return s.policies, s.listPoliciesErr
}

func (s *stubRepository) DeletePolicy(ctx context.Context, roleID uuid.UUID, resource, action string) (int64, error) {
	s.deleteCalls++
	return s.deleteRows, s.deleteErr
}

func TestServiceHasPolicy(t *testing.T) {
	repo := &stubRepository{hasPolicyAllowed: true}
	svc := NewService(repo)

	allowed, err := svc.HasPolicy(context.Background(), "company_store_manager", "company_store", "write")
	if err != nil {
		t.Fatalf("HasPolicy() error = %v", err)
	}
	if !allowed {
		t.Fatal("HasPolicy() = false, want true")
	}
}

func TestServicePropagatesRepositoryError(t *testing.T) {
	expected := errors.New("db down")
	svc := NewService(&stubRepository{hasPolicyErr: expected})

	if _, err := svc.HasPolicy(context.Background(), "user", "company_store", "read"); !errors.Is(err, expected) {
		t.Fatalf("HasPolicy() error = %v, want %v", err, expected)
	}
}

func TestCreateRole(t *testing.T) {
	id := uuid.New()
	repo := &stubRepository{createRoleResult: Role{ID: id, Name: "auditor", Description: "Audits projects"}}
	svc := NewService(repo)

	resp, err := svc.CreateRole(context.Background(), CreateRoleRequest{
		Name:        "  auditor  ",
		Description: "  Audits projects  ",
	})
	if err != nil {
		t.Fatalf("CreateRole() error = %v", err)
	}
	if resp.ID != id.String() || resp.Name != "auditor" || resp.Description != "Audits projects" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if repo.createRoleName != "auditor" || repo.createRoleDescription != "Audits projects" {
		t.Fatalf("repository called with %q/%q", repo.createRoleName, repo.createRoleDescription)
	}
}

func TestCreateRoleRejectsInvalidInput(t *testing.T) {
	repo := &stubRepository{}
	svc := NewService(repo)

	_, err := svc.CreateRole(context.Background(), CreateRoleRequest{Name: "", Description: "valid"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateRole() error = %v, want ErrInvalidInput", err)
	}
	if repo.createRoleCalls != 0 {
		t.Fatalf("repository calls = %d, want 0", repo.createRoleCalls)
	}
}

func TestCreateRoleRejectsDuplicate(t *testing.T) {
	svc := NewService(&stubRepository{createRoleErr: ErrRoleExists})

	_, err := svc.CreateRole(context.Background(), CreateRoleRequest{Name: "auditor", Description: "dup"})
	if !errors.Is(err, ErrRoleExists) {
		t.Fatalf("CreateRole() error = %v, want ErrRoleExists", err)
	}
}

func TestCreatePolicy(t *testing.T) {
	id := uuid.New()
	repo := &stubRepository{getRoleResult: Role{ID: id, Name: "auditor"}}
	svc := NewService(repo)

	resp, err := svc.CreatePolicy(context.Background(), id, CreatePolicyRequest{Resource: " reports ", Action: " read "})
	if err != nil {
		t.Fatalf("CreatePolicy() error = %v", err)
	}
	if resp.RoleID != id.String() || resp.Resource != "reports" || resp.Action != "read" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if repo.addPolicyCalls != 1 {
		t.Fatalf("AddPolicy calls = %d, want 1", repo.addPolicyCalls)
	}
}

func TestCreatePolicyRejectsUnknownRole(t *testing.T) {
	repo := &stubRepository{getRoleErr: ErrRoleNotFound}
	svc := NewService(repo)

	_, err := svc.CreatePolicy(context.Background(), uuid.New(), CreatePolicyRequest{Resource: "reports", Action: "read"})
	if !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("CreatePolicy() error = %v, want ErrRoleNotFound", err)
	}
	if repo.addPolicyCalls != 0 {
		t.Fatalf("AddPolicy calls = %d, want 0", repo.addPolicyCalls)
	}
}

func TestListPolicies(t *testing.T) {
	id := uuid.New()
	repo := &stubRepository{
		getRoleResult: Role{ID: id},
		policies:      []Policy{{RoleID: id, Resource: "reports", Action: "read"}},
	}
	svc := NewService(repo)

	resp, err := svc.ListPolicies(context.Background(), id)
	if err != nil {
		t.Fatalf("ListPolicies() error = %v", err)
	}
	if len(resp) != 1 || resp[0].Resource != "reports" || resp[0].Action != "read" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDeletePolicy(t *testing.T) {
	svc := NewService(&stubRepository{deleteRows: 1})

	if err := svc.DeletePolicy(context.Background(), uuid.New(), "reports", "read"); err != nil {
		t.Fatalf("DeletePolicy() error = %v", err)
	}
}

func TestDeletePolicyNotFound(t *testing.T) {
	svc := NewService(&stubRepository{deleteRows: 0})

	if err := svc.DeletePolicy(context.Background(), uuid.New(), "reports", "read"); !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("DeletePolicy() error = %v, want ErrPolicyNotFound", err)
	}
}

func TestDeletePolicyRejectsEmptyParams(t *testing.T) {
	repo := &stubRepository{deleteRows: 1}
	svc := NewService(repo)

	if err := svc.DeletePolicy(context.Background(), uuid.New(), "", "read"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("DeletePolicy() error = %v, want ErrInvalidInput", err)
	}
	if repo.deleteCalls != 0 {
		t.Fatalf("repository calls = %d, want 0", repo.deleteCalls)
	}
}
