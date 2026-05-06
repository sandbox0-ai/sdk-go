package sandbox0

import (
	"context"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type ObservabilityQueryOptions struct {
	SandboxID string
	TraceID   string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

func (c *Client) ListObservabilityTraceSpans(ctx context.Context, opts *ObservabilityQueryOptions) ([]apispec.ObservabilityTraceSpan, error) {
	params := observabilityTraceParams(opts)
	resp, err := c.api.APIV1ObservabilityTracesGet(ctx, params)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessObservabilityTraceSpanListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, nil
		}
		return data.Spans, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

func (c *Client) ListObservabilityLogs(ctx context.Context, opts *ObservabilityQueryOptions) ([]apispec.ObservabilityLogRecord, error) {
	params := observabilityLogParams(opts)
	resp, err := c.api.APIV1ObservabilityLogsGet(ctx, params)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessObservabilityLogRecordListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, nil
		}
		return data.Logs, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

func observabilityTraceParams(opts *ObservabilityQueryOptions) apispec.APIV1ObservabilityTracesGetParams {
	params := apispec.APIV1ObservabilityTracesGetParams{}
	if opts == nil {
		return params
	}
	applyObservabilityOptions(&params.SandboxID, &params.TraceID, &params.StartTime, &params.EndTime, &params.Limit, opts)
	return params
}

func observabilityLogParams(opts *ObservabilityQueryOptions) apispec.APIV1ObservabilityLogsGetParams {
	params := apispec.APIV1ObservabilityLogsGetParams{}
	if opts == nil {
		return params
	}
	applyObservabilityOptions(&params.SandboxID, &params.TraceID, &params.StartTime, &params.EndTime, &params.Limit, opts)
	return params
}

func applyObservabilityOptions(sandboxID, traceID *apispec.OptString, startTime, endTime *apispec.OptDateTime, limit *apispec.OptInt, opts *ObservabilityQueryOptions) {
	if opts.SandboxID != "" {
		*sandboxID = apispec.NewOptString(opts.SandboxID)
	}
	if opts.TraceID != "" {
		*traceID = apispec.NewOptString(opts.TraceID)
	}
	if !opts.StartTime.IsZero() {
		*startTime = apispec.NewOptDateTime(opts.StartTime)
	}
	if !opts.EndTime.IsZero() {
		*endTime = apispec.NewOptDateTime(opts.EndTime)
	}
	if opts.Limit > 0 {
		*limit = apispec.NewOptInt(opts.Limit)
	}
}
