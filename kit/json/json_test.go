package json

import (
	stdjson "encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, http.StatusCreated, map[string]string{"status": "ok"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body map[string]string
	if err := stdjson.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body[status] = %q, want ok", body["status"])
	}
}

func TestDecodeJSON(t *testing.T) {
	var dest struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice"}`))

	if err := DecodeJSON(req, &dest); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if dest.Name != "alice" {
		t.Fatalf("Name = %q, want alice", dest.Name)
	}
}

func TestDecodeJSONRejectsInvalidBody(t *testing.T) {
	var dest map[string]string
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":`))

	if err := DecodeJSON(req, &dest); err == nil {
		t.Fatal("DecodeJSON() = nil for malformed JSON, want an error")
	}
}
