package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ListTeamQuotas returns all quota policies and statuses for the current team.
func (c *Client) ListTeamQuotas(ctx context.Context) ([]apispec.TeamQuota, error) {
	resp, err := c.api.APIV1QuotasGet(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	return resp.Data, nil
}

// GetTeamQuota returns one quota policy and status for the current team.
func (c *Client) GetTeamQuota(ctx context.Context, dimension apispec.QuotaDimension) (*apispec.TeamQuota, error) {
	resp, err := c.api.APIV1QuotasDimensionGet(
		ctx,
		apispec.APIV1QuotasDimensionGetParams{Dimension: dimension},
	)
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessTeamQuotaResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
