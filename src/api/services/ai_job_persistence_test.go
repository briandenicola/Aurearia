package services

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

type countingAnalysisAgent struct {
	AIJobAgent
	calls atomic.Int32
}

func (a *countingAnalysisAgent) AnalyzeCoin(context.Context, AnalyzeProxyRequest) (string, error) {
	a.calls.Add(1)
	return "new analysis", nil
}

func TestAIJobAnalysisPersistenceFailureRollsBackWithoutInferenceRetry(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	coin := createAIJobTestCoin(t, db, 1)
	addAIJobTestImage(t, db, coin.ID, models.ImageTypeObverse)
	agent := &countingAnalysisAgent{}
	svc.agentProxy = agent
	job, _, err := svc.EnqueueAnalysis(1, coin.ID, "obverse")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER fail_job_completion BEFORE UPDATE OF status ON ai_jobs WHEN NEW.status = 'completed' BEGIN SELECT RAISE(ABORT, 'injected 500 persistence failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	svc.processJob(job.ID)
	if agent.calls.Load() != 1 {
		t.Fatal("persistence failure repeated external inference")
	}
	var stored models.Coin
	if err := db.First(&stored, coin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ObverseAnalysis != "" {
		t.Fatal("analysis persisted without job completion")
	}
	result, err := svc.GetJob(1, job.ID)
	if err != nil || result.Status != models.AIJobStatusFailed {
		t.Fatalf("expected visible failure: %v %v", result, err)
	}
}
