package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// List lists sandbox templates.
func (c *Client) ListTemplate(ctx context.Context) ([]apispec.SandboxTemplate, error) {
	resp, err := c.api.GetApiV1TemplatesWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil && resp.JSON200.Data.Templates != nil {
		return *resp.JSON200.Data.Templates, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Get retrieves a template.
func (c *Client) GetTemplate(ctx context.Context, templateID string) (*apispec.SandboxTemplate, error) {
	resp, err := c.api.GetApiV1TemplatesIdWithResponse(ctx, apispec.TemplateID(templateID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	if resp.JSON404 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON404)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Create creates a template.
func (c *Client) CreateTemplate(ctx context.Context, template apispec.SandboxTemplate) (*apispec.SandboxTemplate, error) {
	resp, err := c.api.PostApiV1TemplatesWithResponse(ctx, template)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		return resp.JSON201.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Update updates a template.
func (c *Client) UpdateTemplate(ctx context.Context, templateID string, template apispec.SandboxTemplate) (*apispec.SandboxTemplate, error) {
	resp, err := c.api.PutApiV1TemplatesIdWithResponse(ctx, apispec.TemplateID(templateID), template)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Delete deletes a template.
func (c *Client) DeleteTemplate(ctx context.Context, templateID string) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.DeleteApiV1TemplatesIdWithResponse(ctx, apispec.TemplateID(templateID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// WarmPool triggers warm pool creation for a template.
func (c *Client) WarmPoolTemplate(ctx context.Context, templateID string, request apispec.WarmPoolRequest) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.PostApiV1TemplatesIdPoolWarmWithResponse(ctx, apispec.TemplateID(templateID), request)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
