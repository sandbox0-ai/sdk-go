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

	"github.com/google/uuid"
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
	Source       apispec.ObservabilityEventSource
	EventType    apispec.SandboxObservabilityEventType
	Outcome      apispec.SandboxObservabilityOutcome
	ActorKind    apispec.SandboxAuditActorKind
	ActorID      string
	Action       string
	ResourceType string
	OperationID  string
	EventID      uuid.UUID
}

type SandboxObservabilityLogOptions struct {
	SandboxObservabilityQueryOptions
	ContextID string
	Stream    apispec.SandboxObservabilityLogStream
}

type SandboxObservabilityMetricOptions struct {
	StartTime   *time.Time
	EndTime     *time.Time
	Metrics     []apispec.SandboxRuntimeMetricName
	StepSeconds int
	Statistic   apispec.SandboxRuntimeMetricStatistic
	MaxPoints   int
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

func (c *Client) GetSandboxRuntimeMetrics(ctx context.Context, sandboxID string, opts *SandboxObservabilityMetricOptions) (*apispec.SandboxRuntimeMetricsResponse, error) {
	params := apispec.GetSandboxRuntimeMetricsParams{ID: sandboxID}
	applyRuntimeMetricOptions(&params, opts)
	resp, err := c.api.GetSandboxRuntimeMetrics(ctx, params)
	if err != nil {
		return nil, err
	}
	if response, ok := resp.(*apispec.SuccessSandboxRuntimeMetricsResponse); ok {
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return &data, nil
	}
	return nil, apiErrorFromResponse(resp)
}

// GetSandboxRuntimeMetricsCatalog returns the canonical runtime metric catalog.
func (c *Client) GetSandboxRuntimeMetricsCatalog(ctx context.Context, sandboxID string) (*apispec.SandboxRuntimeMetricsCatalogResponse, error) {
	resp, err := c.api.GetSandboxRuntimeMetricsCatalog(ctx, apispec.GetSandboxRuntimeMetricsCatalogParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	if response, ok := resp.(*apispec.SuccessSandboxRuntimeMetricsCatalogResponse); ok {
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return &data, nil
	}
	return nil, apiErrorFromResponse(resp)
}

// ListSandboxObservabilityMetrics is retained as an alias for the chart-ready runtime metrics API.
func (c *Client) ListSandboxObservabilityMetrics(ctx context.Context, sandboxID string, opts *SandboxObservabilityMetricOptions) (*apispec.SandboxRuntimeMetricsResponse, error) {
	return c.GetSandboxRuntimeMetrics(ctx, sandboxID, opts)
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

func (s *Sandbox) ListObservabilityEvents(ctx context.Context, opts *SandboxObservabilityEventOptions) (*apispec.SandboxObservabilityEventsResponse, error) {
	return s.client.ListSandboxObservabilityEvents(ctx, s.ID, opts)
}

func (s *Sandbox) ListLogs(ctx context.Context, opts *SandboxObservabilityLogOptions) (*apispec.SandboxObservabilityLogsResponse, error) {
	return s.client.ListSandboxObservabilityLogs(ctx, s.ID, opts)
}

func (s *Sandbox) ListMetrics(ctx context.Context, opts *SandboxObservabilityMetricOptions) (*apispec.SandboxRuntimeMetricsResponse, error) {
	return s.client.GetSandboxRuntimeMetrics(ctx, s.ID, opts)
}

// GetMetricsCatalog returns the canonical runtime metric catalog.
func (s *Sandbox) GetMetricsCatalog(ctx context.Context) (*apispec.SandboxRuntimeMetricsCatalogResponse, error) {
	return s.client.GetSandboxRuntimeMetricsCatalog(ctx, s.ID)
}

func (s *Sandbox) WatchObservabilityEvents(ctx context.Context, opts *SandboxObservabilityEventOptions) (*SandboxObservabilityStream, error) {
	return s.client.WatchSandboxObservabilityEvents(ctx, s.ID, opts)
}

func (s *Sandbox) WatchLogs(ctx context.Context, opts *SandboxObservabilityLogOptions) (*SandboxObservabilityStream, error) {
	return s.client.WatchSandboxObservabilityLogs(ctx, s.ID, opts)
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
	if opts.ActorKind != "" {
		params.ActorKind = apispec.NewOptSandboxAuditActorKind(opts.ActorKind)
	}
	if opts.ActorID != "" {
		params.ActorID = apispec.NewOptString(opts.ActorID)
	}
	if opts.Action != "" {
		params.Action = apispec.NewOptString(opts.Action)
	}
	if opts.ResourceType != "" {
		params.ResourceType = apispec.NewOptString(opts.ResourceType)
	}
	if opts.OperationID != "" {
		params.OperationID = apispec.NewOptString(opts.OperationID)
	}
	if opts.EventID != uuid.Nil {
		params.EventID = apispec.NewOptUUID(opts.EventID)
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

func applyRuntimeMetricOptions(params *apispec.GetSandboxRuntimeMetricsParams, opts *SandboxObservabilityMetricOptions) {
	if opts == nil {
		return
	}
	if opts.StartTime != nil {
		params.StartTime = apispec.NewOptDateTime(*opts.StartTime)
	}
	if opts.EndTime != nil {
		params.EndTime = apispec.NewOptDateTime(*opts.EndTime)
	}
	if len(opts.Metrics) > 0 {
		values := make([]string, 0, len(opts.Metrics))
		for _, metric := range opts.Metrics {
			values = append(values, string(metric))
		}
		params.Metrics = apispec.NewOptString(strings.Join(values, ","))
	}
	if opts.StepSeconds > 0 {
		params.StepSeconds = apispec.NewOptInt(opts.StepSeconds)
	}
	if opts.Statistic != "" {
		params.Statistic = apispec.NewOptSandboxRuntimeMetricStatistic(opts.Statistic)
	}
	if opts.MaxPoints > 0 {
		params.MaxPoints = apispec.NewOptInt(opts.MaxPoints)
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
	if opts.ActorKind != "" {
		values.Set("actor_kind", string(opts.ActorKind))
	}
	if opts.ActorID != "" {
		values.Set("actor_id", opts.ActorID)
	}
	if opts.Action != "" {
		values.Set("action", opts.Action)
	}
	if opts.ResourceType != "" {
		values.Set("resource_type", opts.ResourceType)
	}
	if opts.OperationID != "" {
		values.Set("operation_id", opts.OperationID)
	}
	if opts.EventID != uuid.Nil {
		values.Set("event_id", opts.EventID.String())
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
