package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type stubPolicyChecker struct {
	allowed  bool
	err      error
	calls    int
	role     string
	resource string
	action   string
}

func (s *stubPolicyChecker) HasPolicy(ctx context.Context, role, resource, action string) (bool, error) {
	s.calls++
	s.role = role
	s.resource = resource
	s.action = action
	return s.allowed, s.err
}

func TestRequireAppPolicy(t *testing.T) {
	tests := []struct {
		name    string
		allowed bool
		err     error
		want    int
	}{
		{name: "allowed", allowed: true, want: http.StatusNoContent},
		{name: "forbidden", allowed: false, want: http.StatusForbidden},
		{name: "checker error", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &stubPolicyChecker{allowed: tt.allowed, err: tt.err}
			authz := NewAuthZ(checker)
			handler := authz.RequireAppPolicy("company_store", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "/company-store", nil)
			req = req.WithContext(WithContext(context.Background(), uuid.New(), "company_store_manager", 1))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			if checker.calls != 1 {
				t.Fatalf("policy checker calls = %d, want 1", checker.calls)
			}
			if checker.role != "company_store_manager" || checker.resource != "company_store" || checker.action != "read" {
				t.Fatalf("policy checked with %q/%q/%q", checker.role, checker.resource, checker.action)
			}
		})
	}
}

func TestRequireAppPolicyWithoutRole(t *testing.T) {
	checker := &stubPolicyChecker{allowed: true}
	authz := NewAuthZ(checker)
	handler := authz.RequireAppPolicy("company_store", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/company-store", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if checker.calls != 0 {
		t.Fatalf("policy checker calls = %d, want 0", checker.calls)
	}
}

func TestRequireAppRole(t *testing.T) {
	tests := []struct {
		name    string
		current string
		want    int
	}{
		{name: "allowed", current: "super_admin", want: http.StatusNoContent},
		{name: "forbidden", current: "user", want: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authz := NewAuthZ(&stubPolicyChecker{})
			handler := authz.RequireAppRole("super_admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "/roles", nil)
			req = req.WithContext(WithContext(context.Background(), uuid.New(), tt.current, 1))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestRequireAppRoleWithoutRole(t *testing.T) {
	authz := NewAuthZ(&stubPolicyChecker{})
	handler := authz.RequireAppRole("super_admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
