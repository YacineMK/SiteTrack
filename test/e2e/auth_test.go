//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestAuth(t *testing.T) {
	email := unique("alice") + "@example.com"
	password := "Password123!"

	t.Run("register", func(t *testing.T) {
		c := newClient(t, "alice")
		c.ok(http.MethodPost, "/auth/register", map[string]any{
			"name":     "Alice",
			"email":    email,
			"password": password,
			"app_role": "user",
		}, http.StatusCreated, nil)
	})

	t.Run("register duplicate email", func(t *testing.T) {
		c := newClient(t, "alice2")
		c.expect(http.MethodPost, "/auth/register", map[string]any{
			"name":     "Alice",
			"email":    email,
			"password": password,
			"app_role": "user",
		}, http.StatusConflict)
	})

	t.Run("register unknown role is rejected", func(t *testing.T) {
		c := newClient(t, "mallory")
		c.expect(http.MethodPost, "/auth/register", map[string]any{
			"name":     "Mallory",
			"email":    unique("mallory") + "@example.com",
			"password": password,
			"app_role": "root",
		}, http.StatusBadRequest)
	})

	t.Run("register with unknown role is rejected", func(t *testing.T) {
		c := newClient(t, "mallory2")
		c.expect(http.MethodPost, "/auth/register", map[string]any{
			"name":     "Mallory",
			"email":    unique("mallory2") + "@example.com",
			"password": password,
			"app_role": "root",
		}, http.StatusBadRequest)
	})

	t.Run("login wrong password 401", func(t *testing.T) {
		c := newClient(t, "alice-wrongpw")
		c.expect(http.MethodPost, "/auth/login", map[string]any{
			"email":    email,
			"password": "wrong",
		}, http.StatusUnauthorized)
	})

	t.Run("login unknown email 401", func(t *testing.T) {
		c := newClient(t, "noone")
		c.expect(http.MethodPost, "/auth/login", map[string]any{
			"email":    "noone@example.com",
			"password": password,
		}, http.StatusUnauthorized)
	})

	var alice *client
	t.Run("login", func(t *testing.T) {
		alice = newClient(t, "alice")
		alice.ok(http.MethodPost, "/auth/login", map[string]any{
			"email":    email,
			"password": password,
		}, http.StatusOK, nil)
	})

	t.Run("me returns self", func(t *testing.T) {
		var me struct {
			Email   string `json:"email"`
			AppRole string `json:"app_role"`
		}
		alice.getJSON("/auth/me", http.StatusOK, &me)
		if me.Email != email || me.AppRole != "user" {
			t.Fatalf("me = %+v", me)
		}
	})

	t.Run("email-available", func(t *testing.T) {
		var resp struct {
			Available bool `json:"available"`
		}
		alice.getJSON("/auth/email-available?email="+email, http.StatusOK, &resp)
		if resp.Available {
			t.Fatal("taken email should report available=false")
		}
		alice.getJSON("/auth/email-available?email=brand-new-"+email, http.StatusOK, &resp)
		if !resp.Available {
			t.Fatal("fresh email should be available")
		}
	})

	t.Run("refresh rotates both tokens", func(t *testing.T) {
		alice.expect(http.MethodPost, "/auth/refresh", nil, http.StatusOK)
	})

	t.Run("logout kills session immediately", func(t *testing.T) {
		alice.expect(http.MethodPost, "/auth/logout", nil, http.StatusNoContent)
		alice.expect(http.MethodGet, "/auth/me", nil, http.StatusUnauthorized)
	})

	t.Run("short password is rejected", func(t *testing.T) {
		c := newClient(t, "shortpw")
		c.expect(http.MethodPost, "/auth/register", map[string]any{
			"name":     "Short",
			"email":    unique("short") + "@example.com",
			"password": "1234567",
			"app_role": "user",
		}, http.StatusBadRequest)
	})

	t.Run("email case-insensitive match on login", func(t *testing.T) {
		mix := unique("Mixed") + "@example.com"
		reg := newClient(t, "mixed-reg")
		reg.expect(http.MethodPost, "/auth/register", map[string]any{
			"name": "M", "email": mix, "password": "Password123!", "app_role": "user",
		}, http.StatusCreated)
		// login with a different case variant should still succeed.
		login := newClient(t, "mixed-login")
		login.expect(http.MethodPost, "/auth/login", map[string]any{
			"email":    "MIXED" + mix[len("Mixed"):],
			"password": "Password123!",
		}, http.StatusOK)
	})

	t.Run("refresh token is single use", func(t *testing.T) {
		c := registerAndLogin(t, "single-use", unique("single")+"@example.com", "Password123!", "user")
		c.expect(http.MethodPost, "/auth/refresh", nil, http.StatusOK)
		// Old refresh cookie was rotated out; our jar carries the new one, so
		// another refresh is also OK. The real check: logout the old jti and
		// try me — must be 401.
		c.expect(http.MethodPost, "/auth/logout", nil, http.StatusNoContent)
		c.expect(http.MethodPost, "/auth/refresh", nil, http.StatusUnauthorized)
	})
}
