package models

import "testing"

func TestCoinCopilotDeepHandoffPriorAndResultJobsRemainDistinct(t *testing.T) {
	prior := uint(313)
	valid := CoinCopilotDeepHandoff{
		UserID: 7, RunID: "ccr_1", ExecutionID: "cce_1",
		HandoffKeyHash: "key", RequestFingerprint: "request",
		ExpectedCheckpointVersion: 3, AppContextDigest: "context",
		Operation:  CoinCopilotDeepHandoffOperationRerun,
		TargetKind: CoinCopilotDeepHandoffTargetCoin, TargetID: 42,
		PriorJobID: &prior, TargetSnapshotFingerprint: "snapshot",
		DeepJobID: 314, AdmissionOutcome: CoinCopilotDeepHandoffOutcomeCreated,
	}
	if !IsValidCoinCopilotDeepHandoff(&valid) {
		t.Fatal("valid rerun binding rejected")
	}
	same := valid
	same.DeepJobID = prior
	if IsValidCoinCopilotDeepHandoff(&same) {
		t.Fatal("prior job was overloaded as the resulting Deep job")
	}
	request := valid
	request.Operation = CoinCopilotDeepHandoffOperationRequest
	request.PriorJobID = nil
	if !IsValidCoinCopilotDeepHandoff(&request) {
		t.Fatal("valid request binding rejected")
	}
	request.PriorJobID = &prior
	if IsValidCoinCopilotDeepHandoff(&request) {
		t.Fatal("request admitted a prior rerun job")
	}
}
