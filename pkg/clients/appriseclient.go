package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// HTTPClient is an interface for making HTTP requests.
// This allows for dependency injection and mocking in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AppriseClient sends notifications via an Apprise API server.
type AppriseClient struct {
	httpClient HTTPClient
}

// AppriseRequest represents the JSON payload sent to the Apprise API.
type AppriseRequest struct {
	URLs string `json:"urls"`
	Body string `json:"body"`
}

// NewAppriseClient creates a new AppriseClient with the default HTTP client.
func NewAppriseClient() *AppriseClient {
	return &AppriseClient{
		httpClient: &http.Client{},
	}
}

// NewAppriseClientWithHTTP creates a new AppriseClient with a custom HTTP client.
// This is useful for testing with a mock implementation.
func NewAppriseClientWithHTTP(client HTTPClient) *AppriseClient {
	return &AppriseClient{
		httpClient: client,
	}
}

// SendNotification sends a notification via the Apprise API.
// The appriseURL should be in the format: http://apprise-server:port/notify/
// followed by the notification service URL (e.g., tgram://token/chatid)
func (a *AppriseClient) SendNotification(appriseURL string, body string, dryRun bool) error {
	if dryRun {
		log.Printf("DRY RUN: Would have sent notification to %s with body: %s", appriseURL, body)
		return nil
	}

	payload := AppriseRequest{
		URLs: appriseURL,
		Body: body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal apprise request: %w", err)
	}

	req, err := http.NewRequest("POST", appriseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apprise API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Notification sent successfully via Apprise")
	return nil
}
