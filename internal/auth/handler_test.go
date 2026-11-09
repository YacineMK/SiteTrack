package auth

import (
	"context"
	"encoding/json"
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
	handler := NewHandler(svc, false)
	router := chi.NewRouter()
	handler.Routes(router)
	return &testEnv{svc: svc, repo: repo, blacklist: blacklist, router: router}
}

func (e *testEnv) request(t *testing.T, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func registerAliceHTTP(t *testing.T, e *testEnv) {
	t.Helper()
	rec := e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
}

func loginAliceHTTP(t *testing.T, e *testEnv) (*AuthResponse, []*http.Cookie) {
	t.Helper()
	rec := e.request(t, http.MethodPost, "/auth/login", `{"email":"alice@example.com","password":"password123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding login response: %v", err)
	}
	cookies := rec.Result().Cookies()
	if findCookie(cookies, accessCookieName) == nil {
		t.Fatal("login did not set the access cookie")
	}
	if findCookie(cookies, refreshCookieName) == nil {
		t.Fatal("login did not set the refresh cookie")
	}
	return &resp, cookies
}

func TestRegisterEndpoint(t *testing.T) {
	e := newTestEnv()

	rec := e.request(t, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`)
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
		`{"name":"Alice","email":"alice@example.com","password":"password123","app_role":"user"}`)
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
		`{"name":"Alice","email":"alice@example.com","password":"short","app_role":"user"}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", invalid.Code)
	}
}

func TestLoginEndpoint(t *testing.T) {
	e := newTestEnv()
	registerAliceHTTP(t, e)

	rec := e.request(t, http.MethodPost, "/auth/login", `{"email":"alice@example.com","password":"password123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if _, leaked := body["access_token"]; leaked {
		t.Fatal("response leaked the access token")
	}
	if _, leaked := body["refresh_token"]; leaked {
		t.Fatal("response leaked the refresh token")
	}

	cookies := rec.Result().Cookies()
	access := findCookie(cookies, accessCookieName)
	refresh := findCookie(cookies, refreshCookieName)
	if access == nil || refresh == nil {
		t.Fatal("login did not set both cookies")
	}
	if !access.HttpOnly || !refresh.HttpOnly {
		t.Fatal("session cookies must be HttpOnly")
	}
	if access.Path != "/" {
		t.Fatalf("access cookie path = %q, want /", access.Path)
	}
	if refresh.Path != "/auth" {
		t.Fatalf("refresh cookie path = %q, want /auth", refresh.Path)
	}

	wrong := e.request(t, http.MethodPost, "/auth/login", `{"email":"alice@example.com","password":"wrong"}`)
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

	unknown := e.request(t, http.MethodPost, "/auth/login", `{"email":"nobody@example.com","password":"password123"}`)
	if unknown.Code != http.StatusUnauthorized {
		t.Fatalf("unknown email status = %d, want 401", unknown.Code)
	}
}

func TestMeEndpoint(t *testing.T) {
	e := newTestEnv()
	registerAliceHTTP(t, e)
	_, cookies := loginAliceHTTP(t, e)
	access := findCookie(cookies, accessCookieName)
	refresh := findCookie(cookies, refreshCookieName)

	noCookie := e.request(t, http.MethodGet, "/auth/me", "")
	if noCookie.Code != http.StatusUnauthorized {
		t.Fatalf("missing cookie status = %d, want 401", noCookie.Code)
	}

	ok := e.request(t, http.MethodGet, "/auth/me", "", access)
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

	refreshAsAccess := e.request(t, http.MethodGet, "/auth/me", "", &http.Cookie{Name: accessCookieName, Value: refresh.Value, Path: "/"})
	if refreshAsAccess.Code != http.StatusUnauthorized {
		t.Fatalf("refresh-as-access status = %d, want 401", refreshAsAccess.Code)
	}

	claims, err := e.svc.jwt.ParseAccess(access.Value)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if err := e.blacklist.Add(context.Background(), claims.ID, time.Minute); err != nil {
		t.Fatalf("blacklist.Add() error = %v", err)
	}

	revoked := e.request(t, http.MethodGet, "/auth/me", "", access)
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status = %d, want 401", revoked.Code)
	}
}

func TestRefreshEndpoint(t *testing.T) {
	e := newTestEnv()
	registerAliceHTTP(t, e)
	_, cookies := loginAliceHTTP(t, e)
	access := findCookie(cookies, accessCookieName)
	refresh := findCookie(cookies, refreshCookieName)

	noCookies := e.request(t, http.MethodPost, "/auth/refresh", "")
	if noCookies.Code != http.StatusUnauthorized {
		t.Fatalf("refresh without cookies status = %d, want 401", noCookies.Code)
	}

	rec := e.request(t, http.MethodPost, "/auth/refresh", "", access, refresh)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	newCookies := rec.Result().Cookies()
	newAccess := findCookie(newCookies, accessCookieName)
	newRefresh := findCookie(newCookies, refreshCookieName)
	if newAccess == nil || newRefresh == nil {
		t.Fatal("refresh did not set both cookies")
	}
	if newAccess.Value == access.Value {
		t.Fatal("refresh must rotate the access token")
	}

	me := e.request(t, http.MethodGet, "/auth/me", "", newAccess)
	if me.Code != http.StatusOK {
		t.Fatalf("new access cookie status = %d, want 200", me.Code)
	}

	reused := e.request(t, http.MethodPost, "/auth/refresh", "", newAccess, refresh)
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh cookie status = %d, want 401", reused.Code)
	}
}

func TestLogoutEndpoint(t *testing.T) {
	e := newTestEnv()
	registerAliceHTTP(t, e)
	_, cookies := loginAliceHTTP(t, e)
	access := findCookie(cookies, accessCookieName)
	refresh := findCookie(cookies, refreshCookieName)

	noCookies := e.request(t, http.MethodPost, "/auth/logout", "")
	if noCookies.Code != http.StatusUnauthorized {
		t.Fatalf("logout without cookies status = %d, want 401", noCookies.Code)
	}

	rec := e.request(t, http.MethodPost, "/auth/logout", "", access, refresh)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204; body = %s", rec.Code, rec.Body.String())
	}
	cleared := rec.Result().Cookies()
	if c := findCookie(cleared, accessCookieName); c == nil || c.Value != "" {
		t.Fatal("logout must clear the access cookie")
	}
	if c := findCookie(cleared, refreshCookieName); c == nil || c.Value != "" {
		t.Fatal("logout must clear the refresh cookie")
	}

	me := e.request(t, http.MethodGet, "/auth/me", "", access)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("access after logout status = %d, want 401", me.Code)
	}

	refreshAfter := e.request(t, http.MethodPost, "/auth/refresh", "", access, refresh)
	if refreshAfter.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d, want 401", refreshAfter.Code)
	}
}
