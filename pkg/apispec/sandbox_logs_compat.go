package apispec

// SandboxLogs preserves the high-level SDK snapshot shape for text/plain log responses.
type SandboxLogs struct {
	SandboxID string `json:"sandbox_id"`
	PodName   string `json:"pod_name"`
	Container string `json:"container"`
	Previous  bool   `json:"previous"`
	Logs      string `json:"logs"`
}
