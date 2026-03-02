package sandbox0

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "sandbox0-webhook-secret"
	payload := []byte(`{"event_id":"evt_1","event_type":"sandbox.ready","sandbox_id":"sb_1"}`)
	validSignature := signLikeProcd(secret, payload)

	tests := []struct {
		name      string
		signature string
		payload   []byte
		want      bool
	}{
		{
			name:      "valid signature",
			signature: validSignature,
			payload:   payload,
			want:      true,
		},
		{
			name:      "valid uppercase signature",
			signature: strings.ToUpper(validSignature),
			payload:   payload,
			want:      true,
		},
		{
			name:      "invalid signature",
			signature: signLikeProcd("wrong-secret", payload),
			payload:   payload,
			want:      false,
		},
		{
			name:      "tampered payload",
			signature: validSignature,
			payload:   []byte(`{"event_id":"evt_1","event_type":"sandbox.killed","sandbox_id":"sb_1"}`),
			want:      false,
		},
		{
			name:      "malformed signature",
			signature: "not-hex",
			payload:   payload,
			want:      false,
		},
		{
			name:      "empty signature",
			signature: "",
			payload:   payload,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyWebhookSignature(secret, tt.payload, tt.signature)
			if got != tt.want {
				t.Fatalf("VerifyWebhookSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func signLikeProcd(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
