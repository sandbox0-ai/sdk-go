package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ListUserSSHPublicKeys lists SSH public keys for the authenticated user.
func (c *Client) ListUserSSHPublicKeys(ctx context.Context) ([]apispec.SSHPublicKey, error) {
	resp, err := c.api.UsersMeSSHKeysGet(ctx)
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessSSHPublicKeyListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return data.SSHKeys, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateUserSSHPublicKey creates an SSH public key for the authenticated user.
func (c *Client) CreateUserSSHPublicKey(ctx context.Context, request apispec.CreateSSHPublicKeyRequest) (*apispec.SSHPublicKey, error) {
	resp, err := c.api.UsersMeSSHKeysPost(ctx, &request)
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessSSHPublicKeyResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteUserSSHPublicKey deletes an SSH public key by ID for the authenticated user.
func (c *Client) DeleteUserSSHPublicKey(ctx context.Context, id string) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.UsersMeSSHKeysIDDelete(ctx, apispec.UsersMeSSHKeysIDDeleteParams{ID: id})
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
