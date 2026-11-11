//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

type roleResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"is_admin"`
	IsSystem bool   `json:"is_system"`
}

type roleList []roleResp

func TestRolesAppScope(t *testing.T) {
	root := registerAndLogin(t, "root-ar", unique("rootar")+"@example.com", "Password123!", "super_admin")
	user := registerAndLogin(t, "user-ar", unique("userar")+"@example.com", "Password123!", "user")

	t.Run("non super_admin cannot list app roles", func(t *testing.T) {
		user.expect(http.MethodGet, "/roles/", nil, http.StatusForbidden)
	})

	t.Run("super_admin lists app roles", func(t *testing.T) {
		var rl roleList
		root.getJSON("/roles/", http.StatusOK, &rl)
		if len(rl) < 3 {
			t.Fatalf("expected at least 3 seeded app roles, got %d", len(rl))
		}
	})

	var custom roleResp
	t.Run("super_admin creates a custom app role", func(t *testing.T) {
		root.ok(http.MethodPost, "/roles/", map[string]any{
			"name": unique("custom"), "description": "e2e", "is_admin": false,
		}, http.StatusCreated, &custom)
	})

	t.Run("registering with that role succeeds (role DB lookup)", func(t *testing.T) {
		c := newClient(t, "custom-user")
		c.expect(http.MethodPost, "/auth/register", map[string]any{
			"name":     "C", "email": unique("custom-reg") + "@example.com",
			"password": "Password123!", "app_role": custom.Name,
		}, http.StatusCreated)
	})

	t.Run("deleting a system role is rejected", func(t *testing.T) {
		var rl roleList
		root.getJSON("/roles/", http.StatusOK, &rl)
		var sysID string
		for _, r := range rl {
			if r.IsSystem {
				sysID = r.ID
				break
			}
		}
		if sysID == "" {
			t.Fatal("no system role to test against")
		}
		root.expect(http.MethodDelete, "/roles/"+sysID, nil, http.StatusConflict)
	})

	t.Run("assign user role lands on next request", func(t *testing.T) {
		var me struct {
			ID string `json:"id"`
		}
		user.getJSON("/auth/me", http.StatusOK, &me)
		root.expect(http.MethodPatch, "/users/"+me.ID+"/role", map[string]any{
			"role": "company_store_manager",
		}, http.StatusOK)
		// user should now be able to list company stock (manager permission).
		user.expect(http.MethodGet, "/company-store/stock", nil, http.StatusOK)
	})
}

func TestRolesProjectScope(t *testing.T) {
	admin := registerAndLogin(t, "proj-ra", unique("pra")+"@example.com", "Password123!", "user")
	assignee := registerAndLogin(t, "proj-assn", unique("pas")+"@example.com", "Password123!", "user")

	p := createProject(t, admin, unique("ProjRolesProj"))

	t.Run("project admin lists built-in Admin+Member", func(t *testing.T) {
		var rl roleList
		admin.getJSON("/projects/"+p.DisplayID+"/roles/", http.StatusOK, &rl)
		names := map[string]bool{}
		for _, r := range rl {
			names[r.Name] = true
		}
		if !names["Admin"] || !names["Member"] {
			t.Fatalf("missing built-in roles, got %+v", rl)
		}
	})

	var foreman roleResp
	t.Run("admin creates a custom project role", func(t *testing.T) {
		admin.ok(http.MethodPost, "/projects/"+p.DisplayID+"/roles/", map[string]any{
			"name": "Foreman", "description": "can edit tasks, nothing else", "is_admin": false,
		}, http.StatusCreated, &foreman)
	})

	t.Run("admin attaches a policy to the custom role (tasks:write)", func(t *testing.T) {
		admin.expect(http.MethodPost, "/projects/"+p.DisplayID+"/roles/"+foreman.ID+"/policies", map[string]any{
			"resource": "tasks", "action": "write",
		}, http.StatusCreated)
	})

	// Shortcut: join directly for authz checks; invite flow has its own tests.
	joinProject(t, assignee, p.DisplayID, false)

	t.Run("deleting a system role is rejected", func(t *testing.T) {
		var rl roleList
		admin.getJSON("/projects/"+p.DisplayID+"/roles/", http.StatusOK, &rl)
		var sysID string
		for _, r := range rl {
			if r.IsSystem {
				sysID = r.ID
				break
			}
		}
		admin.expect(http.MethodDelete, "/projects/"+p.DisplayID+"/roles/"+sysID, nil, http.StatusConflict)
	})

	t.Run("non admin cannot CRUD project roles", func(t *testing.T) {
		assignee.expect(http.MethodPost, "/projects/"+p.DisplayID+"/roles/", map[string]any{
			"name": "Attacker", "description": "x",
		}, http.StatusForbidden)
		assignee.expect(http.MethodGet, "/projects/"+p.DisplayID+"/roles/", nil, http.StatusForbidden)
	})

	t.Run("admin deletes the custom Foreman role", func(t *testing.T) {
		admin.expect(http.MethodDelete, "/projects/"+p.DisplayID+"/roles/"+foreman.ID, nil, http.StatusNoContent)
	})
}
