package sandbox0

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// SandboxLogsOptions controls sandbox pod log reads.
type SandboxLogsOptions struct {
	// Container is the pod container name. When empty, the server uses its default container.
	Container string
	// TailLines limits how many log lines are returned from the end of the log.
	TailLines *int64
	// LimitBytes limits response bytes. For streams, the server applies it only when set.
	LimitBytes *int64
	// Previous returns logs for the previously terminated container instance.
	Previous bool
	// Timestamps includes Kubernetes log timestamps when available.
	Timestamps bool
	// SinceSeconds returns logs newer than this many seconds.
	SinceSeconds *int64
}

// SandboxLogsStream is a live sandbox pod log stream. Callers must close it.
type SandboxLogsStream struct {
	io.ReadCloser
	SandboxID string
	PodName   string
	Container string
}

// GetSandboxLogs returns a bounded snapshot of sandbox pod logs.
func (c *Client) GetSandboxLogs(ctx context.Context, sandboxID string, opts *SandboxLogsOptions) (*apispec.SandboxLogs, error) {
	params := apispec.APIV1SandboxesIDLogsGetParams{ID: sandboxID}
	applySandboxLogsGeneratedOptions(&params, opts)

	resp, err := c.api.APIV1SandboxesIDLogsGet(ctx, params)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxLogsResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// StreamSandboxLogs returns a streaming sandbox pod log reader. Callers must close it.
func (c *Client) StreamSandboxLogs(ctx context.Context, sandboxID string, opts *SandboxLogsOptions) (*SandboxLogsStream, error) {
	req, err := c.newSandboxLogsStreamRequest(ctx, sandboxID, opts)
	if err != nil {
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
	if contentType := resp.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "text/plain") {
		_ = resp.Body.Close()
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "unexpected_response",
			Message:    fmt.Sprintf("unexpected log stream content type: %s", contentType),
		}
	}

	return &SandboxLogsStream{
		ReadCloser: resp.Body,
		SandboxID:  resp.Header.Get("X-Sandbox-ID"),
		PodName:    resp.Header.Get("X-Sandbox-Pod-Name"),
		Container:  resp.Header.Get("X-Sandbox-Log-Container"),
	}, nil
}

// GetLogs returns a bounded snapshot of this sandbox's pod logs.
func (s *Sandbox) GetLogs(ctx context.Context, opts *SandboxLogsOptions) (*apispec.SandboxLogs, error) {
	return s.client.GetSandboxLogs(ctx, s.ID, opts)
}

// StreamLogs returns a streaming pod log reader for this sandbox. Callers must close it.
func (s *Sandbox) StreamLogs(ctx context.Context, opts *SandboxLogsOptions) (*SandboxLogsStream, error) {
	return s.client.StreamSandboxLogs(ctx, s.ID, opts)
}

func applySandboxLogsGeneratedOptions(params *apispec.APIV1SandboxesIDLogsGetParams, opts *SandboxLogsOptions) {
	if opts == nil {
		return
	}
	if container := strings.TrimSpace(opts.Container); container != "" {
		params.Container = apispec.NewOptString(container)
	}
	if opts.TailLines != nil {
		params.TailLines = apispec.NewOptInt64(*opts.TailLines)
	}
	if opts.LimitBytes != nil {
		params.LimitBytes = apispec.NewOptInt64(*opts.LimitBytes)
	}
	if opts.Previous {
		params.Previous = apispec.NewOptBool(true)
	}
	if opts.Timestamps {
		params.Timestamps = apispec.NewOptBool(true)
	}
	if opts.SinceSeconds != nil {
		params.SinceSeconds = apispec.NewOptInt64(*opts.SinceSeconds)
	}
}

func (c *Client) newSandboxLogsStreamRequest(ctx context.Context, sandboxID string, opts *SandboxLogsOptions) (*http.Request, error) {
	endpoint, err := url.JoinPath(c.baseURL, "api", "v1", "sandboxes", sandboxID, "logs")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	query.Set("follow", "true")
	applySandboxLogsQueryOptions(query, opts)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if err := c.applyRequestEditors(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func applySandboxLogsQueryOptions(query url.Values, opts *SandboxLogsOptions) {
	if opts == nil {
		return
	}
	if container := strings.TrimSpace(opts.Container); container != "" {
		query.Set("container", container)
	}
	if opts.TailLines != nil {
		query.Set("tail_lines", strconv.FormatInt(*opts.TailLines, 10))
	}
	if opts.LimitBytes != nil {
		query.Set("limit_bytes", strconv.FormatInt(*opts.LimitBytes, 10))
	}
	if opts.Previous {
		query.Set("previous", "true")
	}
	if opts.Timestamps {
		query.Set("timestamps", "true")
	}
	if opts.SinceSeconds != nil {
		query.Set("since_seconds", strconv.FormatInt(*opts.SinceSeconds, 10))
	}
}
