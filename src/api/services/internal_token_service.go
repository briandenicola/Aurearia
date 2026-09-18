package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

var (
	// ErrInvalidInternalToken is returned when a token is invalid, expired, or malformed.
	ErrInvalidInternalToken = errors.New("invalid internal token")
)

// jobTokenHKDFInfo is the HKDF "info" label used to derive the job-token
// signing secret from cfg.JWTSecret (T080). It is a fixed, distinct
// constant - never the user JWT secret itself - so a leaked job token
// cannot be used as an oracle against the user JWT secret: the two are
// cryptographically unrelated beyond both descending from the same input
// keying material, and HKDF's info-label separation is exactly what
// prevents recovering one from the other. Deriving rather than requiring a
// new configured value means existing deployments need no action.
const jobTokenHKDFInfo = "ancient-coins-api:deep-identification-job-token:v1"
const copilotTokenHKDFInfo = "ancient-coins-api:coin-copilot-execution-token:v1"
const (
	coinCopilotExecutionTokenBuffer = 30 * time.Second
	coinCopilotMaxExecutionTimeout  = 150 * time.Second
	coinCopilotExecutionTokenMaxTTL = 180 * time.Second
)

// jobTokenRevocationRetention bounds how long a settled job's revocation
// record is retained (T081). It only needs to outlive the longest possible
// job token TTL (bounds.total_timeout_s maxes at 900s per
// contracts/agent-internal-contract.md §2, plus MintForJobWithTTL's +30s
// buffer), after which any token for that job would already be rejected by
// its own embedded expiry regardless of revocation state. This keeps the
// revoked-jobs map self-pruning and bounded, rather than growing forever
// like the bug fixed in T078.
const jobTokenRevocationRetention = 20 * time.Minute

func deriveJobTokenSecret(userJWTSecret []byte) []byte {
	return deriveInternalTokenSecret(userJWTSecret, jobTokenHKDFInfo)
}

func deriveInternalTokenSecret(secret []byte, info string) []byte {
	reader := hkdf.New(sha256.New, secret, nil, []byte(info))
	derived := make([]byte, sha256.Size)
	if _, err := reader.Read(derived); err != nil {
		// hkdf.Read only fails if more bytes are requested than the RFC
		// 5869 maximum (255 * hash length) - unreachable for a single
		// sha256.Size read - so this is a defensive panic, not a runtime
		// possibility.
		panic("internal_token_service: hkdf derivation failed: " + err.Error())
	}
	return derived
}

// InternalTokenService mints and verifies short-lived HMAC-signed tokens
// for internal service-to-service communication (Go API <-> Python Agent).
// User-scoped tokens (Mint/Verify) carry a userID and expire after 30
// seconds. Job-scoped tokens (MintForJob/VerifyForJob) are signed with a
// distinct, HKDF-derived secret (jobSecret) so the two token families
// remain cryptographically independent (T080): a compromised job token
// can never be leveraged as an oracle against the user JWT secret.
type InternalTokenService struct {
	secret        []byte
	jobSecret     []byte
	copilotSecret []byte

	settledMu         sync.Mutex
	settled           map[uint]time.Time
	revokedExecutions map[string]time.Time
	now               func() time.Time
}

