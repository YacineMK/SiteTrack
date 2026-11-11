//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"
)

type invitationResp struct {
	ID     string `json:"id"`
	RoleID string `json:"role_id"`
	Email  string `json:"email"`
}

type memberResp struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

type membersList struct {
	Items []memberResp `json:"items"`
	Total int          `json:"total"`
}

var _ = strings.HasPrefix // silence linter if helpers unused

// client is a per-user http.Client with a cookie jar so Login persists cookies.
type client struct {
	t    *testing.T
	http *http.Client
	name string
}

func newClient(t *testing.T, name string) *client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &client{
		t:    t,
		http: &http.Client{Jar: jar, Timeout: 10 * time.Second},
		name: name,
	}
}

func (c *client) do(method, path string, body any) *http.Response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, ts.URL+path, r)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func (c *client) ok(method, path string, body any, want int, out any) {
	c.t.Helper()
	resp := c.do(method, path, body)
	defer resp.Body.Close()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		c.t.Fatalf("[%s] %s %s = %d, want %d: %s", c.name, method, path, resp.StatusCode, want, string(b))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			c.t.Fatalf("[%s] decode %s %s: %v", c.name, method, path, err)
		}
	}
}

// expect runs the request and checks only the status code, discarding the body.
func (c *client) expect(method, path string, body any, want int) {
	c.t.Helper()
	resp := c.do(method, path, body)
	defer resp.Body.Close()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		c.t.Fatalf("[%s] %s %s = %d, want %d: %s", c.name, method, path, resp.StatusCode, want, string(b))
	}
}

func (c *client) getJSON(path string, want int, out any) {
	c.ok(http.MethodGet, path, nil, want, out)
}

// register + login in one call, returns a logged-in client.
func registerAndLogin(t *testing.T, name, email, password, appRole string) *client {
	t.Helper()
	c := newClient(t, name)
	c.ok(http.MethodPost, "/auth/register", map[string]any{
		"name":     name,
		"email":    email,
		"password": password,
		"app_role": appRole,
	}, http.StatusCreated, nil)
	c.ok(http.MethodPost, "/auth/login", map[string]any{
		"email":    email,
		"password": password,
	}, http.StatusOK, nil)
	return c
}

func urlWithQuery(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}
	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	if !strings.Contains(base, "?") {
		return base + "?" + q.Encode()
	}
	return base + "&" + q.Encode()
}

func unique(prefix string) string {
	return prefix + "-" + time.Now().Format("150405.000000")
}

// lookupEmail returns the email bound to a logged-in client via /auth/me.
func (c *client) lookupEmail(t *testing.T) string {
	t.Helper()
	var me struct {
		Email string `json:"email"`
	}
	c.getJSON("/auth/me", http.StatusOK, &me)
	return me.Email
}

// joinProject bypasses the invite flow by writing project_members directly.
// Only used by tests that need a second member fast and don't test invites.
func joinProject(t *testing.T, c *client, projectDisplayID string, asAdmin bool) {
	t.Helper()
	var me struct {
		ID string `json:"id"`
	}
	c.getJSON("/auth/me", http.StatusOK, &me)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	roleName := "Member"
	if asAdmin {
		roleName = "Admin"
	}
	_, err := dbpool.Pool.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role_id)
		SELECT p.id, $1::uuid, r.id
		FROM projects p
		JOIN roles r ON r.project_id = p.id AND r.is_system AND r.name = $2
		WHERE p.display_id = $3
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, me.ID, roleName, projectDisplayID)
	if err != nil {
		t.Fatalf("joinProject(%s, admin=%v): %v", projectDisplayID, asAdmin, err)
	}
}
