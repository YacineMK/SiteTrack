//go:build e2e

// escalation_test.go collects one subtest per privilege escalation attempt.
// Each must be denied. If one passes, the API has a real security bug.
package e2e

import (
	"net/http"
	"testing"
)

func TestEscalation(t *testing.T) {
	admin := registerAndLogin(t, "esc-admin", unique("eadmin")+"@example.com", "Password123!", "user")
	stranger := registerAndLogin(t, "esc-stranger", unique("estranger")+"@example.com", "Password123!", "user")
	p := createProject(t, admin, unique("EscProj"))
	ta := createTask(t, admin, p.DisplayID, "T1", "2026-01-01", "2026-01-10")

	t.Run("read someone else's project 404", func(t *testing.T) {
		stranger.expect(http.MethodGet, "/projects/"+p.DisplayID, nil, http.StatusNotFound)
	})
	t.Run("read task of a project you aren't in 404", func(t *testing.T) {
		stranger.expect(http.MethodGet, "/projects/"+p.DisplayID+"/tasks/"+ta.DisplayID, nil, http.StatusNotFound)
	})
	t.Run("non-admin member cannot invite", func(t *testing.T) {
		member := registerAndLogin(t, "esc-member", unique("em")+"@example.com", "Password123!", "user")
		joinProject(t, member, p.DisplayID, false)
		member.expect(http.MethodPost, "/projects/"+p.DisplayID+"/invitations", map[string]any{
			"email": "someone@example.com",
		}, http.StatusForbidden)
	})
	t.Run("non-super_admin cannot POST /roles", func(t *testing.T) {
		stranger.expect(http.MethodPost, "/roles/", map[string]any{
			"name": "evil", "description": "x",
		}, http.StatusForbidden)
	})
	t.Run("non-super_admin cannot PATCH user role", func(t *testing.T) {
		var me struct {
			ID string `json:"id"`
		}
		admin.getJSON("/auth/me", http.StatusOK, &me)
		stranger.expect(http.MethodPatch, "/users/"+me.ID+"/role", map[string]any{
			"role": "super_admin",
		}, http.StatusForbidden)
	})
	t.Run("logged-out token is rejected", func(t *testing.T) {
		victim := registerAndLogin(t, "esc-victim", unique("ev")+"@example.com", "Password123!", "user")
		victim.expect(http.MethodPost, "/auth/logout", nil, http.StatusNoContent)
		victim.expect(http.MethodGet, "/auth/me", nil, http.StatusUnauthorized)
	})
	t.Run("non-sender cannot cancel delivery", func(t *testing.T) {
		mgr := registerAndLogin(t, "esc-mgr", unique("emgr")+"@example.com", "Password123!", "company_store_manager")
		mat := createMaterial(t, mgr, unique("EscMat"))
		mgr.expect(http.MethodPost, "/company-store/stock/additions", map[string]any{
			"material_id": mat.ID, "quantity": 5,
		}, http.StatusCreated)
		var d deliveryResp
		mgr.ok(http.MethodPost, "/company-store/deliveries", map[string]any{
			"receiver_project": p.DisplayID,
			"lines":            []deliveryLine{{MaterialID: mat.ID, Quantity: 1}},
		}, http.StatusCreated, &d)
		admin.expect(http.MethodPost, "/deliveries/"+d.ID+"/cancel", nil, http.StatusNotFound)
	})
	t.Run("cycle dependency rejected", func(t *testing.T) {
		tb := createTask(t, admin, p.DisplayID, "T2", "2026-01-11", "2026-01-20")
		admin.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+tb.DisplayID+"/dependencies", map[string]any{
			"depends_on": ta.DisplayID,
		}, http.StatusNoContent)
		admin.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+ta.DisplayID+"/dependencies", map[string]any{
			"depends_on": tb.DisplayID,
		}, http.StatusConflict)
	})
	t.Run("empty role is rejected", func(t *testing.T) {
		c := newClient(t, "esc-typo")
		c.expect(http.MethodPost, "/auth/register", map[string]any{
			"name": "T", "email": unique("typo") + "@example.com", "password": "Password123!", "app_role": "",
		}, http.StatusBadRequest)
	})
	t.Run("cross-project role on member — the honest one", func(t *testing.T) {
		// Admin A owns p; admin B owns p2. Admin A creates a custom role in p.
		// Tries to assign it to a member of p2. This SHOULD fail; if it
		// succeeds, the repo is not scope-checking the role_id against the
		// member's project. Without a trigger, that check lives in SQL.
		adminB := registerAndLogin(t, "esc-adminB", unique("adminb")+"@example.com", "Password123!", "user")
		memberB := registerAndLogin(t, "esc-memberB", unique("memberb")+"@example.com", "Password123!", "user")
		p2 := createProject(t, adminB, unique("EscP2"))
		joinProject(t, memberB, p2.DisplayID, false)

		// Role from p1 (admin's project):
		var pr roleResp
		admin.ok(http.MethodPost, "/projects/"+p.DisplayID+"/roles/", map[string]any{
			"name": "Spy", "description": "cross-scope",
		}, http.StatusCreated, &pr)

		// adminB tries to PATCH member in p2 with a role_id from p1.
		// The handler must reject it because the role's project_id != p2.
		var ml membersList
		adminB.getJSON("/projects/"+p2.DisplayID+"/members", http.StatusOK, &ml)
		var memBID string
		for _, m := range ml.Items {
			if m.Email == memberB.lookupEmail(t) {
				memBID = m.UserID
			}
		}
		adminB.expect(http.MethodPatch, "/projects/"+p2.DisplayID+"/members/"+memBID, map[string]any{
			"role_id": pr.ID,
		}, http.StatusNotFound)
	})
}
