package services

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCopilotExecutionTokenBindingExpiryAllowlistAndRevocation(t *testing.T) {
	service := NewInternalTokenService("01234567890123456789012345678901")
	token, err := service.MintForCopilotExecution(7, "ccr_1", "cce_1", []string{"get_coin"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.VerifyForCopilotExecution(token, "get_coin")
	if err != nil || claims.UserID != 7 || claims.RunID != "ccr_1" || claims.ExecutionID != "cce_1" {
		t.Fatalf("claims=%#v err=%v", claims, err)
	}
	if _, err := service.VerifyForCopilotExecution(token, "propose_update"); !errors.Is(err, ErrInvalidInternalToken) {
		t.Fatalf("write tool accepted: %v", err)
	}
	tampered := token[:len(token)-1] + map[bool]string{true: "A", false: "B"}[strings.HasSuffix(token, "B")]
	if _, err := service.VerifyForCopilotExecution(tampered, "get_coin"); !errors.Is(err, ErrInvalidInternalToken) {
		t.Fatalf("tampered token accepted: %v", err)
	}
	service.RevokeCopilotExecution("cce_1")
	if _, err := service.VerifyForCopilotExecution(token, "get_coin"); !errors.Is(err, ErrInvalidInternalToken) {
		t.Fatalf("revoked token accepted: %v", err)
	}
	if _, err := service.MintForCopilotExecution(7, "ccr_1", "cce_2", []string{"shell"}, time.Minute); !errors.Is(err, ErrInvalidInternalToken) {
		t.Fatalf("forbidden allowlist accepted: %v", err)
	}
	now := time.Date(2026, 9, 17, 20, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	maximum, err := service.MintForCopilotExecution(7, "ccr_1", "cce_max", []string{"get_coin"}, coinCopilotExecutionTokenMaxTTL)
	if err != nil {
		t.Fatalf("maximum configured execution TTL rejected: %v", err)
	}
	claims, err = service.VerifyForCopilotExecution(maximum, "get_coin")
	if err != nil || claims.ExpiresAt != now.Add(coinCopilotExecutionTokenMaxTTL).Unix() {
		t.Fatalf("maximum TTL claims=%#v err=%v", claims, err)
	}
	if _, err := service.MintForCopilotExecution(7, "ccr_1", "cce_too_long", []string{"get_coin"}, coinCopilotExecutionTokenMaxTTL+time.Second); !errors.Is(err, ErrInvalidInternalToken) {
		t.Fatalf("over-maximum TTL accepted: %v", err)
	}
	expired, err := service.MintForCopilotExecution(7, "ccr_1", "cce_expired", []string{"get_coin"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if _, err := service.VerifyForCopilotExecution(expired, "get_coin"); !errors.Is(err, ErrInvalidInternalToken) {
		t.Fatalf("expired token accepted: %v", err)
	}
}
