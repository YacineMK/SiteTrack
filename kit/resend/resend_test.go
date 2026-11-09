package resend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendSuccess(t *testing.T) {
	var gotPath, gotAuth, gotContentType string
	var got sendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	defer server.Close()

	client := New("test-key", "from@example.com", server.URL+"/")
	id, err := client.Send(context.Background(), Email{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		HTML:    "<p>hi</p>",
		Text:    "hi",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if id != "email_123" {
		t.Fatalf("id = %q, want email_123", id)
	}
	if gotPath != "/emails" {
		t.Fatalf("path = %q, want /emails", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if got.From != "from@example.com" || got.Subject != "Hello" || got.HTML != "<p>hi</p>" || got.Text != "hi" {
		t.Fatalf("payload = %+v", got)
	}
	if len(got.To) != 1 || got.To[0] != "to@example.com" {
		t.Fatalf("To = %v", got.To)
	}
}

func TestSendProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"message":"invalid from address"}`)
	}))
	defer server.Close()

	client := New("secret-key", "from@example.com", server.URL)
	_, err := client.Send(context.Background(), Email{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		Text:    "hi",
	})
	if err == nil {
		t.Fatal("Send() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "422") || !strings.Contains(err.Error(), "invalid from address") {
		t.Fatalf("error = %q", err)
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("error leaked api key: %q", err)
	}
}

func TestSendInvalidEmail(t *testing.T) {
	client := New("test-key", "from@example.com", "http://127.0.0.1:1")

	cases := []Email{
		{Subject: "Hello", Text: "hi"},
		{To: []string{"to@example.com"}, Text: "hi"},
		{To: []string{"to@example.com"}, Subject: "Hello"},
	}

	for _, email := range cases {
		if _, err := client.Send(context.Background(), email); !errors.Is(err, ErrInvalidEmail) {
			t.Fatalf("Send(%+v) error = %v, want ErrInvalidEmail", email, err)
		}
	}
}

func TestSendNotConfigured(t *testing.T) {
	email := Email{To: []string{"to@example.com"}, Subject: "Hello", Text: "hi"}

	if _, err := New("", "from@example.com", "http://127.0.0.1:1").Send(context.Background(), email); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("missing api key error = %v, want ErrNotConfigured", err)
	}
	if _, err := New("test-key", "", "http://127.0.0.1:1").Send(context.Background(), email); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("missing from error = %v, want ErrNotConfigured", err)
	}
}

func TestSendBadResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not-json")
	}))
	defer server.Close()

	client := New("test-key", "from@example.com", server.URL)
	_, err := client.Send(context.Background(), Email{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		Text:    "hi",
	})
	if err == nil {
		t.Fatal("Send() error = nil, want decode error")
	}
}
