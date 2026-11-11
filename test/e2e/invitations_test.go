//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestInvitationFlow exercises the full invite → SSE → accept path:
//   admin invites invitee → outbox worker fans out → SSE delivers the
//   notification → invitee reads the notification id from the stream and
//   POSTs /invitations/{id}/accept → invitee becomes a member.
func TestInvitationFlow(t *testing.T) {
	admin := registerAndLogin(t, "inv-admin", unique("invad")+"@example.com", "Password123!", "user")
	invitee := registerAndLogin(t, "inv-invitee", unique("invit")+"@example.com", "Password123!", "user")

	p := createProject(t, admin, unique("InvFlow"))

	// Open the SSE stream BEFORE the invite so we don't miss the event.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	notifID := subscribeAndWait(t, ctx, invitee, 5*time.Second)
	go func() {
		// small delay so the stream handler has subscribed before the publish.
		time.Sleep(200 * time.Millisecond)
		var inv invitationResp
		admin.ok(http.MethodPost, "/projects/"+p.DisplayID+"/invitations", map[string]any{
			"email": invitee.lookupEmail(t),
		}, http.StatusCreated, &inv)
	}()

	id := <-notifID
	if id == "" {
		t.Fatal("did not receive notification via SSE within 5s")
	}

	t.Run("invitee accepts via notification id", func(t *testing.T) {
		invitee.expect(http.MethodPost, "/invitations/"+id+"/accept", nil, http.StatusNoContent)
	})

	t.Run("invitee is now a member", func(t *testing.T) {
		var ml membersList
		admin.getJSON("/projects/"+p.DisplayID+"/members", http.StatusOK, &ml)
		email := invitee.lookupEmail(t)
		var found bool
		for _, m := range ml.Items {
			if m.Email == email {
				found = true
			}
		}
		if !found {
			t.Fatal("invitee not in members list after accept")
		}
	})

	t.Run("second accept on same invitation is 404 (one-time)", func(t *testing.T) {
		invitee.expect(http.MethodPost, "/invitations/"+id+"/accept", nil, http.StatusNotFound)
	})

	t.Run("pending list is empty for this project after acceptance", func(t *testing.T) {
		var out struct {
			Items []invitationResp `json:"items"`
		}
		admin.getJSON("/projects/"+p.DisplayID+"/invitations", http.StatusOK, &out)
		if len(out.Items) != 0 {
			t.Fatalf("pending list should be empty, got %d", len(out.Items))
		}
	})

	t.Run("admin can list a fresh pending invitation", func(t *testing.T) {
		other := registerAndLogin(t, "inv-other", unique("invot")+"@example.com", "Password123!", "user")
		admin.expect(http.MethodPost, "/projects/"+p.DisplayID+"/invitations", map[string]any{
			"email": other.lookupEmail(t),
		}, http.StatusCreated)
		var out struct {
			Items []invitationResp `json:"items"`
		}
		admin.getJSON("/projects/"+p.DisplayID+"/invitations", http.StatusOK, &out)
		if len(out.Items) != 1 {
			t.Fatalf("expected 1 pending invitation, got %d", len(out.Items))
		}
	})
}

// subscribeAndWait opens an SSE stream and returns a channel that fires once
// with the first notification id it receives, or "" on timeout.
func subscribeAndWait(t *testing.T, ctx context.Context, c *client, timeout time.Duration) <-chan string {
	t.Helper()
	ch := make(chan string, 1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/notifications/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("stream dial: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("stream status = %d", resp.StatusCode)
	}

	go func() {
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		deadline := time.Now().Add(timeout)
		for scanner.Scan() {
			if time.Now().After(deadline) {
				ch <- ""
				return
			}
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var payload struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err == nil {
				if payload.Type == "app_invite" {
					ch <- payload.ID
					return
				}
			}
		}
		ch <- ""
	}()
	return ch
}
