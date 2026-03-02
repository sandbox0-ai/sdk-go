package sandbox0

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifyWebhookSignature validates X-Sandbox0-Signature with HMAC-SHA256.
// The signature must be a hex-encoded digest of the raw webhook request body.
func VerifyWebhookSignature(secret string, payload []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}

	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	expected := mac.Sum(nil)

	return hmac.Equal(expected, provided)
}
