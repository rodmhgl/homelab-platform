package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for the Argo CD API
type Client struct {
	serverURL  string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Argo CD API client
func NewClient(serverURL, token string) *Client {
	return &Client{
		serverURL: strings.TrimSuffix(serverURL, "/"),
		token:     token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest executes an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.serverURL, path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// ListApplications retrieves all applications from Argo CD
func (c *Client) ListApplications(ctx context.Context) (*ApplicationList, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/applications", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var appList ApplicationList
	if err := json.NewDecoder(resp.Body).Decode(&appList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &appList, nil
}

// GetApplication retrieves a specific application by name
func (c *Client) GetApplication(ctx context.Context, name string) (*Application, error) {
	path := fmt.Sprintf("/api/v1/applications/%s", name)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("application %s not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var app Application
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// SyncApplication triggers a sync for a specific application
func (c *Client) SyncApplication(ctx context.Context, name string, req *SyncRequest) (*Application, error) {
	path := fmt.Sprintf("/api/v1/applications/%s/sync", name)

	var body io.Reader
	if req != nil {
		jsonData, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal sync request: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var app Application
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}
