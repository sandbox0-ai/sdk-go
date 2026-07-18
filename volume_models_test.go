package sandbox0

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxVolumeMeteredStorageRoundTrip(t *testing.T) {
	t.Parallel()

	const payload = `{
		"id":"vol_123",
		"team_id":"team_123",
		"user_id":"user_123",
		"backend":"s0fs",
		"metered_storage_bytes":8734523392,
		"storage_observed_at":"2026-07-18T08:30:00Z",
		"created_at":"2026-07-18T08:00:00Z",
		"updated_at":"2026-07-18T08:30:00Z"
	}`

	var volume apispec.SandboxVolume
	if err := json.Unmarshal([]byte(payload), &volume); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got, ok := volume.MeteredStorageBytes.Get(); !ok || got != 8_734_523_392 {
		t.Fatalf("MeteredStorageBytes = %d, set=%v, want 8734523392", got, ok)
	}
	observedAt, ok := volume.StorageObservedAt.Get()
	if !ok {
		t.Fatal("StorageObservedAt is not set")
	}
	wantObservedAt := time.Date(2026, 7, 18, 8, 30, 0, 0, time.UTC)
	if !observedAt.Equal(wantObservedAt) {
		t.Fatalf("StorageObservedAt = %s, want %s", observedAt, wantObservedAt)
	}

	encoded, err := volume.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var roundTrip map[string]any
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("Unmarshal(round trip) error = %v", err)
	}
	if got := roundTrip["metered_storage_bytes"]; got != float64(8_734_523_392) {
		t.Fatalf("metered_storage_bytes = %#v, want 8734523392", got)
	}
	if got := roundTrip["storage_observed_at"]; got != "2026-07-18T08:30:00Z" {
		t.Fatalf("storage_observed_at = %#v, want 2026-07-18T08:30:00Z", got)
	}
}

func TestSandboxVolumeMeteredStoragePreservesNull(t *testing.T) {
	t.Parallel()

	const payload = `{
		"id":"vol_s3",
		"team_id":"team_123",
		"user_id":"user_123",
		"backend":"s3",
		"metered_storage_bytes":null,
		"storage_observed_at":null,
		"created_at":"2026-07-18T08:00:00Z",
		"updated_at":"2026-07-18T08:30:00Z"
	}`

	var volume apispec.SandboxVolume
	if err := json.Unmarshal([]byte(payload), &volume); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !volume.MeteredStorageBytes.IsSet() || !volume.MeteredStorageBytes.IsNull() {
		t.Fatalf("MeteredStorageBytes set=%v null=%v, want true/true",
			volume.MeteredStorageBytes.IsSet(), volume.MeteredStorageBytes.IsNull())
	}
	if !volume.StorageObservedAt.IsSet() || !volume.StorageObservedAt.IsNull() {
		t.Fatalf("StorageObservedAt set=%v null=%v, want true/true",
			volume.StorageObservedAt.IsSet(), volume.StorageObservedAt.IsNull())
	}
}
