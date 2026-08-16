package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// seedDeepJob inserts a bare job row directly (bypassing CreateJob's
// upload/validation path) so SSE tests can control status/seq/pruning
// precisely without depending on the (intentionally unstarted-in-tests)
// worker pool.
func seedDeepJobForSSE(t *testing.T, deps deepHandlerTestDeps, userID uint, status models.DeepJobStatus) *models.DeepIdentificationJob {
	t.Helper()
	job := &models.DeepIdentificationJob{
		UserID:           userID,
		Source:           models.DeepJobSourceIntake,
		Status:           status,
		InputFingerprint: fmt.Sprintf("sse-fp-%d-%d", userID, time.Now().UnixNano()),
		ExpiresAt:        time.Now().Add(90 * 24 * time.Hour),
	}
	if err := deps.db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return job
}

func deepEventsRouteFor(jobID uint, query string) string {
	path := fmt.Sprintf("/api/deep-identification/jobs/%d/events", jobID)
	if query != "" {
		path += "?" + query
	}
	return path
}

// TestDeepIdentificationHandler_StreamEvents_FirstConnectReplaysAll covers
// T102: a fresh connection with no `since` replays every retained event in
// order and then keeps the connection open (no `event: end` yet) because
// the job is not terminal.
func TestDeepIdentificationHandler_StreamEvents_FirstConnectReplaysAll(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusRunning)
	repo := repository.NewDeepIdentificationRepository(deps.db)
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventJobAccepted, `{}`); err != nil {
		t.Fatalf("append event 1: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventRouterSelected, `{"providers":["nomisma"]}`); err != nil {
		t.Fatalf("append event 2: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, ""), nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()

	// Give the handler time to write the initial replay, then disconnect.
	time.Sleep(150 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after context cancellation")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "id: 1\nevent: job_accepted") {
		t.Fatalf("expected seq 1 job_accepted frame, got body:\n%s", body)
	}
	if !strings.Contains(body, "id: 2\nevent: router_selected") {
		t.Fatalf("expected seq 2 router_selected frame, got body:\n%s", body)
	}
	if strings.Contains(body, "event: end") {
		t.Fatalf("non-terminal job must not emit event: end, got body:\n%s", body)
	}
}

// TestDeepIdentificationHandler_StreamEvents_ReconnectSinceReplaysOnlyGap
// covers the reconnect-with-since row of contract §3: since=1 must replay
// only seq 2 onward, never re-sending seq 1.
func TestDeepIdentificationHandler_StreamEvents_ReconnectSinceReplaysOnlyGap(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusRunning)
	repo := repository.NewDeepIdentificationRepository(deps.db)
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventJobAccepted, `{}`); err != nil {
		t.Fatalf("append event 1: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventProgress, `{"pct":50}`); err != nil {
		t.Fatalf("append event 2: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, "since=1"), nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()
	time.Sleep(150 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after context cancellation")
	}

	body := rec.Body.String()
	if strings.Contains(body, "event: job_accepted") {
		t.Fatalf("since=1 must not replay seq 1, got body:\n%s", body)
	}
	if !strings.Contains(body, "id: 2\nevent: progress") {
		t.Fatalf("expected seq 2 progress frame, got body:\n%s", body)
	}
}

// TestDeepIdentificationHandler_StreamEvents_LiveTailDeliversNewEvents
// proves a connected client sees an event appended after connect (T095's
// broker wake) without reconnecting, and that the connection closes with
// `event: end` exactly when the terminal event arrives.
func TestDeepIdentificationHandler_StreamEvents_LiveTailDeliversNewEvents(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusRunning)
	repo := repository.NewDeepIdentificationRepository(deps.db)
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventJobAccepted, `{}`); err != nil {
		t.Fatalf("append event 1: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, ""), nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for the handler to reach its live-tail loop, then publish a new
	// event exactly the way runJob/onFrame do (append then wake).
	time.Sleep(150 * time.Millisecond)
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventProviderResult, `{"provider":"nomisma"}`); err != nil {
		t.Fatalf("append live event: %v", err)
	}
	deps.svc.Broker().Publish(job.ID)

	// Now settle the job terminal and publish again, exactly as runJob does.
	time.Sleep(150 * time.Millisecond)
	if err := deps.db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).
		Updates(map[string]interface{}{"status": models.DeepJobStatusCompleted, "completed_at": time.Now()}).Error; err != nil {
		t.Fatalf("settle job completed: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventTerminal, `{"status":"completed"}`); err != nil {
		t.Fatalf("append terminal event: %v", err)
	}
	deps.svc.Broker().Publish(job.ID)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not close after terminal event")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "id: 1\nevent: job_accepted") {
		t.Fatalf("expected initial replay frame, got body:\n%s", body)
	}
	if !strings.Contains(body, "id: 2\nevent: provider_result") {
		t.Fatalf("expected live-tailed provider_result frame, got body:\n%s", body)
	}
	if !strings.Contains(body, "id: 3\nevent: terminal") {
		t.Fatalf("expected terminal frame, got body:\n%s", body)
	}
	if !strings.Contains(body, "event: end") {
		t.Fatalf("expected event: end after terminal frame, got body:\n%s", body)
	}
}

// TestDeepIdentificationHandler_StreamEvents_TerminalJobRepliesImmediately
// covers the already-terminal-job row: connecting after completion must
// replay the terminal event and close with `event: end` without blocking.
func TestDeepIdentificationHandler_StreamEvents_TerminalJobRepliesImmediately(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusCompleted)
	repo := repository.NewDeepIdentificationRepository(deps.db)
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventJobAccepted, `{}`); err != nil {
		t.Fatalf("append event 1: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventTerminal, `{"status":"completed"}`); err != nil {
		t.Fatalf("append terminal event: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, ""), nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler blocked on an already-terminal job instead of closing immediately")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: terminal") {
		t.Fatalf("expected terminal frame, got body:\n%s", body)
	}
	if !strings.Contains(body, "event: end") {
		t.Fatalf("expected event: end, got body:\n%s", body)
	}
}

