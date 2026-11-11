//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"
)

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
	return prefix + "-" + time.Now().Format("150405.000")
}
