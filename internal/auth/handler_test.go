package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type testEnv struct {
	svc       *Service
	repo      *memoryUserRepository
	blacklist *memoryTokenBlacklist
	router    http.Handler
}

func newTestEnv() *testEnv {
	svc, repo, blacklist, _ := newTestService()
	handler := NewHandler(svc)
	router := chi.NewRouter()
	handler.Routes(router)
	return &testEnv{svc: svc, repo: repo, blacklist: blacklist, router: router}
}

func (e *testEnv) request(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func loginResponse(t *testing.T, e *testEnv) AuthResponse {
	t.Helper()
	rec := e.request(t, http.MethodPost, "/auth/login", `{"email":"alice@example.com","password":"password123"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding login response: %v", err)
	}
	return resp
}

func TestRegisterEndpoint(t *testing.T) {
	e := newTestEnv()

	rec := e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body["email"] != "alice@example.com" {
		t.Fatalf("email = %v, want alice@example.com", body["email"])
	}
	if _, leaked := body["password"]; leaked {
		t.Fatal("response leaked the password field")
	}
	if _, leaked := body["password_hash"]; leaked {
		t.Fatal("response leaked the password hash")
	}

	duplicate := e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`, "")
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", duplicate.Code)
	}
	var errBody map[string]string
	if err := json.Unmarshal(duplicate.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decoding error response: %v", err)
	}
	if errBody["error"] != "registration failed" {
		t.Fatalf("duplicate error = %q, want generic message", errBody["error"])
	}

	invalid := e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"short","app_role":"user"}`, "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", invalid.Code)
	}
}

func TestLoginEndpoint(t *testing.T) {
	e := newTestEnv()
	e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`, "")

	resp := loginResponse(t, e)
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("login response is missing tokens")
	}
	if resp.User.Email != "alice@example.com" {
		t.Fatalf("user email = %q, want alice@example.com", resp.User.Email)
	}

	wrong := e.request(t, http.MethodPost, "/auth/login", `{"email":"alice@example.com","password":"wrong"}`, "")
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want 401", wrong.Code)
	}
	var errBody map[string]string
	if err := json.Unmarshal(wrong.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decoding error response: %v", err)
	}
	if errBody["error"] != "invalid email or password" {
		t.Fatalf("error = %q, want %q", errBody["error"], "invalid email or password")
	}

	unknown := e.request(t, http.MethodPost, "/auth/login", `{"email":"nobody@example.com","password":"password123"}`, "")
	if unknown.Code != http.StatusUnauthorized {
		t.Fatalf("unknown email status = %d, want 401", unknown.Code)
	}
}

func TestMeEndpoint(t *testing.T) {
	e := newTestEnv()
	e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`, "")
	login := loginResponse(t, e)

	noToken := e.request(t, http.MethodGet, "/auth/me", "", "")
	if noToken.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want 401", noToken.Code)
	}

	ok := e.request(t, http.MethodGet, "/auth/me", "", login.AccessToken)
	if ok.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", ok.Code, ok.Body.String())
	}
	var profile UserResponse
	if err := json.Unmarshal(ok.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decoding profile: %v", err)
	}
	if profile.Email != "alice@example.com" {
		t.Fatalf("email = %q, want alice@example.com", profile.Email)
	}

	refreshAsAccess := e.request(t, http.MethodGet, "/auth/me", "", login.RefreshToken)
	if refreshAsAccess.Code != http.StatusUnauthorized {
		t.Fatalf("refresh-as-access status = %d, want 401", refreshAsAccess.Code)
	}

	claims, err := e.svc.jwt.ParseAccess(login.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if err := e.blacklist.Add(context.Background(), claims.ID, time.Minute); err != nil {
		t.Fatalf("blacklist.Add() error = %v", err)
	}

	revoked := e.request(t, http.MethodGet, "/auth/me", "", login.AccessToken)
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status = %d, want 401", revoked.Code)
	}
}

func TestRefreshEndpoint(t *testing.T) {
	e := newTestEnv()
	e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`, "")
	login := loginResponse(t, e)

	noAccess := e.request(t, http.MethodPost, "/auth/refresh", fmt.Sprintf(`{"refresh_token":%q}`, login.RefreshToken), "")
	if noAccess.Code != http.StatusUnauthorized {
		t.Fatalf("refresh without access token status = %d, want 401", noAccess.Code)
	}

	rec := e.request(t, http.MethodPost, "/auth/refresh", fmt.Sprintf(`{"refresh_token":%q}`, login.RefreshToken), login.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var refreshed AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("decoding refresh response: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatal("refresh response is missing tokens")
	}

	me := e.request(t, http.MethodGet, "/auth/me", "", refreshed.AccessToken)
	if me.Code != http.StatusOK {
		t.Fatalf("new access token status = %d, want 200", me.Code)
	}

	reused := e.request(t, http.MethodPost, "/auth/refresh", fmt.Sprintf(`{"refresh_token":%q}`, login.RefreshToken), refreshed.AccessToken)
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh token status = %d, want 401", reused.Code)
	}
}

func TestLogoutEndpoint(t *testing.T) {
	e := newTestEnv()
	e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`, "")
	login := loginResponse(t, e)
	logoutBody := fmt.Sprintf(`{"refresh_token":%q}`, login.RefreshToken)

	noAuth := e.request(t, http.MethodPost, "/auth/logout", logoutBody, "")
	if noAuth.Code != http.StatusUnauthorized {
		t.Fatalf("logout without access token status = %d, want 401", noAuth.Code)
	}

	noBody := e.request(t, http.MethodPost, "/auth/logout", "", login.AccessToken)
	if noBody.Code != http.StatusBadRequest {
		t.Fatalf("logout without body status = %d, want 400", noBody.Code)
	}

	rec := e.request(t, http.MethodPost, "/auth/logout", logoutBody, login.AccessToken)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204; body = %s", rec.Code, rec.Body.String())
	}

	me := e.request(t, http.MethodGet, "/auth/me", "", login.AccessToken)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("access after logout status = %d, want 401", me.Code)
	}

	refresh := e.request(t, http.MethodPost, "/auth/refresh", logoutBody, "")
	if refresh.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d, want 401", refresh.Code)
	}
}
