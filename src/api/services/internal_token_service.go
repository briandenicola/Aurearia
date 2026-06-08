package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidInternalToken is returned when a token is invalid, expired, or malformed.
	ErrInvalidInternalToken = errors.New("invalid internal token")
)

// InternalTokenService mints and verifies short-lived HMAC-signed tokens
// for internal service-to-service communication (Go API <-> Python Agent).
// Tokens carry a userID and expire after 30 seconds.
type InternalTokenService struct {
	secret []byte
}

// NewInternalTokenService creates a token service with the given HMAC secret.
func NewInternalTokenService(secret string) *InternalTokenService {
	return &InternalTokenService{
		secret: []byte(secret),
	}
}

// Mint creates a new internal token for the given userID with a 30-second TTL.
// Returns a base64-encoded token string: base64(userID|expiry|hmac).
func (s *InternalTokenService) Mint(userID uint) (string, error) {
	expiry := time.Now().Add(30 * time.Second).Unix()

	// Build payload: "userID:expiry"
	payload := strconv.FormatUint(uint64(userID), 10) + ":" + strconv.FormatInt(expiry, 10)

	// Compute HMAC
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(payload))
	signature := h.Sum(nil)

	// Encode token: payload + ":" + base64(signature)
	token := payload + ":" + base64.RawURLEncoding.EncodeToString(signature)

	return token, nil
}

// Verify validates the token and returns the embedded userID.
// Returns ErrInvalidInternalToken if the token is malformed, expired, or has invalid signature.
func (s *InternalTokenService) Verify(token string) (uint, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return 0, ErrInvalidInternalToken
	}

	userIDStr := parts[0]
	expiryStr := parts[1]
	signatureB64 := parts[2]

	// Parse userID
	userIDInt, err := strconv.ParseUint(userIDStr, 10, strconv.IntSize)
	if err != nil {
		return 0, ErrInvalidInternalToken
	}
	userID := uint(userIDInt)

	// Parse expiry
	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return 0, ErrInvalidInternalToken
	}

	// Check expiration
	if time.Now().Unix() > expiry {
		return 0, ErrInvalidInternalToken
	}

	// Decode signature
	signature, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return 0, ErrInvalidInternalToken
	}

	// Recompute HMAC
	payload := userIDStr + ":" + expiryStr
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(payload))
	expected := h.Sum(nil)

	// Constant-time compare
	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return 0, ErrInvalidInternalToken
	}

	return userID, nil
}
