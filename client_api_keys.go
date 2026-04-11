package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// CurrentAPIKeyIdentity is the authenticated API key identity returned by sandbox0.
type CurrentAPIKeyIdentity struct {
	ID        string
	TeamID    string
	CreatedBy *string
	Type      string
	Roles     []string
}

// GetCurrentAPIKey returns the currently authenticated API key identity.
func (c *Client) GetCurrentAPIKey(ctx context.Context) (*CurrentAPIKeyIdentity, error) {
	resp, err := c.api.APIKeysCurrentGet(ctx)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessCurrentAPIKeyResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		apiKey, ok := data.APIKey.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		identity := &CurrentAPIKeyIdentity{
			ID:     apiKey.ID,
			TeamID: apiKey.TeamID,
			Type:   apiKey.Type,
			Roles:  append([]string(nil), apiKey.Roles...),
		}
		if apiKey.CreatedBy != "" {
			createdBy := apiKey.CreatedBy
			identity.CreatedBy = &createdBy
		}
		return identity, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
