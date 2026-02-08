package sandbox0

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var nullMapKeys = map[string]struct{}{
	"annotations":  {},
	"envVars":      {},
	"env_vars":     {},
	"labels":       {},
	"matchLabels":  {},
	"nodeSelector": {},
}

var nullArrayKeys = map[string]struct{}{
	"allowedCidrs":                              {},
	"allowedDomains":                            {},
	"allowedPorts":                              {},
	"allowedTeams":                              {},
	"api_keys":                                  {},
	"args":                                      {},
	"candidates":                                {},
	"command":                                   {},
	"conditions":                                {},
	"contexts":                                  {},
	"data":                                      {},
	"deniedCidrs":                               {},
	"deniedDomains":                             {},
	"deniedPorts":                               {},
	"drop":                                      {},
	"entries":                                   {},
	"env":                                       {},
	"identities":                                {},
	"matchExpressions":                          {},
	"matchFields":                               {},
	"members":                                   {},
	"mounts":                                    {},
	"namespaces":                                {},
	"nodeSelectorTerms":                         {},
	"preferredDuringSchedulingIgnoredDuringExecution": {},
	"providers":                                 {},
	"requiredDuringSchedulingIgnoredDuringExecution":  {},
	"roles":                                     {},
	"sidecars":                                  {},
	"tags":                                      {},
	"teams":                                     {},
	"templates":                                 {},
	"tolerations":                               {},
	"values":                                    {},
}

func normalizeNullMapResponse(_ context.Context, resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if len(body) == 0 {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	if !normalizeNullMaps(&payload) {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	normalized, err := json.Marshal(payload)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	resp.Body = io.NopCloser(bytes.NewReader(normalized))
	resp.ContentLength = int64(len(normalized))
	resp.Header.Set("Content-Length", strconv.Itoa(len(normalized)))
	return nil
}

func normalizeNullMaps(payload *any) bool {
	switch value := (*payload).(type) {
	case map[string]any:
		changed := false
		for key, raw := range value {
			if raw == nil && isNullMapKey(key) {
				value[key] = map[string]any{}
				changed = true
				continue
			}
			if raw == nil && isNullArrayKey(key) {
				value[key] = []any{}
				changed = true
				continue
			}
			if isNullMapKey(key) {
				if strKeyed, ok := raw.(map[string]any); ok {
					changed = normalizeNullStringMap(strKeyed) || changed
				}
			}
			if normalizeNullMaps(&raw) {
				value[key] = raw
				changed = true
			}
		}
		return changed
	case []any:
		changed := false
		for i, raw := range value {
			if normalizeNullMaps(&raw) {
				value[i] = raw
				changed = true
			}
		}
		return changed
	default:
		return false
	}
}

func isNullMapKey(key string) bool {
	_, ok := nullMapKeys[key]
	return ok
}

func isNullArrayKey(key string) bool {
	_, ok := nullArrayKeys[key]
	return ok
}

func normalizeNullStringMap(value map[string]any) bool {
	changed := false
	for key, raw := range value {
		if raw == nil {
			value[key] = ""
			changed = true
		}
	}
	return changed
}
