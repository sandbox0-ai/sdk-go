package sandbox0

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const defaultTemplateReadyPollInterval = 500 * time.Millisecond

// CreateTemplateFromSandboxOptions configures an asynchronous template build request.
type CreateTemplateFromSandboxOptions struct {
	IdempotencyKey string
}

// WaitTemplateReadyOptions configures template readiness polling.
type WaitTemplateReadyOptions struct {
	PollInterval time.Duration
}

// TemplateCreationFailedError reports a server-side template image build failure.
type TemplateCreationFailedError struct {
	TemplateID string
	Stage      apispec.TemplateCreationStatusStage
	Reason     string
	Message    string
}

func (e *TemplateCreationFailedError) Error() string {
	if e == nil {
		return "<nil>"
	}
	detail := e.Message
	if detail == "" {
		detail = e.Reason
	}
	if detail == "" {
		detail = "template creation failed"
	}
	return fmt.Sprintf("template %q creation failed during %s: %s", e.TemplateID, e.Stage, detail)
}

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

// CreateTemplateFromSandbox starts an asynchronous template build from a sandbox root filesystem.
func (c *Client) CreateTemplateFromSandbox(
	ctx context.Context,
	request apispec.TemplateFromSandboxCreateRequest,
	opts *CreateTemplateFromSandboxOptions,
) (*apispec.Template, error) {
	params := apispec.APIV1TemplatesFromSandboxPostParams{}
	if opts != nil && opts.IdempotencyKey != "" {
		params.IdempotencyKey = apispec.NewOptString(opts.IdempotencyKey)
	}
	resp, err := c.api.APIV1TemplatesFromSandboxPost(ctx, &request, params)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessTemplateResponseHeaders:
		data, ok := response.Response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// WaitTemplateReady polls a template until its asynchronous creation is ready or failed.
//
// Traditional image-based templates omit status.creation and are treated as ready.
// Cancel or time out ctx to stop waiting; this does not cancel the server-side build.
func (c *Client) WaitTemplateReady(
	ctx context.Context,
	templateID string,
	opts *WaitTemplateReadyOptions,
) (*apispec.Template, error) {
	pollInterval := defaultTemplateReadyPollInterval
	if opts != nil && opts.PollInterval > 0 {
		pollInterval = opts.PollInterval
	}

	for {
		tpl, err := c.GetTemplate(ctx, templateID)
		if err != nil {
			return nil, err
		}
		status, ok := tpl.Status.Get()
		if !ok {
			return tpl, nil
		}
		creation, ok := status.Creation.Get()
		if !ok {
			return tpl, nil
		}
		switch creation.State {
		case apispec.TemplateCreationStatusStateReady:
			return tpl, nil
		case apispec.TemplateCreationStatusStateFailed:
			return nil, &TemplateCreationFailedError{
				TemplateID: templateID,
				Stage:      creation.Stage,
				Reason:     creation.Reason.Or(""),
				Message:    creation.Message.Or(""),
			}
		case apispec.TemplateCreationStatusStateCreating:
		default:
			return nil, fmt.Errorf("template %q has unknown creation state %q", templateID, creation.State)
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// UpdateTemplate updates a template.
func (c *Client) UpdateTemplate(ctx context.Context, templateID string, request apispec.TemplateUpdateRequest) (*apispec.Template, error) {
	resp, err := c.api.APIV1TemplatesIDPut(ctx, &request, apispec.APIV1TemplatesIDPutParams{ID: templateID})
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
	case *apispec.ErrorEnvelopeHeaders:
		return nil, apiErrorFromEnvelope(
			http.StatusConflict,
			&response.Response,
			response.RetryAfter.Or(0),
		)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteTemplate deletes a template.
func (c *Client) DeleteTemplate(ctx context.Context, templateID string) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.APIV1TemplatesIDDelete(ctx, apispec.APIV1TemplatesIDDeleteParams{ID: templateID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
