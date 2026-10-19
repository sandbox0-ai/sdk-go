package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestExternalMountClaimAndPausedDetails(t *testing.T) {
	binding := apispec.ExternalMountBinding{SourceRef: "customer-data", MountPath: "/data", Prefix: apispec.NewOptString("runs/"), Access: apispec.NewOptExternalMountBindingAccess(apispec.ExternalMountBindingAccessReadWrite)}
	wire := map[string]any{"source_ref": "customer-data", "mount_path": "/data", "prefix": "runs/", "access": "read_write"}
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == http.MethodPost {
			if r.URL.Path != "/api/v1/sandboxes" {
				t.Fatalf("unexpected route %s", r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			config := body["config"].(map[string]any)
			if !reflect.DeepEqual(config["external_mounts"], []any{wire}) {
				t.Fatalf("unexpected binding: %#v", config["external_mounts"])
			}
			if !reflect.DeepEqual(config["resources"], map[string]any{"memory": "1Gi"}) {
				t.Fatal("memory override lost")
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{"success": true, "data": map[string]any{"sandbox_id": "sb_1", "template": "default", "runtime_id": "alloc", "status": "running"}})
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/sandboxes/sb_1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		data := sandboxJSON("sb_1")
		data["external_mounts"] = []any{wire}
		writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	mounts := []apispec.ExternalMountBinding{binding}
	option := WithSandboxExternalMounts(mounts)
	mounts[0].SourceRef = "changed-after-option"
	sandbox, err := client.ClaimSandbox(context.Background(), "default", option, WithSandboxMemory("1Gi"))
	if err != nil {
		t.Fatal(err)
	}
	detail, err := client.GetSandbox(context.Background(), sandbox.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Paused || !reflect.DeepEqual(detail.ExternalMounts, []apispec.ExternalMountBinding{binding}) {
		t.Fatal("paused bindings lost")
	}
	if calls != 2 {
		t.Fatalf("unexpected request count %d", calls)
	}
}

func TestS3CredentialSourceWire(t *testing.T) {
	spec := apispec.StaticS3SourceSpec{Bucket: "customer-bucket", Region: "us-east-1", AccessKey: "test-access", SecretKey: "test-secret", SessionToken: apispec.NewOptString("test-session"), Prefix: apispec.NewOptString("agents/"), EndpointUrl: apispec.NewOptString("https://s3.example.test"), AllowWrite: apispec.NewOptBool(true)}
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/credential-sources" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"name": "customer-data", "resolverKind": "static_s3", "spec": map[string]any{"staticS3": map[string]any{"bucket": "customer-bucket", "region": "us-east-1", "accessKey": "test-access", "secretKey": "test-secret", "sessionToken": "test-session", "prefix": "agents/", "endpointUrl": "https://s3.example.test", "allowWrite": true}}}
		if !reflect.DeepEqual(body, want) {
			t.Fatal("S3 source fields did not survive serialization")
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{"success": true, "data": map[string]any{"name": "customer-data", "resolverKind": "static_s3", "currentVersion": 1, "status": "active"}})
	})
	defer server.Close()
	result, err := client.CreateCredentialSource(context.Background(), apispec.CredentialSourceWriteRequest{Name: "customer-data", ResolverKind: apispec.CredentialSourceResolverKindStaticS3, Spec: apispec.CredentialSourceWriteSpec{StaticS3: apispec.NewOptStaticS3SourceSpec(spec)}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResolverKind != apispec.CredentialSourceResolverKindStaticS3 || calls != 1 {
		t.Fatal("unexpected source result or retry")
	}
}

func TestExternalMountClaimDenialDoesNotRetry(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(t, w, http.StatusConflict, map[string]any{"success": false, "error": map[string]any{"code": "conflict", "message": "mount unavailable"}})
	})
	defer server.Close()
	_, err := client.ClaimSandbox(context.Background(), "default", WithSandboxExternalMounts([]apispec.ExternalMountBinding{{SourceRef: "customer-data", MountPath: "/data"}}))
	if err == nil || calls != 1 {
		t.Fatalf("denial must return without retry: err=%v calls=%d", err, calls)
	}
}
