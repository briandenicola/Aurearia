package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HashAPIKey returns a deterministic keyed digest for API key storage/lookup.
func HashAPIKey(plainKey, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(plainKey))
	return hex.EncodeToString(mac.Sum(nil))
}
