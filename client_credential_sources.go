package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ListCredentialSources lists team-scoped credential sources.
func (c *Client) ListCredentialSources(ctx context.Context) ([]apispec.CredentialSourceMetadata, error) {
	resp, err := c.api.APIV1CredentialSourcesGet(ctx)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessCredentialSourceListResponse:
		return response.Data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetCredentialSource retrieves one credential source by name.
func (c *Client) GetCredentialSource(ctx context.Context, name string) (*apispec.CredentialSourceMetadata, error) {
	resp, err := c.api.APIV1CredentialSourcesNameGet(ctx, apispec.APIV1CredentialSourcesNameGetParams{Name: name})
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessCredentialSourceResponse:
		data := response.Data
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateCredentialSource creates a credential source.
func (c *Client) CreateCredentialSource(ctx context.Context, request apispec.CredentialSourceWriteRequest) (*apispec.CredentialSourceMetadata, error) {
	resp, err := c.api.APIV1CredentialSourcesPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessCredentialSourceResponse:
		data := response.Data
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateCredentialSource updates a credential source.
func (c *Client) UpdateCredentialSource(ctx context.Context, name string, request apispec.CredentialSourceWriteRequest) (*apispec.CredentialSourceMetadata, error) {
	request.Name = name

	resp, err := c.api.APIV1CredentialSourcesNamePut(ctx, &request, apispec.APIV1CredentialSourcesNamePutParams{Name: name})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessCredentialSourceResponse:
		data := response.Data
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteCredentialSource deletes a credential source.
func (c *Client) DeleteCredentialSource(ctx context.Context, name string) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.APIV1CredentialSourcesNameDelete(ctx, apispec.APIV1CredentialSourcesNameDeleteParams{Name: name})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessMessageResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