// NewInternalTokenService creates a token service with the given HMAC secret.
func NewInternalTokenService(secret string) *InternalTokenService {
	secretBytes := []byte(secret)
	return &InternalTokenService{
		secret:            secretBytes,
		jobSecret:         deriveJobTokenSecret(secretBytes),
		copilotSecret:     deriveInternalTokenSecret(secretBytes, copilotTokenHKDFInfo),
		settled:           make(map[uint]time.Time),
		revokedExecutions: make(map[string]time.Time),
		now:               time.Now,
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
	return s.MintForJobWithTTL(userID, jobID, 30*time.Second)
}

// MintForJobWithTTL is MintForJob with an explicit TTL. The deep
// identification pipeline (Phase 7) can run up to `bounds.total_timeout_s`
// (up to 900s, contracts/agent-internal-contract.md §2), far longer than
// MintForJob's fixed 30-second TTL, yet Python must be able to call the Go
// provider-tool endpoints with this same job-scoped token for the entire
// run. VerifyForJob only checks the embedded expiry timestamp (never a
// fixed original TTL), so this reuses the identical signing/verification
// logic with no change to VerifyForJob or MintForJob's own 30s callers.
func (s *InternalTokenService) MintForJobWithTTL(userID, jobID uint, ttl time.Duration) (string, error) {
	expiry := time.Now().Add(ttl).Unix()

	payload := strconv.FormatUint(uint64(userID), 10) + ":" +
		strconv.FormatUint(uint64(jobID), 10) + ":" +
		strconv.FormatInt(expiry, 10)

	h := hmac.New(sha256.New, s.jobSecret)
	h.Write([]byte(payload))
	signature := h.Sum(nil)

	token := payload + ":" + base64.RawURLEncoding.EncodeToString(signature)
	return token, nil
}

// RevokeJob marks jobID as settled (T081) so any not-yet-expired job token
// for it is rejected by VerifyForJob from this moment on, even though its
// own embedded expiry has not yet passed. Call this from the job service's
// terminal path (alongside the T078 budget-tracker Reset) so a long-TTL
// token cannot be replayed after the job it was scoped to has completed,
// failed, or been cancelled.
//
// The settled map is bounded: each call opportunistically prunes entries
// older than jobTokenRevocationRetention, which is chosen to always exceed
// the longest possible job-token TTL, so the map cannot grow without bound
// across the life of a long-running process (mirroring the fix for the
// unbounded-map bug in T078).
func (s *InternalTokenService) RevokeJob(jobID uint) {
	now := time.Now()
	s.settledMu.Lock()
	defer s.settledMu.Unlock()
	s.settled[jobID] = now
	for id, settledAt := range s.settled {
		if now.Sub(settledAt) > jobTokenRevocationRetention {
			delete(s.settled, id)
		}
	}
}

func (s *InternalTokenService) isJobRevoked(jobID uint) bool {
	s.settledMu.Lock()
	defer s.settledMu.Unlock()
	_, revoked := s.settled[jobID]
	return revoked
}

// VerifyForJob validates a job-scoped token minted by MintForJob and
// returns the embedded (userID, jobID) pair. Returns
// ErrInvalidInternalToken for malformed, expired, or mis-signed tokens,
// including a userID-only token from Mint (wrong field count), and for any
// token whose jobID has been revoked via RevokeJob (T081) - even one that
// has not yet reached its own embedded expiry.
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
	if base64.RawURLEncoding.EncodeToString(signature) != signatureB64 {
		return 0, 0, ErrInvalidInternalToken
	}

	payload := userIDStr + ":" + jobIDStr + ":" + expiryStr
	h := hmac.New(sha256.New, s.jobSecret)
	h.Write([]byte(payload))
	expected := h.Sum(nil)

	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return 0, 0, ErrInvalidInternalToken
	}

	jobID = uint(jobIDInt)
	if s.isJobRevoked(jobID) {
		return 0, 0, ErrInvalidInternalToken
	}

	return uint(userIDInt), jobID, nil
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
	if base64.RawURLEncoding.EncodeToString(signature) != signatureB64 {
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

type CopilotExecutionClaims struct {
	Version      int      `json:"version"`
	UserID       uint     `json:"user_id"`
	RunID        string   `json:"run_id"`
	ExecutionID  string   `json:"execution_id"`
	AllowedTools []string `json:"allowed_tools"`
	ExpiresAt    int64    `json:"expires_at"`
	Nonce        string   `json:"nonce"`
}

func (s *InternalTokenService) MintForCopilotExecution(userID uint, runID, executionID string, allowedTools []string, ttl time.Duration) (string, error) {
	if userID == 0 || runID == "" || executionID == "" || ttl <= 0 || ttl > coinCopilotExecutionTokenMaxTTL {
		return "", ErrInvalidInternalToken
	}
	tools := append([]string(nil), allowedTools...)
	sort.Strings(tools)
	for i, tool := range tools {
		if !IsCoinCopilotToolAllowed(tool) || (i > 0 && tools[i-1] == tool) {
			return "", ErrInvalidInternalToken
		}
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	claims := CopilotExecutionClaims{
		Version: 1, UserID: userID, RunID: runID, ExecutionID: executionID,
		AllowedTools: tools, ExpiresAt: s.now().UTC().Add(ttl).Unix(),
		Nonce: base64.RawURLEncoding.EncodeToString(nonceBytes),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.copilotSecret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *InternalTokenService) VerifyForCopilotExecution(token, requestedTool string) (*CopilotExecutionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidInternalToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || base64.RawURLEncoding.EncodeToString(signature) != parts[1] {
		return nil, ErrInvalidInternalToken
	}
	mac := hmac.New(sha256.New, s.copilotSecret)
	_, _ = mac.Write([]byte(parts[0]))
	if subtle.ConstantTimeCompare(signature, mac.Sum(nil)) != 1 {
		return nil, ErrInvalidInternalToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || base64.RawURLEncoding.EncodeToString(payload) != parts[0] {
		return nil, ErrInvalidInternalToken
	}
	var claims CopilotExecutionClaims
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&claims); err != nil || claims.Version != 1 || claims.UserID == 0 ||
		claims.RunID == "" || claims.ExecutionID == "" || claims.Nonce == "" || s.now().UTC().Unix() >= claims.ExpiresAt {
		return nil, ErrInvalidInternalToken
	}
	allowed := false
	for _, tool := range claims.AllowedTools {
		if !IsCoinCopilotToolAllowed(tool) {
			return nil, ErrInvalidInternalToken
		}
		if tool == requestedTool {
			allowed = true
		}
	}
	if !allowed {
		return nil, ErrInvalidInternalToken
	}
	s.settledMu.Lock()
	_, revoked := s.revokedExecutions[claims.ExecutionID]
	s.settledMu.Unlock()
	if revoked {
		return nil, ErrInvalidInternalToken
	}
	return &claims, nil
}

func (s *InternalTokenService) RevokeCopilotExecution(executionID string) {
	now := s.now().UTC()
	s.settledMu.Lock()
	defer s.settledMu.Unlock()
	s.revokedExecutions[executionID] = now
	for id, revokedAt := range s.revokedExecutions {
		if now.Sub(revokedAt) > coinCopilotExecutionTokenMaxTTL {
			delete(s.revokedExecutions, id)
		}
	}
}
