//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

type deliveryLine struct {
	MaterialID string  `json:"material_id"`
	Quantity   float64 `json:"quantity"`
}

type deliveryResp struct {
	ID       string         `json:"id"`
	Status   string         `json:"status"`
	Lines    []deliveryLine `json:"lines"`
	SenderID *string        `json:"sender_id"`
}

type deliveryList struct {
	Items []deliveryResp `json:"items"`
}

func TestDeliveries(t *testing.T) {
	mgr := registerAndLogin(t, "dlv-mgr", unique("dmgr")+"@example.com", "Password123!", "company_store_manager")
	owner := registerAndLogin(t, "dlv-owner", unique("downer")+"@example.com", "Password123!", "user")

	p := createProject(t, owner, unique("DeliveriesProj"))
	mat := createMaterial(t, mgr, unique("DMat"))

	// Seed company stock.
	mgr.expect(http.MethodPost, "/company-store/stock/additions", map[string]any{
		"material_id": mat.ID, "quantity": 100,
	}, http.StatusCreated)

	var dlv deliveryResp
	t.Run("manager creates delivery to project", func(t *testing.T) {
		mgr.ok(http.MethodPost, "/company-store/deliveries", map[string]any{
			"receiver_project": p.DisplayID,
			"lines":            []deliveryLine{{MaterialID: mat.ID, Quantity: 10}},
		}, http.StatusCreated, &dlv)
		if dlv.Status != "in_transit" || len(dlv.Lines) == 0 {
			t.Fatalf("bad delivery: %+v", dlv)
		}
	})

	t.Run("insufficient stock is 409", func(t *testing.T) {
		mgr.expect(http.MethodPost, "/company-store/deliveries", map[string]any{
			"receiver_project": p.DisplayID,
			"lines":            []deliveryLine{{MaterialID: mat.ID, Quantity: 1e9}},
		}, http.StatusConflict)
	})

	t.Run("non-member cannot see the delivery", func(t *testing.T) {
		stranger := registerAndLogin(t, "dlv-stranger", unique("dst")+"@example.com", "Password123!", "user")
		stranger.expect(http.MethodGet, "/deliveries/"+dlv.ID, nil, http.StatusNotFound)
	})

	t.Run("project member records delivery", func(t *testing.T) {
		owner.expect(http.MethodPost, "/deliveries/"+dlv.ID+"/record", nil, http.StatusOK)
	})

	t.Run("record twice is 409", func(t *testing.T) {
		owner.expect(http.MethodPost, "/deliveries/"+dlv.ID+"/record", nil, http.StatusConflict)
	})

	// New delivery to test reject.
	var dlv2 deliveryResp
	t.Run("new delivery then reject", func(t *testing.T) {
		mgr.ok(http.MethodPost, "/company-store/deliveries", map[string]any{
			"receiver_project": p.DisplayID,
			"lines":            []deliveryLine{{MaterialID: mat.ID, Quantity: 5}},
		}, http.StatusCreated, &dlv2)
		owner.expect(http.MethodPost, "/deliveries/"+dlv2.ID+"/reject", nil, http.StatusOK)
	})

	// Cancel while in transit.
	t.Run("sender cancel while in transit", func(t *testing.T) {
		var d deliveryResp
		mgr.ok(http.MethodPost, "/company-store/deliveries", map[string]any{
			"receiver_project": p.DisplayID,
			"lines":            []deliveryLine{{MaterialID: mat.ID, Quantity: 3}},
		}, http.StatusCreated, &d)
		mgr.expect(http.MethodPost, "/deliveries/"+d.ID+"/cancel", nil, http.StatusOK)
	})

	t.Run("project list shows our incoming deliveries", func(t *testing.T) {
		var out deliveryList
		owner.getJSON("/projects/"+p.DisplayID+"/deliveries", http.StatusOK, &out)
		if len(out.Items) == 0 {
			t.Fatal("project deliveries list empty")
		}
	})

	t.Run("project member cannot cancel manager's delivery (not sender)", func(t *testing.T) {
		var d deliveryResp
		mgr.ok(http.MethodPost, "/company-store/deliveries", map[string]any{
			"receiver_project": p.DisplayID,
			"lines":            []deliveryLine{{MaterialID: mat.ID, Quantity: 2}},
		}, http.StatusCreated, &d)
		owner.expect(http.MethodPost, "/deliveries/"+d.ID+"/cancel", nil, http.StatusNotFound)
	})
}
