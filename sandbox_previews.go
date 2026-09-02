package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// CreatePreview creates a short-lived private browser preview for a sandbox loopback port.
func (s *Sandbox) CreatePreview(ctx context.Context, request apispec.SandboxPreviewCreateRequest) (*apispec.SandboxPreviewGrant, error) {
	resp, err := s.client.api.APIV1SandboxesIDPreviewsPost(ctx, &request, apispec.APIV1SandboxesIDPreviewsPostParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxPreviewResponse:
		grant := response.Data
		return &grant, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// RenewPreview extends an active preview grant without changing its browser session.
func (s *Sandbox) RenewPreview(ctx context.Context, previewID string, request apispec.SandboxPreviewRenewRequest) (*apispec.SandboxPreviewGrant, error) {
	resp, err := s.client.api.APIV1SandboxesIDPreviewsPreviewIDPut(ctx, &request, apispec.APIV1SandboxesIDPreviewsPreviewIDPutParams{
		ID:        s.ID,
		PreviewID: previewID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxPreviewResponse:
		grant := response.Data
		return &grant, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// RevokePreview immediately invalidates a preview grant.
func (s *Sandbox) RevokePreview(ctx context.Context, previewID string) error {
	resp, err := s.client.api.APIV1SandboxesIDPreviewsPreviewIDDelete(ctx, apispec.APIV1SandboxesIDPreviewsPreviewIDDeleteParams{
		ID:        s.ID,
		PreviewID: previewID,
	})
	if err != nil {
		return err
	}
	if _, ok := resp.(*apispec.SuccessMessageResponse); !ok {
		return apiErrorFromResponse(resp)
	}
	return nil
}