// TestDeepIdentificationHandler_StreamEvents_TerminalJobPrunedSynthesizesFrame
// covers PruneEventsBefore having deleted the terminal event row itself:
// the handler must synthesize a terminal frame from the job row rather
// than blocking forever.
func TestDeepIdentificationHandler_StreamEvents_TerminalJobPrunedSynthesizesFrame(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusCompleted)
	prunedAt := time.Now()
	if err := deps.db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).
		Updates(map[string]interface{}{"events_pruned_at": prunedAt, "last_seq": int64(7)}).Error; err != nil {
		t.Fatalf("stamp pruned: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, ""), nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler blocked instead of synthesizing a terminal frame for a pruned terminal job")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "id: 7\nevent: terminal") {
		t.Fatalf("expected synthesized seq 7 terminal frame, got body:\n%s", body)
	}
	if !strings.Contains(body, "event: end") {
		t.Fatalf("expected event: end, got body:\n%s", body)
	}
}

// TestDeepIdentificationHandler_StreamEvents_StreamTruncatedOnPrunedGap
// covers the "reconnect after long outage, events pruned" row: since points
// into a gap that pruning already erased, so the handler must emit
// stream_truncated before replaying the surviving tail.
func TestDeepIdentificationHandler_StreamEvents_StreamTruncatedOnPrunedGap(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusRunning)
	repo := repository.NewDeepIdentificationRepository(deps.db)
	// Simulate seq 1-3 having existed and been pruned; only seq 4 (progress)
	// survives.
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventProgress, `{"pct":10}`); err != nil {
		t.Fatalf("append seq1: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventProgress, `{"pct":20}`); err != nil {
		t.Fatalf("append seq2: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventProgress, `{"pct":30}`); err != nil {
		t.Fatalf("append seq3: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventProgress, `{"pct":40}`); err != nil {
		t.Fatalf("append seq4: %v", err)
	}
	if err := deps.db.Where("job_id = ? AND seq <= ?", job.ID, int64(3)).Delete(&models.DeepIdentificationEvent{}).Error; err != nil {
		t.Fatalf("simulate prune delete: %v", err)
	}
	if err := deps.db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).
		Update("events_pruned_at", time.Now()).Error; err != nil {
		t.Fatalf("stamp pruned: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, "since=1"), nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()
	time.Sleep(150 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after context cancellation")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: stream_truncated") {
		t.Fatalf("expected stream_truncated frame, got body:\n%s", body)
	}
	if !strings.Contains(body, `"earliestSeq":4`) {
		t.Fatalf("expected earliestSeq 4 in stream_truncated payload, got body:\n%s", body)
	}
	if !strings.Contains(body, "id: 4\nevent: progress") {
		t.Fatalf("expected surviving seq 4 replayed after truncation notice, got body:\n%s", body)
	}
	summary, err := deps.svc.GetObservabilitySummary()
	if err != nil {
		t.Fatalf("GetObservabilitySummary: %v", err)
	}
	if summary.ActiveSSEStreams != 0 || summary.ReconnectCount != 1 || summary.TruncationCount != 1 {
		t.Fatalf("unexpected SSE observability: active=%d reconnects=%d truncations=%d",
			summary.ActiveSSEStreams, summary.ReconnectCount, summary.TruncationCount)
	}
}

// TestDeepIdentificationHandler_StreamEvents_ExpiredJobReturns410 covers
// the retention-expired-job row of contract §3.
func TestDeepIdentificationHandler_StreamEvents_ExpiredJobReturns410(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusCompleted)
	if err := deps.db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).
		Update("expires_at", time.Now().Add(-1*time.Hour)).Error; err != nil {
		t.Fatalf("expire job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, ""), nil)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 for expired job, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDeepIdentificationHandler_StreamEvents_CrossUserReturns404 covers the
// same non-owner-invisibility guarantee as GetJob/Cancel/Retry, extended to
// the SSE endpoint.
func TestDeepIdentificationHandler_StreamEvents_CrossUserReturns404(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	if err := deps.db.Create(&models.User{ID: 2, Username: "other", Email: "other@example.com", PasswordHash: "x"}).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}
	otherJob := seedDeepJobForSSE(t, deps, 2, models.DeepJobStatusRunning)
	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(otherJob.ID, ""), nil)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-user stream, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDeepIdentificationHandler_StreamEvents_FourthConcurrentStreamReturns429
// covers the "max 3 concurrent streams per job" cap (contract §4): a 4th
// simultaneous connection attempt must 429 rather than displacing or
// queuing behind the existing three.
func TestDeepIdentificationHandler_StreamEvents_FourthConcurrentStreamReturns429(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusRunning)

	broker := deps.svc.Broker()
	var unsubs []func()
	for i := 0; i < 3; i++ {
		_, unsubscribe := broker.Subscribe(job.ID)
		unsubs = append(unsubs, unsubscribe)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, ""), nil)
	rec := httptest.NewRecorder()
	deps.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 4th concurrent stream, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDeepIdentificationHandler_StreamEvents_RestartRecoverySettlesAndBroadcasts
// covers T105: a job left "running" by a prior process instance (stale
// heartbeat, no live worker) must, once the janitor's startup sweep runs,
// settle to failed/stale_restart with a single terminal event that an SSE
// client observes exactly once - proving the broker Publish call inside
// recoverStaleAndSweepHints (T095) actually reaches a connected reader.
func TestDeepIdentificationHandler_StreamEvents_RestartRecoverySettlesAndBroadcasts(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusRunning)
	if err := deps.db.Model(&models.DeepIdentificationJob{}).Where("id = ?", job.ID).
		Updates(map[string]interface{}{
			"active_key":   "active",
			"heartbeat_at": time.Now().Add(-1 * time.Hour),
		}).Error; err != nil {
		t.Fatalf("stamp stale heartbeat: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, ""), nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()

	// Give the handler time to subscribe and enter its live-tail loop
	// before the restart-recovery sweep runs, exactly like a client that
	// was already streaming when the process restarted underneath it.
	time.Sleep(150 * time.Millisecond)
	deps.svc.RecoverStaleAndSweepHintsForTest()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not close after restart-recovery settled the job terminal")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: terminal") {
		t.Fatalf("expected a terminal frame after restart recovery, got body:\n%s", body)
	}
	if !strings.Contains(body, "event: end") {
		t.Fatalf("expected event: end after restart recovery settled the job, got body:\n%s", body)
	}
	if strings.Count(body, "event: terminal") != 1 {
		t.Fatalf("expected exactly one terminal event, got body:\n%s", body)
	}

	var final models.DeepIdentificationJob
	if err := deps.db.First(&final, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if final.Status != models.DeepJobStatusFailed || final.FailureCode != "stale_restart" {
		t.Fatalf("expected failed/stale_restart, got status=%s code=%s", final.Status, final.FailureCode)
	}
}

// TestDeepIdentificationHandler_StreamEvents_SinceQueryWinsOverLastEventID
// covers contract §3's "since wins if both present" resolution rule.
func TestDeepIdentificationHandler_StreamEvents_SinceQueryWinsOverLastEventID(t *testing.T) {
	deps := setupDeepIdentificationHandlerTest(t, 1, true)
	job := seedDeepJobForSSE(t, deps, 1, models.DeepJobStatusRunning)
	repo := repository.NewDeepIdentificationRepository(deps.db)
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventJobAccepted, `{}`); err != nil {
		t.Fatalf("append seq1: %v", err)
	}
	if _, err := repo.AppendEvent(job.ID, 1, models.DeepEventProgress, `{"pct":10}`); err != nil {
		t.Fatalf("append seq2: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, deepEventsRouteFor(job.ID, "since=0"), nil).WithContext(ctx)
	req.Header.Set("Last-Event-ID", "1")
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		deps.router.ServeHTTP(rec, req)
		close(done)
	}()
	time.Sleep(150 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after context cancellation")
	}

	body := rec.Body.String()
	// since=0 (query) must win over Last-Event-ID:1, so seq 1 is replayed
	// even though the header alone would have suppressed it.
	if !strings.Contains(body, "id: 1\nevent: job_accepted") {
		t.Fatalf("expected since=0 (query) to win over Last-Event-ID:1 and replay seq 1, got body:\n%s", body)
	}
}
