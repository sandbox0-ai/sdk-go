package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ListTemplate lists sandbox templates.
func (c *Client) ListTemplate(ctx context.Context) ([]apispec.Template, error) {
	resp, err := c.api.APIV1TemplatesGet(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Templates, nil
}

// GetTemplate retrieves a template.
func (c *Client) GetTemplate(ctx context.Context, templateID string) (*apispec.Template, error) {
	resp, err := c.api.APIV1TemplatesIDGet(ctx, apispec.APIV1TemplatesIDGetParams{ID: templateID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessTemplateResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateTemplate creates a template.
func (c *Client) CreateTemplate(ctx context.Context, request apispec.TemplateCreateRequest) (*apispec.Template, error) {
	resp, err := c.api.APIV1TemplatesPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// UpdateTemplate updates a template.
func (c *Client) UpdateTemplate(ctx context.Context, templateID string, request apispec.TemplateUpdateRequest) (*apispec.Template, error) {
	resp, err := c.api.APIV1TemplatesIDPut(ctx, &request, apispec.APIV1TemplatesIDPutParams{ID: templateID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// DeleteTemplate deletes a template.
func (c *Client) DeleteTemplate(ctx context.Context, templateID string) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.APIV1TemplatesIDDelete(ctx, apispec.APIV1TemplatesIDDeleteParams{ID: templateID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
