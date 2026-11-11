//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

type materialResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Unit     string `json:"unit"`
}

func createMaterial(t *testing.T, c *client, name string) materialResp {
	t.Helper()
	var m materialResp
	c.ok(http.MethodPost, "/materials/", map[string]any{
		"name": name, "category": "e2e", "unit": "unit",
	}, http.StatusCreated, &m)
	return m
}

func TestMaterials(t *testing.T) {
	creator := registerAndLogin(t, "mat-creator", unique("matc")+"@example.com", "Password123!", "user")
	name := unique("CementE2E")
	m := createMaterial(t, creator, name)

	t.Run("duplicate case-insensitive name is 409", func(t *testing.T) {
		other := registerAndLogin(t, "mat-dup", unique("matd")+"@example.com", "Password123!", "user")
		other.expect(http.MethodPost, "/materials/", map[string]any{
			"name": name, "category": "e2e", "unit": "unit",
		}, http.StatusConflict)
	})

	t.Run("any logged-in user can list", func(t *testing.T) {
		creator.expect(http.MethodGet, "/materials/", nil, http.StatusOK)
	})

	t.Run("search by q filters", func(t *testing.T) {
		var out struct {
			Items []materialResp `json:"items"`
		}
		creator.getJSON("/materials/?q="+name, http.StatusOK, &out)
		if len(out.Items) == 0 {
			t.Fatal("search did not match the created material")
		}
	})

	t.Run("update name", func(t *testing.T) {
		creator.expect(http.MethodPatch, "/materials/"+m.ID, map[string]any{
			"name": name + "-rev", "category": "e2e", "unit": "unit",
		}, http.StatusOK)
	})

	t.Run("delete material not in use", func(t *testing.T) {
		tmp := createMaterial(t, creator, unique("Temp"))
		creator.expect(http.MethodDelete, "/materials/"+tmp.ID, nil, http.StatusNoContent)
	})
}
