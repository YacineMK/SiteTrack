package roles

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/auth"
)

func newTestRouter(repo *stubRepository) http.Handler {
	svc := NewService(repo)
	handler := NewHandler(svc, auth.NewAuthZ(svc))
	router := chi.NewRouter()
	handler.Routes(router)
	return router
}

func doRequest(t *testing.T, router http.Handler, method, path, body, role string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req = req.WithContext(auth.WithContext(context.Background(), uuid.New(), role, 1))
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCreateRoleEndpoint(t *testing.T) {
	id := uuid.New()
	repo := &stubRepository{createRoleResult: Role{ID: id, Name: "auditor", Description: "Audits projects"}}
	router := newTestRouter(repo)

	rec := doRequest(t, router, http.MethodPost, "/roles", `{"name":"auditor","description":"Audits projects"}`, "super_admin")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}

	var body RoleResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.ID != id.String() || body.Name != "auditor" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestCreateRoleRequiresSuperAdmin(t *testing.T) {
	repo := &stubRepository{}
	router := newTestRouter(repo)

	rec := doRequest(t, router, http.MethodPost, "/roles", `{"name":"auditor","description":"Audits"}`, "user")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if repo.createRoleCalls != 0 {
		t.Fatalf("repository calls = %d, want 0", repo.createRoleCalls)
	}
}

func TestCreateRoleRequiresAuthentication(t *testing.T) {
	router := newTestRouter(&stubRepository{})

	rec := doRequest(t, router, http.MethodPost, "/roles", `{"name":"auditor","description":"Audits"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestListRolesEndpoint(t *testing.T) {
	repo := &stubRepository{roles: []Role{{ID: uuid.New(), Name: "user", Description: "Basic"}}}
	router := newTestRouter(repo)

	rec := doRequest(t, router, http.MethodGet, "/roles", "", "super_admin")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var body []RoleResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(body) != 1 || body[0].Name != "user" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestCreatePolicyEndpoint(t *testing.T) {
	id := uuid.New()
	repo := &stubRepository{getRoleResult: Role{ID: id, Name: "auditor"}}
	router := newTestRouter(repo)

	rec := doRequest(t, router, http.MethodPost, "/roles/"+id.String()+"/policies",
		`{"resource":"reports","action":"read"}`, "super_admin")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}

	var body PolicyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.RoleID != id.String() || body.Resource != "reports" || body.Action != "read" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestCreatePolicyUnknownRole(t *testing.T) {
	repo := &stubRepository{getRoleErr: ErrRoleNotFound}
	router := newTestRouter(repo)

	rec := doRequest(t, router, http.MethodPost, "/roles/"+uuid.New().String()+"/policies",
		`{"resource":"reports","action":"read"}`, "super_admin")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestCreatePolicyInvalidRoleID(t *testing.T) {
	router := newTestRouter(&stubRepository{})

	rec := doRequest(t, router, http.MethodPost, "/roles/not-a-uuid/policies",
		`{"resource":"reports","action":"read"}`, "super_admin")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDeletePolicyEndpointNotFound(t *testing.T) {
	repo := &stubRepository{deleteRows: 0}
	router := newTestRouter(repo)

	rec := doRequest(t, router, http.MethodDelete, "/roles/"+uuid.New().String()+"/policies/reports/read", "", "super_admin")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
