package sandbox0

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type SandboxObservabilityQueryOptions struct {
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Cursor    string
}

type SandboxObservabilityEventOptions struct {
	SandboxObservabilityQueryOptions
	Source    apispec.ObservabilityEventSource
	EventType apispec.SandboxObservabilityEventType
	Outcome   apispec.SandboxObservabilityOutcome
}

type SandboxObservabilityLogOptions struct {
	SandboxObservabilityQueryOptions
	ContextID string
	Stream    apispec.SandboxObservabilityLogStream
}

type SandboxObservabilityMetricOptions struct {
	SandboxObservabilityQueryOptions
	ContextID string
	Names     []string
}

type SandboxObservabilityWatchLine struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Cursor    string          `json:"cursor,omitempty"`
	Watermark string          `json:"watermark,omitempty"`
	Time      string          `json:"time,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type SandboxObservabilityStream struct {
	io.ReadCloser
	scanner *bufio.Scanner
}

func (s *SandboxObservabilityStream) Recv() (*SandboxObservabilityWatchLine, error) {
	if s == nil || s.scanner == nil {
		return nil, io.EOF
	}
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	var line SandboxObservabilityWatchLine
	if err := json.Unmarshal(s.scanner.Bytes(), &line); err != nil {
		return nil, err
	}
	return &line, nil
}

func (c *Client) ListSandboxObservabilityEvents(ctx context.Context, sandboxID string, opts *SandboxObservabilityEventOptions) (*apispec.SandboxObservabilityEventsResponse, error) {
	params := apispec.APIV1SandboxesIDObservabilityEventsGetParams{ID: sandboxID}
	applyEventObservabilityOptions(&params, opts, false)
	resp, err := c.api.APIV1SandboxesIDObservabilityEventsGet(ctx, params)
	if err != nil {
		return nil, err
	}
	if response, ok := resp.(*apispec.SuccessSandboxObservabilityEventsResponse); ok {
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return &data, nil
	}
	return nil, apiErrorFromResponse(resp)
}

func (c *Client) ListSandboxObservabilityLogs(ctx context.Context, sandboxID string, opts *SandboxObservabilityLogOptions) (*apispec.SandboxObservabilityLogsResponse, error) {
	params := apispec.APIV1SandboxesIDObservabilityLogsGetParams{ID: sandboxID}
	applyLogObservabilityOptions(&params, opts, false)
	resp, err := c.api.APIV1SandboxesIDObservabilityLogsGet(ctx, params)
	if err != nil {
		return nil, err
	}
	if response, ok := resp.(*apispec.SuccessSandboxObservabilityLogsResponse); ok {
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return &data, nil
	}
	return nil, apiErrorFromResponse(resp)
}

func (c *Client) ListSandboxObservabilityMetrics(ctx context.Context, sandboxID string, opts *SandboxObservabilityMetricOptions) (*apispec.SandboxObservabilityMetricsResponse, error) {
	params := apispec.APIV1SandboxesIDObservabilityMetricsGetParams{ID: sandboxID}
	applyMetricObservabilityOptions(&params, opts, false)
	resp, err := c.api.APIV1SandboxesIDObservabilityMetricsGet(ctx, params)
	if err != nil {
		return nil, err
	}
	if response, ok := resp.(*apispec.SuccessSandboxObservabilityMetricsResponse); ok {
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return &data, nil
	}
	return nil, apiErrorFromResponse(resp)
}

func (c *Client) WatchSandboxObservabilityEvents(ctx context.Context, sandboxID string, opts *SandboxObservabilityEventOptions) (*SandboxObservabilityStream, error) {
	values := make(url.Values)
	applyEventObservabilityValues(values, opts)
	return c.watchSandboxObservability(ctx, sandboxID, "/observability/events", values)
}

func (c *Client) WatchSandboxObservabilityLogs(ctx context.Context, sandboxID string, opts *SandboxObservabilityLogOptions) (*SandboxObservabilityStream, error) {
	values := make(url.Values)
	applyLogObservabilityValues(values, opts)
	return c.watchSandboxObservability(ctx, sandboxID, "/observability/logs", values)
}

func (c *Client) WatchSandboxObservabilityMetrics(ctx context.Context, sandboxID string, opts *SandboxObservabilityMetricOptions) (*SandboxObservabilityStream, error) {
	values := make(url.Values)
	applyMetricObservabilityValues(values, opts)
	return c.watchSandboxObservability(ctx, sandboxID, "/observability/metrics", values)
}

func (s *Sandbox) ListObservabilityEvents(ctx context.Context, opts *SandboxObservabilityEventOptions) (*apispec.SandboxObservabilityEventsResponse, error) {
	return s.client.ListSandboxObservabilityEvents(ctx, s.ID, opts)
}

func (s *Sandbox) ListLogs(ctx context.Context, opts *SandboxObservabilityLogOptions) (*apispec.SandboxObservabilityLogsResponse, error) {
	return s.client.ListSandboxObservabilityLogs(ctx, s.ID, opts)
}

func (s *Sandbox) ListMetrics(ctx context.Context, opts *SandboxObservabilityMetricOptions) (*apispec.SandboxObservabilityMetricsResponse, error) {
	return s.client.ListSandboxObservabilityMetrics(ctx, s.ID, opts)
}

func (s *Sandbox) WatchObservabilityEvents(ctx context.Context, opts *SandboxObservabilityEventOptions) (*SandboxObservabilityStream, error) {
	return s.client.WatchSandboxObservabilityEvents(ctx, s.ID, opts)
}

func (s *Sandbox) WatchLogs(ctx context.Context, opts *SandboxObservabilityLogOptions) (*SandboxObservabilityStream, error) {
	return s.client.WatchSandboxObservabilityLogs(ctx, s.ID, opts)
}

func (s *Sandbox) WatchMetrics(ctx context.Context, opts *SandboxObservabilityMetricOptions) (*SandboxObservabilityStream, error) {
	return s.client.WatchSandboxObservabilityMetrics(ctx, s.ID, opts)
}

func (c *Client) watchSandboxObservability(ctx context.Context, sandboxID, suffix string, values url.Values) (*SandboxObservabilityStream, error) {
	values.Set("watch", "true")
	reqURL, err := c.sandboxObservabilityURL(sandboxID, suffix, values)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-ndjson")
	if err := c.applyRequestEditors(ctx, req); err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		truncated := false
		if len(body) > maxErrorBodyBytes {
			body = body[:maxErrorBodyBytes]
			truncated = true
		}
		return nil, apiErrorFromHTTPResponse(resp, body, truncated)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "application/x-ndjson") {
		_ = resp.Body.Close()
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "unexpected_response",
			Message:    fmt.Sprintf("unexpected observability stream content type: %s", contentType),
		}
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	return &SandboxObservabilityStream{ReadCloser: resp.Body, scanner: scanner}, nil
}

func (c *Client) sandboxObservabilityURL(sandboxID, suffix string, values url.Values) (string, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/") + "/api/v1/sandboxes/" + url.PathEscape(sandboxID) + suffix
	baseURL.RawQuery = values.Encode()
	return baseURL.String(), nil
}

func applyEventObservabilityOptions(params *apispec.APIV1SandboxesIDObservabilityEventsGetParams, opts *SandboxObservabilityEventOptions, watch bool) {
	if opts == nil {
		if watch {
			params.Watch = apispec.NewOptBool(true)
		}
		return
	}
	applyCommonObservabilityOptions(&params.StartTime, &params.EndTime, &params.Limit, &params.Cursor, &params.Watch, opts.SandboxObservabilityQueryOptions, watch)
	if opts.Source != "" {
		params.Source = apispec.NewOptObservabilityEventSource(opts.Source)
	}
	if opts.EventType != "" {
		params.EventType = apispec.NewOptSandboxObservabilityEventType(opts.EventType)
	}
	if opts.Outcome != "" {
		params.Outcome = apispec.NewOptSandboxObservabilityOutcome(opts.Outcome)
	}
}

func applyLogObservabilityOptions(params *apispec.APIV1SandboxesIDObservabilityLogsGetParams, opts *SandboxObservabilityLogOptions, watch bool) {
	if opts == nil {
		if watch {
			params.Watch = apispec.NewOptBool(true)
		}
		return
	}
	applyCommonObservabilityOptions(&params.StartTime, &params.EndTime, &params.Limit, &params.Cursor, &params.Watch, opts.SandboxObservabilityQueryOptions, watch)
	if opts.ContextID != "" {
		params.ContextID = apispec.NewOptString(opts.ContextID)
	}
	if opts.Stream != "" {
		params.Stream = apispec.NewOptSandboxObservabilityLogStream(opts.Stream)
	}
}

func applyMetricObservabilityOptions(params *apispec.APIV1SandboxesIDObservabilityMetricsGetParams, opts *SandboxObservabilityMetricOptions, watch bool) {
	if opts == nil {
		if watch {
			params.Watch = apispec.NewOptBool(true)
		}
		return
	}
	applyCommonObservabilityOptions(&params.StartTime, &params.EndTime, &params.Limit, &params.Cursor, &params.Watch, opts.SandboxObservabilityQueryOptions, watch)
	if opts.ContextID != "" {
		params.ContextID = apispec.NewOptString(opts.ContextID)
	}
	if len(opts.Names) > 0 {
		params.Name = opts.Names
	}
}

func applyCommonObservabilityOptions(start, end *apispec.OptDateTime, limit *apispec.OptInt, cursor *apispec.OptString, watchParam *apispec.OptBool, opts SandboxObservabilityQueryOptions, watch bool) {
	if opts.StartTime != nil {
		*start = apispec.NewOptDateTime(*opts.StartTime)
	}
	if opts.EndTime != nil {
		*end = apispec.NewOptDateTime(*opts.EndTime)
	}
	if opts.Limit > 0 {
		*limit = apispec.NewOptInt(opts.Limit)
	}
	if opts.Cursor != "" {
		*cursor = apispec.NewOptString(opts.Cursor)
	}
	if watch {
		*watchParam = apispec.NewOptBool(true)
	}
}

func applyEventObservabilityValues(values url.Values, opts *SandboxObservabilityEventOptions) {
	if opts == nil {
		return
	}
	applyCommonObservabilityValues(values, opts.SandboxObservabilityQueryOptions)
	if opts.Source != "" {
		values.Set("source", string(opts.Source))
	}
	if opts.EventType != "" {
		values.Set("event_type", string(opts.EventType))
	}
	if opts.Outcome != "" {
		values.Set("outcome", string(opts.Outcome))
	}
}

func applyLogObservabilityValues(values url.Values, opts *SandboxObservabilityLogOptions) {
	if opts == nil {
		return
	}
	applyCommonObservabilityValues(values, opts.SandboxObservabilityQueryOptions)
	if opts.ContextID != "" {
		values.Set("context_id", opts.ContextID)
	}
	if opts.Stream != "" {
		values.Set("stream", string(opts.Stream))
	}
}

func applyMetricObservabilityValues(values url.Values, opts *SandboxObservabilityMetricOptions) {
	if opts == nil {
		return
	}
	applyCommonObservabilityValues(values, opts.SandboxObservabilityQueryOptions)
	if opts.ContextID != "" {
		values.Set("context_id", opts.ContextID)
	}
	for _, name := range opts.Names {
		if name != "" {
			values.Add("name", name)
		}
	}
}

func applyCommonObservabilityValues(values url.Values, opts SandboxObservabilityQueryOptions) {
	if opts.StartTime != nil {
		values.Set("start_time", opts.StartTime.UTC().Format(time.RFC3339Nano))
	}
	if opts.EndTime != nil {
		values.Set("end_time", opts.EndTime.UTC().Format(time.RFC3339Nano))
	}
	if opts.Limit > 0 {
		values.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Cursor != "" {
		values.Set("cursor", opts.Cursor)
	}
}
