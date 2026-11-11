//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

type projectResp struct {
	DisplayID string `json:"display_id"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"is_admin"`
}

type projectList struct {
	Items []projectResp `json:"items"`
	Total int           `json:"total"`
}

func createProject(t *testing.T, c *client, name string) projectResp {
	t.Helper()
	var p projectResp
	c.ok(http.MethodPost, "/projects/", map[string]any{
		"name":               name,
		"description":        "e2e",
		"planned_start_date": "2026-01-01",
		"planned_end_date":   "2026-12-31",
	}, http.StatusCreated, &p)
	if p.DisplayID == "" || !p.IsAdmin {
		t.Fatalf("project create response: %+v", p)
	}
	return p
}

func TestProjects(t *testing.T) {
	owner := registerAndLogin(t, "proj-owner", unique("owner")+"@example.com", "Password123!", "user")
	stranger := registerAndLogin(t, "proj-stranger", unique("stranger")+"@example.com", "Password123!", "user")

	p := createProject(t, owner, unique("Site"))

	t.Run("owner is admin and sees the project", func(t *testing.T) {
		var got projectResp
		owner.getJSON("/projects/"+p.DisplayID, http.StatusOK, &got)
		if !got.IsAdmin {
			t.Fatal("creator must be admin")
		}
	})

	t.Run("stranger gets 404 (no 403 leak)", func(t *testing.T) {
		stranger.expect(http.MethodGet, "/projects/"+p.DisplayID, nil, http.StatusNotFound)
	})

	t.Run("owner list shows is_admin=true for own project", func(t *testing.T) {
		var list projectList
		owner.getJSON("/projects/", http.StatusOK, &list)
		var seen bool
		for _, it := range list.Items {
			if it.DisplayID == p.DisplayID {
				seen = true
				if !it.IsAdmin {
					t.Fatal("list said is_admin=false for owner's own project")
				}
			}
		}
		if !seen {
			t.Fatal("owner did not see their own project in /projects/")
		}
	})

	t.Run("stranger list does not see it", func(t *testing.T) {
		var list projectList
		stranger.getJSON("/projects/", http.StatusOK, &list)
		for _, it := range list.Items {
			if it.DisplayID == p.DisplayID {
				t.Fatal("stranger saw a project they aren't a member of")
			}
		}
	})

	t.Run("stranger cannot patch", func(t *testing.T) {
		stranger.expect(http.MethodPatch, "/projects/"+p.DisplayID, map[string]any{
			"name": "pwned", "description": "", "planned_start_date": "2026-01-01", "planned_end_date": "2026-12-31",
		}, http.StatusNotFound)
	})

	t.Run("stranger cannot delete", func(t *testing.T) {
		stranger.expect(http.MethodDelete, "/projects/"+p.DisplayID, nil, http.StatusNotFound)
	})

	t.Run("owner can patch", func(t *testing.T) {
		owner.expect(http.MethodPatch, "/projects/"+p.DisplayID, map[string]any{
			"name": unique("Site-renamed"), "description": "", "planned_start_date": "2026-01-01", "planned_end_date": "2026-12-31",
		}, http.StatusOK)
	})

	t.Run("owner delete empty project", func(t *testing.T) {
		empty := createProject(t, owner, unique("to-delete"))
		owner.expect(http.MethodDelete, "/projects/"+empty.DisplayID, nil, http.StatusNoContent)
	})

	t.Run("end before start is rejected", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/", map[string]any{
			"name": "Bad dates", "description": "", "planned_start_date": "2026-06-01", "planned_end_date": "2026-01-01",
		}, http.StatusBadRequest)
	})

	t.Run("search by query filters list", func(t *testing.T) {
		needle := unique("Needle")
		createProject(t, owner, needle)
		var list projectList
		owner.getJSON("/projects/?q="+needle, http.StatusOK, &list)
		if len(list.Items) == 0 {
			t.Fatalf("search did not find the project named %q", needle)
		}
	})

	t.Run("super_admin sees any project without membership", func(t *testing.T) {
		admin := registerAndLogin(t, "root", unique("root")+"@example.com", "Password123!", "super_admin")
		admin.expect(http.MethodGet, "/projects/"+p.DisplayID, nil, http.StatusOK)
	})
}
