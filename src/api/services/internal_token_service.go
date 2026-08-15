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

// MintForJob creates a new internal token bound to both userID and jobID,
// used to authorize the Go internal provider-tool endpoints
// (contracts/agent-internal-contract.md §7) called by the Python deep
// identification pipeline for a single job run. It is a distinct 4-field
// token shape from Mint's 3-field userID-only token so the two token
// families can never be confused by Verify/VerifyForJob.
func (s *InternalTokenService) MintForJob(userID, jobID uint) (string, error) {
	expiry := time.Now().Add(30 * time.Second).Unix()

	payload := strconv.FormatUint(uint64(userID), 10) + ":" +
		strconv.FormatUint(uint64(jobID), 10) + ":" +
		strconv.FormatInt(expiry, 10)

	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(payload))
	signature := h.Sum(nil)

	token := payload + ":" + base64.RawURLEncoding.EncodeToString(signature)
	return token, nil
}

// VerifyForJob validates a job-scoped token minted by MintForJob and
// returns the embedded (userID, jobID) pair. Returns
// ErrInvalidInternalToken for malformed, expired, or mis-signed tokens,
// including a userID-only token from Mint (wrong field count).
func (s *InternalTokenService) VerifyForJob(token string) (userID uint, jobID uint, err error) {
	parts := strings.Split(token, ":")
	if len(parts) != 4 {
		return 0, 0, ErrInvalidInternalToken
	}

	userIDStr, jobIDStr, expiryStr, signatureB64 := parts[0], parts[1], parts[2], parts[3]

	userIDInt, err := strconv.ParseUint(userIDStr, 10, strconv.IntSize)
	if err != nil {
		return 0, 0, ErrInvalidInternalToken
	}
	jobIDInt, err := strconv.ParseUint(jobIDStr, 10, strconv.IntSize)
	if err != nil {
		return 0, 0, ErrInvalidInternalToken
	}

	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidInternalToken
	}
	if time.Now().Unix() > expiry {
		return 0, 0, ErrInvalidInternalToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return 0, 0, ErrInvalidInternalToken
	}

	payload := userIDStr + ":" + jobIDStr + ":" + expiryStr
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(payload))
	expected := h.Sum(nil)

	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return 0, 0, ErrInvalidInternalToken
	}

	return uint(userIDInt), uint(jobIDInt), nil
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
