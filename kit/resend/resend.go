package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNotConfigured = errors.New("resend: not configured")
	ErrInvalidEmail  = errors.New("resend: invalid email")
)

type Email struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

type Client struct {
	apiKey  string
	from    string
	baseURL string
	http    *http.Client
}

func New(apiKey, from, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.resend.com"
	}

	return &Client{
		apiKey:  apiKey,
		from:    from,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type sendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

type sendResponse struct {
	ID string `json:"id"`
}

func (c *Client) Send(ctx context.Context, email Email) (string, error) {
	if c.apiKey == "" || c.from == "" {
		return "", ErrNotConfigured
	}
	if len(email.To) == 0 || strings.TrimSpace(email.Subject) == "" || (email.HTML == "" && email.Text == "") {
		return "", ErrInvalidEmail
	}

	body, err := json.Marshal(sendRequest{
		From:    c.from,
		To:      email.To,
		Subject: email.Subject,
		HTML:    email.HTML,
		Text:    email.Text,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("resend: %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}

	var result sendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("resend: decode response: %w", err)
	}
	if result.ID == "" {
		return "", errors.New("resend: missing email id")
	}

	return result.ID, nil
}
