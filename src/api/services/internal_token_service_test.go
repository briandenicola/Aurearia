package services

import (
	"testing"
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
