package sandbox0

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// SandboxLogs is a bounded snapshot of sandbox pod logs.
type SandboxLogs struct {
	SandboxID string `json:"sandbox_id"`
	PodName   string `json:"pod_name"`
	Container string `json:"container"`
	Previous  bool   `json:"previous"`
	Logs      string `json:"logs"`
}

// SandboxLogsStream is a live sandbox pod log stream. Callers must close it.
type SandboxLogsStream struct {
	io.ReadCloser
	SandboxID string
	PodName   string
	Container string
	Previous  bool
}

// GetSandboxLogs returns a bounded snapshot of sandbox pod logs.
func (c *Client) GetSandboxLogs(ctx context.Context, sandboxID string, opts *SandboxLogsOptions) (*SandboxLogs, error) {
	req, err := c.newSandboxLogsRequest(ctx, sandboxID, opts, false)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, apiErrorFromLogHTTPResponse(resp)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" && !hasContentTypePrefix(contentType, "text/plain") {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "unexpected_response",
			Message:    fmt.Sprintf("unexpected log snapshot content type: %s", contentType),
		}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return sandboxLogsFromResponse(resp, sandboxID, opts, string(body)), nil
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
		err := apiErrorFromLogHTTPResponse(resp)
		_ = resp.Body.Close()
		return nil, err
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !hasContentTypePrefix(contentType, "text/plain") {
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
		Previous:   sandboxLogPreviousFromResponse(resp, opts),
	}, nil
}

// GetLogs returns a bounded snapshot of this sandbox's pod logs.
func (s *Sandbox) GetLogs(ctx context.Context, opts *SandboxLogsOptions) (*SandboxLogs, error) {
	return s.client.GetSandboxLogs(ctx, s.ID, opts)
}

// StreamLogs returns a streaming pod log reader for this sandbox. Callers must close it.
func (s *Sandbox) StreamLogs(ctx context.Context, opts *SandboxLogsOptions) (*SandboxLogsStream, error) {
	return s.client.StreamSandboxLogs(ctx, s.ID, opts)
}

func (c *Client) newSandboxLogsStreamRequest(ctx context.Context, sandboxID string, opts *SandboxLogsOptions) (*http.Request, error) {
	return c.newSandboxLogsRequest(ctx, sandboxID, opts, true)
}

func (c *Client) newSandboxLogsRequest(ctx context.Context, sandboxID string, opts *SandboxLogsOptions, follow bool) (*http.Request, error) {
	endpoint, err := url.JoinPath(c.baseURL, "api", "v1", "sandboxes", sandboxID, "logs")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	query.Set("follow", strconv.FormatBool(follow))
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

func apiErrorFromLogHTTPResponse(resp *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
	if readErr != nil {
		return readErr
	}
	truncated := false
	if len(body) > maxErrorBodyBytes {
		body = body[:maxErrorBodyBytes]
		truncated = true
	}
	return apiErrorFromHTTPResponse(resp, body, truncated)
}

func sandboxLogsFromResponse(resp *http.Response, sandboxID string, opts *SandboxLogsOptions, logs string) *SandboxLogs {
	return &SandboxLogs{
		SandboxID: firstNonEmpty(resp.Header.Get("X-Sandbox-ID"), sandboxID),
		PodName:   resp.Header.Get("X-Sandbox-Pod-Name"),
		Container: resp.Header.Get("X-Sandbox-Log-Container"),
		Previous:  sandboxLogPreviousFromResponse(resp, opts),
		Logs:      logs,
	}
}

func sandboxLogPreviousFromResponse(resp *http.Response, opts *SandboxLogsOptions) bool {
	previous := opts != nil && opts.Previous
	if header := resp.Header.Get("X-Sandbox-Log-Previous"); header != "" {
		if parsed, err := strconv.ParseBool(header); err == nil {
			previous = parsed
		}
	}
	return previous
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func hasContentTypePrefix(contentType, expected string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), expected)
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
