package sandbox0

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetCurrentAPIKey retrieves the API key represented by the current bearer token.
func (c *Client) GetCurrentAPIKey(ctx context.Context) (*apispec.APIKey, error) {
	requestURL := strings.TrimRight(c.baseURL, "/") + "/api-keys/current"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if err := c.applyRequestEditors(ctx, req); err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sandbox0 api key introspection failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var envelope struct {
		Data struct {
			APIKey struct {
				ID        string   `json:"id"`
				TeamID    string   `json:"team_id"`
				CreatedBy string   `json:"created_by"`
				Roles     []string `json:"roles"`
				IsActive  bool     `json:"is_active"`
				ExpiresAt string   `json:"expires_at"`
			} `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	var expiresAt time.Time
	if raw := strings.TrimSpace(envelope.Data.APIKey.ExpiresAt); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err == nil {
			expiresAt = parsed
		}
	}
	return &apispec.APIKey{
		ID:        envelope.Data.APIKey.ID,
		TeamID:    envelope.Data.APIKey.TeamID,
		CreatedBy: envelope.Data.APIKey.CreatedBy,
		Roles:     append([]string(nil), envelope.Data.APIKey.Roles...),
		IsActive:  envelope.Data.APIKey.IsActive,
		ExpiresAt: expiresAt,
	}, nil
}
