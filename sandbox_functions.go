package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// InvokeFunction invokes a sandbox function by name.
func (s *Sandbox) InvokeFunction(ctx context.Context, name string, request apispec.FunctionInvokeRequest) (*apispec.FunctionInvokeResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDFunctionsNameInvokePost(ctx, &request, apispec.APIV1SandboxesIDFunctionsNameInvokePostParams{
		ID:   s.ID,
		Name: name,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionInvokeResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
