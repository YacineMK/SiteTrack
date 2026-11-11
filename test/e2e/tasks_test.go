//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

type taskResp struct {
	DisplayID string `json:"display_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Blocked   bool   `json:"blocked"`
	Late      bool   `json:"late"`
}

func createTask(t *testing.T, c *client, pid, name, start, end string) taskResp {
	t.Helper()
	var task taskResp
	c.ok(http.MethodPost, "/projects/"+pid+"/tasks", map[string]any{
		"name": name, "planned_start_date": start, "planned_end_date": end,
	}, http.StatusCreated, &task)
	return task
}

func TestTasks(t *testing.T) {
	owner := registerAndLogin(t, "task-owner", unique("tasko")+"@example.com", "Password123!", "user")
	stranger := registerAndLogin(t, "task-stranger", unique("tasks")+"@example.com", "Password123!", "user")
	p := createProject(t, owner, unique("TasksProj"))

	a := createTask(t, owner, p.DisplayID, "A", "2026-01-01", "2026-01-10")
	b := createTask(t, owner, p.DisplayID, "B", "2026-01-11", "2026-01-20")

	t.Run("stranger cannot read tasks (404)", func(t *testing.T) {
		stranger.expect(http.MethodGet, "/projects/"+p.DisplayID+"/tasks", nil, http.StatusNotFound)
	})

	t.Run("get task", func(t *testing.T) {
		owner.expect(http.MethodGet, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID, nil, http.StatusOK)
	})

	t.Run("start then done then reopen", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID+"/start", nil, http.StatusOK)
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID+"/done", nil, http.StatusOK)
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID+"/reopen", nil, http.StatusOK)
	})

	t.Run("start when already started is 409", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID+"/start", nil, http.StatusConflict)
	})

	t.Run("add dependency B -> A", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+b.DisplayID+"/dependencies", map[string]any{
			"depends_on": a.DisplayID,
		}, http.StatusNoContent)
	})

	t.Run("cycle A -> B is rejected", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID+"/dependencies", map[string]any{
			"depends_on": b.DisplayID,
		}, http.StatusConflict)
	})

	t.Run("start B is blocked because A is in_progress", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+b.DisplayID+"/start", nil, http.StatusConflict)
	})

	t.Run("finish A, start B works", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID+"/done", nil, http.StatusOK)
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+b.DisplayID+"/start", nil, http.StatusOK)
	})

	t.Run("reopen A is blocked because B is in_progress", func(t *testing.T) {
		owner.expect(http.MethodPost, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID+"/reopen", nil, http.StatusConflict)
	})

	t.Run("list filters by status", func(t *testing.T) {
		var out struct {
			Items []taskResp `json:"items"`
		}
		owner.getJSON("/projects/"+p.DisplayID+"/tasks?status=in_progress", http.StatusOK, &out)
		for _, it := range out.Items {
			if it.Status != "in_progress" {
				t.Fatalf("status filter leaked %q", it.Status)
			}
		}
	})

	t.Run("update task name", func(t *testing.T) {
		owner.expect(http.MethodPatch, "/projects/"+p.DisplayID+"/tasks/"+a.DisplayID, map[string]any{
			"name": "A renamed", "planned_start_date": "2026-01-01", "planned_end_date": "2026-01-15",
		}, http.StatusOK)
	})

	t.Run("delete task", func(t *testing.T) {
		tmp := createTask(t, owner, p.DisplayID, "tmp", "2026-02-01", "2026-02-02")
		owner.expect(http.MethodDelete, "/projects/"+p.DisplayID+"/tasks/"+tmp.DisplayID, nil, http.StatusNoContent)
	})

	t.Run("remove dependency", func(t *testing.T) {
		owner.expect(http.MethodDelete, "/projects/"+p.DisplayID+"/tasks/"+b.DisplayID+"/dependencies/"+a.DisplayID, nil, http.StatusNoContent)
	})
}
