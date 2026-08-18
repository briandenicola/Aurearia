package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"
)

func TestInternalTokenService_MintForJobRoundTrip(t *testing.T) {
	svc := NewInternalTokenService("test-secret")

	token, err := svc.MintForJob(42, 100)
	if err != nil {
		t.Fatalf("MintForJob: %v", err)
	}

	userID, jobID, err := svc.VerifyForJob(token)
	if err != nil {
		t.Fatalf("VerifyForJob: %v", err)
	}
	if userID != 42 || jobID != 100 {
		t.Fatalf("expected userID=42 jobID=100, got userID=%d jobID=%d", userID, jobID)
	}
}

func TestInternalTokenService_VerifyForJobRejectsTampering(t *testing.T) {
	svc := NewInternalTokenService("test-secret")

	token, err := svc.MintForJob(1, 5)
	if err != nil {
		t.Fatalf("MintForJob: %v", err)
	}

	tampered := token[:len(token)-1] + "x"
	if tampered == token {
		t.Fatal("tampering did not change the token")
	}
	if _, _, err := svc.VerifyForJob(tampered); err != ErrInvalidInternalToken {
		t.Fatalf("expected ErrInvalidInternalToken for tampered token, got %v", err)
	}

	// A token minted for a different job must not verify as jobID=5 - the
	// signature covers jobID so forging a "foreign job" binding by editing
	// the jobID segment directly (leaving the rest of the token as-is)
	// must also fail.
	forged, err := svc.MintForJob(1, 5)
	if err != nil {
		t.Fatalf("MintForJob: %v", err)
	}
	// Replace the "5" job segment with "6" (both single digit, so the
	// overall token shape/length is unaffected) - this must invalidate
	// the signature.
	parts := splitInternalTokenForTest(forged)
	if parts[1] != "5" {
		t.Fatalf("expected job segment '5', got %q", parts[1])
	}
	forgedToken := parts[0] + ":6:" + parts[2] + ":" + parts[3]
	if _, _, err := svc.VerifyForJob(forgedToken); err != ErrInvalidInternalToken {
		t.Fatalf("expected ErrInvalidInternalToken for forged job binding, got %v", err)
	}
}

func splitInternalTokenForTest(token string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == ':' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

func TestInternalTokenService_MintAndMintForJobAreDistinctFamilies(t *testing.T) {
	svc := NewInternalTokenService("test-secret")

	userOnlyToken, err := svc.Mint(3)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if _, _, err := svc.VerifyForJob(userOnlyToken); err != ErrInvalidInternalToken {
		t.Fatalf("expected VerifyForJob to reject a userID-only token, got %v", err)
	}

	jobToken, err := svc.MintForJob(3, 11)
	if err != nil {
		t.Fatalf("MintForJob: %v", err)
	}
	if _, err := svc.Verify(jobToken); err != ErrInvalidInternalToken {
		t.Fatalf("expected Verify to reject a job-scoped token, got %v", err)
	}
}

// TestInternalTokenService_RevokeJob_TokenDoesNotVerifyAfterSettle proves
// T081(a): a token minted for job A no longer verifies once job A settles,
// even though its own embedded expiry has not yet passed.
func TestInternalTokenService_RevokeJob_TokenDoesNotVerifyAfterSettle(t *testing.T) {
	svc := NewInternalTokenService("test-secret")

	token, err := svc.MintForJobWithTTL(1, 42, time.Minute)
	if err != nil {
		t.Fatalf("MintForJobWithTTL: %v", err)
	}
	if _, _, err := svc.VerifyForJob(token); err != nil {
		t.Fatalf("expected token to verify before job settles, got %v", err)
	}

	svc.RevokeJob(42)

	if _, _, err := svc.VerifyForJob(token); err != ErrInvalidInternalToken {
		t.Fatalf("expected ErrInvalidInternalToken for a token whose job has settled, got %v", err)
	}
}

// TestInternalTokenService_RevokeJob_OnlyAffectsTheSettledJob proves a
// concurrently running second job's token is unaffected by a different
// job's settlement.
func TestInternalTokenService_RevokeJob_OnlyAffectsTheSettledJob(t *testing.T) {
	svc := NewInternalTokenService("test-secret")

	tokenA, err := svc.MintForJobWithTTL(1, 1, time.Minute)
	if err != nil {
		t.Fatalf("MintForJobWithTTL: %v", err)
	}
	tokenB, err := svc.MintForJobWithTTL(1, 2, time.Minute)
	if err != nil {
		t.Fatalf("MintForJobWithTTL: %v", err)
	}

	svc.RevokeJob(1)

	if _, _, err := svc.VerifyForJob(tokenA); err != ErrInvalidInternalToken {
		t.Fatalf("expected job A's token to be rejected after RevokeJob(1), got %v", err)
	}
	if _, _, err := svc.VerifyForJob(tokenB); err != nil {
		t.Fatalf("expected job B's token to remain valid, got %v", err)
	}
}

// TestInternalTokenService_JobSecretIsDistinctFromUserSecret proves
// T080/T082(b,c): a job-shaped token signed with the *user* JWT secret
// must not verify as a job token, and a user-shaped token signed with the
// *job* secret must not verify as a user JWT. Both tokens below share the
// exact field count/shape their real counterparts have, so this isolates
// the signing-secret separation itself rather than the field-count
// mismatch already covered by
// TestInternalTokenService_MintAndMintForJobAreDistinctFamilies.
func TestInternalTokenService_JobSecretIsDistinctFromUserSecret(t *testing.T) {
	svc := NewInternalTokenService("test-secret")

	expiry := time.Now().Add(time.Minute).Unix()

	// Job-shaped (4-field) payload signed with the user secret.
	jobShapedPayload := strconv.FormatUint(1, 10) + ":" + strconv.FormatUint(42, 10) + ":" + strconv.FormatInt(expiry, 10)
	h := hmac.New(sha256.New, svc.secret)
	h.Write([]byte(jobShapedPayload))
	forgedJobToken := jobShapedPayload + ":" + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	if _, _, err := svc.VerifyForJob(forgedJobToken); err != ErrInvalidInternalToken {
		t.Fatalf("expected a job token signed with the user secret to be rejected, got %v", err)
	}

	// User-shaped (3-field) payload signed with the job secret.
	userShapedPayload := strconv.FormatUint(1, 10) + ":" + strconv.FormatInt(expiry, 10)
	h2 := hmac.New(sha256.New, svc.jobSecret)
	h2.Write([]byte(userShapedPayload))
	forgedUserToken := userShapedPayload + ":" + base64.RawURLEncoding.EncodeToString(h2.Sum(nil))
	if _, err := svc.Verify(forgedUserToken); err != ErrInvalidInternalToken {
		t.Fatalf("expected a user token signed with the job secret to be rejected, got %v", err)
	}
}
