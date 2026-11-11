//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

type projectStockResp struct {
	MaterialID string  `json:"material_id"`
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
}

type projectStockList struct {
	Items []projectStockResp `json:"items"`
}

func TestProjectStore(t *testing.T) {
	owner := registerAndLogin(t, "ps-owner", unique("pso")+"@example.com", "Password123!", "user")
	stranger := registerAndLogin(t, "ps-stranger", unique("pss")+"@example.com", "Password123!", "user")
	p := createProject(t, owner, unique("PSProj"))

	t.Run("owner can list project store (empty)", func(t *testing.T) {
		var out projectStockList
		owner.getJSON("/projects/"+p.DisplayID+"/store/items", http.StatusOK, &out)
	})

	t.Run("stranger gets 404", func(t *testing.T) {
		stranger.expect(http.MethodGet, "/projects/"+p.DisplayID+"/store/items", nil, http.StatusNotFound)
	})
}
