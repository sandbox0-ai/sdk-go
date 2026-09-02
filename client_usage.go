package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ListUsageWindowsOptions filters and paginates immutable usage windows.
type ListUsageWindowsOptions struct {
	Cursor     string
	Limit      int
	WindowType string
}

// ListUsageWindows returns immutable usage windows for the current team.
func (c *Client) ListUsageWindows(
	ctx context.Context,
	opts *ListUsageWindowsOptions,
) (*apispec.UsageWindowPage, error) {
	params := apispec.APIV1UsageWindowsGetParams{}
	if opts != nil {
		if opts.Cursor != "" {
			params.Cursor = apispec.NewOptString(opts.Cursor)
		}
		if opts.Limit != 0 {
			params.Limit = apispec.NewOptInt(opts.Limit)
		}
		if opts.WindowType != "" {
			params.WindowType = apispec.NewOptString(opts.WindowType)
		}
	}

	resp, err := c.api.APIV1UsageWindowsGet(ctx, params)
	if err != nil {
		return nil, err
	}
	success, ok := resp.(*apispec.SuccessUsageWindowsResponse)
	if !ok {
		return nil, apiErrorFromResponse(resp)
	}
	data := success.Data
	return &data, nil
}
