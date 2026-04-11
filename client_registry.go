package sandbox0

import (
	"context"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetRegistryCredentials returns short-lived credentials for the team image registry.
func (c *Client) GetRegistryCredentials(ctx context.Context, targetImage string) (*apispec.RegistryCredentials, error) {
	request := apispec.RegistryCredentialsRequest{}
	if strings.TrimSpace(targetImage) != "" {
		request.TargetImage = apispec.NewOptString(targetImage)
	}
	resp, err := c.api.APIV1RegistryCredentialsPost(ctx, apispec.NewOptRegistryCredentialsRequest(request))
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRegistryCredentialsResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
