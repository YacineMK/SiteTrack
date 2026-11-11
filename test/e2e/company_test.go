//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

type stockResp struct {
	MaterialID string  `json:"material_id"`
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	ReorderPt  float64 `json:"reorder_point"`
	Status     string  `json:"status"`
	InTransit  float64 `json:"in_transit"`
}

type stockList struct {
	Items []stockResp `json:"items"`
}

func TestCompanyStore(t *testing.T) {
	user := registerAndLogin(t, "cs-user", unique("csu")+"@example.com", "Password123!", "user")
	mgr := registerAndLogin(t, "cs-mgr", unique("csm")+"@example.com", "Password123!", "company_store_manager")

	t.Run("plain user cannot list company stock", func(t *testing.T) {
		user.expect(http.MethodGet, "/company-store/stock", nil, http.StatusForbidden)
	})
	t.Run("plain user cannot add stock", func(t *testing.T) {
		user.expect(http.MethodPost, "/company-store/stock/additions", map[string]any{
			"material_id": "00000000-0000-0000-0000-000000000000", "quantity": 1,
		}, http.StatusForbidden)
	})

	mat := createMaterial(t, mgr, unique("CSMat"))

	t.Run("manager adds stock", func(t *testing.T) {
		mgr.expect(http.MethodPost, "/company-store/stock/additions", map[string]any{
			"material_id": mat.ID, "quantity": 10,
		}, http.StatusCreated)
	})

	t.Run("manager sets reorder point", func(t *testing.T) {
		mgr.expect(http.MethodPut, "/company-store/stock/"+mat.ID+"/reorder-point", map[string]any{
			"reorder_point": 20,
		}, http.StatusOK)
	})

	t.Run("status reflects quantity vs reorder point", func(t *testing.T) {
		var one stockResp
		mgr.getJSON("/company-store/stock/"+mat.ID, http.StatusOK, &one)
		if one.Status != "low" {
			t.Fatalf("expected low status (qty=10, reorder=20), got %q", one.Status)
		}
	})

	t.Run("list filter by status=low", func(t *testing.T) {
		var out stockList
		mgr.getJSON("/company-store/stock?status=low", http.StatusOK, &out)
		if len(out.Items) == 0 {
			t.Fatal("status=low did not include our material")
		}
		for _, s := range out.Items {
			if s.Status != "low" {
				t.Fatalf("status filter leaked %q", s.Status)
			}
		}
	})

	t.Run("out_of_stock is reported at quantity=0", func(t *testing.T) {
		m2 := createMaterial(t, mgr, unique("CSZero"))
		mgr.expect(http.MethodPut, "/company-store/stock/"+m2.ID+"/reorder-point", map[string]any{
			"reorder_point": 5,
		}, http.StatusOK)
		var one stockResp
		mgr.getJSON("/company-store/stock/"+m2.ID, http.StatusOK, &one)
		if one.Status != "out_of_stock" {
			t.Fatalf("expected out_of_stock, got %q", one.Status)
		}
	})
}
