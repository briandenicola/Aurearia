//go:build seam

package integration_test

// T106 [F4] (spec 351, Phase 15): a real Go<->Python seam test.
//
// Every other deep-identification test in this repo drives
// DeepIdentificationPipelineRunner.Run against an httptest fake that emits
// hand-written, Go-shaped Python SSE frames (see
// src/api/services/deep_identification_pipeline_runner_stream_test.go).
// Those fixtures are internally consistent and have been wrong before -
// commit 080e598 shipped a production outage because the Go and Python
// sides each maintained fixtures by convention, and neither side's tests
// ever exercised the other side's real code. This file has no fixture: it
// boots the REAL `uvicorn app.main:app` Python process from src/agent and
// drives the REAL Go pipeline runner (`DeepIdentificationPipelineRunner.
// Run`, exported from deep_identification_pipeline_runner.go, which this
// file only calls - it is never edited here) over a REAL HTTP/SSE round
// trip. If Python's Pydantic request/response models ever drift from what
// Go's proxy structs and deep_identification_frame_translator.go's
// re-encoding expect, this test fails on the wire, not on an agreeable
// fake.
//
// LLM tradeoff (documented in full in
// .squad/decisions/inbox/maximus-seam-test.md): the configured LLM
// provider is Ollama, pointed at a local TCP port nothing is listening on.
// Every LLM call site inside the deep-identification pipeline (the vision
// hypothesis node, the evaluator's disagreement summary, and the
// synthesis narrative) is specified (FR-006/FR-040) to degrade to a
// deterministic fallback on ANY LLM failure - never to raise. Pointing at
// an unreachable endpoint therefore is not a stub of the seam itself: it
// is a real LLM call, over the real network stack, that fails fast and is
// handled by the pipeline's own documented resilience path. That keeps
// this test hermetic (no API key, no external network egress, no
// nondeterministic model output) while still genuinely exercising the
// vision node's structured-output call path end to end. The provider
// catalog is left at its real default (numista/nomisma/ngc/ocre/rpc, see
// deepPipelineProviderCatalog) but `tools_base_url` is left empty, which
// makes the two network-capable providers (numista/nomisma) settle
// immediately as "unconfigured" with zero upstream calls - by the exact
// same code path production uses when the tools client is absent, not by
// test-only branching - while NGC/RPC (never automated) and OCRE (disabled
// by default) still run their real, always-network-free provider nodes.
// This keeps the test hermetic without touching numista.org/nomisma.org
// while still exercising the real router -> provider-fanout -> evaluator
// -> synthesis pipeline and the real provider_result reduction contract.
//
// Guarded twice so it can never affect unattended CI: (1) a `seam` build
// tag, so `go build ./...`/`go vet ./...`/`go test ./...` never even
// compile this file without `-tags=seam`; and (2) an explicit
// `DEEP_SEAM_TEST=1` environment variable, so simply building with the tag
// is still not enough to run it. See docs/testing.md for exact run
// instructions, prerequisites, and expected runtime.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeepIdentificationSeam_RealPythonServiceRoundTrip(t *testing.T) {
	if os.Getenv("DEEP_SEAM_TEST") != "1" {
		t.Skip("set DEEP_SEAM_TEST=1 (and build/run with -tags=seam) to run the Go<->Python deep-identification seam test; see docs/testing.md")
	}

	pythonExe, err := seamPythonExecutable()
	if err != nil {
		t.Skipf("skipping seam test: %v", err)
	}

	agentDir, err := filepath.Abs(filepath.Join("..", "..", "agent"))
	if err != nil {
		t.Fatalf("resolve agent dir: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(agentDir, "app", "main.py")); statErr != nil {
		t.Skipf("skipping seam test: agent service not found at %s: %v", agentDir, statErr)
	}

	agentPort := seamFreePort(t)
	ollamaPort := seamFreePort(t) // intentionally never bound to; see LLM tradeoff note above
	internalToken := "seam-test-internal-token"
	agentBaseURL := fmt.Sprintf("http://127.0.0.1:%d", agentPort)
	ollamaURL := fmt.Sprintf("http://127.0.0.1:%d", ollamaPort)

	procCtx, procCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer procCancel()

	cmd := exec.CommandContext(procCtx, pythonExe,
		"-m", "uvicorn", "app.main:app",
		"--host", "127.0.0.1", "--port", fmt.Sprintf("%d", agentPort),
	)
	cmd.Dir = agentDir
	cmd.Env = append(os.Environ(),
		"AGENT_INTERNAL_SERVICE_TOKEN="+internalToken,
		"AGENT_TRUSTED_OUTBOUND_ORIGINS="+ollamaURL,
		"AGENT_ALLOW_LOCAL_OUTBOUND=true",
		"AGENT_LOG_LEVEL=ERROR",
	)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Start(); err != nil {
		t.Skipf("skipping seam test: could not start python agent service (%s): %v", pythonExe, err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
		if t.Failed() {
			t.Logf("agent service stdout:\n%s", outBuf.String())
			t.Logf("agent service stderr:\n%s", errBuf.String())
		}
	}()

	if !seamWaitForHealth(agentBaseURL, 30*time.Second) {
		t.Fatalf("python agent service did not become healthy at %s within timeout\nstdout:\n%s\nstderr:\n%s", agentBaseURL, outBuf.String(), errBuf.String())
	}

	// --- Wire up the REAL Go side: the exported DeepIdentificationPipelineRunner
	// (deep_identification_pipeline_runner.go) over a real AgentProxy pointed
	// at the real service just booted above. Neither type is edited by this
	// file - both are called exactly as main.go wires them in production.
	dsn := fmt.Sprintf("file:deep_identification_seam_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AppSetting{},
		&models.DeepIdentificationJob{}, &models.DeepIdentificationEvent{},
		&models.DeepIdentificationProviderRun{}, &models.DeepIdentificationArtifact{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := repository.NewDeepIdentificationRepository(db)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	if err := settingsSvc.SetSetting(services.SettingAIProvider, "ollama"); err != nil {
		t.Fatalf("set provider setting: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingOllamaModel, "llama3.1"); err != nil {
		t.Fatalf("set ollama model setting: %v", err)
	}
	if err := settingsSvc.SetSetting(services.SettingOllamaURL, ollamaURL); err != nil {
		t.Fatalf("set ollama url setting: %v", err)
	}

	proxy := services.NewAgentProxy(agentBaseURL, internalToken, services.NewLogger(200))
	tokenSvc := services.NewInternalTokenService("seam-test-token-secret")
	// toolsBaseURL is deliberately "" - see the LLM/network tradeoff note
	// above for why this keeps the test hermetic without stubbing the seam.
	runner := services.NewDeepIdentificationPipelineRunner(proxy, repo, settingsSvc, tokenSvc, "", services.NewLogger(200), nil)

	user := models.User{Username: "seam-test-user", Email: "seam-test-user@example.com", PasswordHash: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	job := &models.DeepIdentificationJob{
		UserID:           user.ID,
		Source:           models.DeepJobSourceIntake,
		InputFingerprint: fmt.Sprintf("seam-fp-%d", time.Now().UnixNano()),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	seamSeedArtifact(t, db, job.ID, user.ID, models.DeepArtifactRoleObverse)
	seamSeedArtifact(t, db, job.ID, user.ID, models.DeepArtifactRoleReverse)

	runCtx, runCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer runCancel()

	result, runErr := runner.Run(runCtx, job)
	if runErr != nil {
		t.Fatalf("pipeline runner returned an error talking to the real python service: %v", runErr)
	}
	if result == nil {
		t.Fatal("pipeline runner returned a nil result with a nil error")
	}

	// --- Assert the DeepSynthesis shape actually returned over the wire -
	// never a hand-written fixture.
	var report map[string]any
	if err := json.Unmarshal([]byte(result.ReportJSON), &report); err != nil {
		t.Fatalf("terminal synthesis report is not valid JSON: %v\nraw: %s", err, result.ReportJSON)
	}
	for _, key := range []string{
		"narrative", "proposed_fields", "disagreements",
		"unresolved_questions", "coverage", "attributions", "partial_success",
	} {
		if _, ok := report[key]; !ok {
			t.Errorf("terminal synthesis report missing expected field %q; got keys=%v", key, seamMapKeys(report))
		}
	}

	// --- Assert the persisted event log went through the real translation
	// contract (deep_identification_frame_translator.go), fed exclusively
	// by frames the real Python process actually emitted.
	events, err := repo.ListEventsSince(job.ID, user.ID, 0)
	if err != nil {
		t.Fatalf("list persisted events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one persisted event from the real run")
	}

	allowedTypes := map[models.DeepIdentificationEventType]bool{
		models.DeepEventRouterSelected:  true,
		models.DeepEventProviderStarted: true,
		models.DeepEventProviderResult:  true,
		models.DeepEventEvaluation:      true,
		models.DeepEventSynthesisStart:  true,
		models.DeepEventProgress:        true,
	}
	var sawProgress, sawRouterSelected, sawProviderResult bool
	for _, event := range events {
		if !allowedTypes[event.Type] {
			t.Errorf("persisted event type %q is outside deepPipelineEventType's closed whitelist", event.Type)
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
			t.Errorf("event %q payload is not valid JSON: %v (%s)", event.Type, err, event.PayloadJSON)
			continue
		}
		switch event.Type {
		case models.DeepEventProgress:
			sawProgress = true
			if len(payload) != 2 {
				t.Errorf("progress event must re-encode to exactly {phase, message}, got %v", payload)
			}
			if _, ok := payload["phase"]; !ok {
				t.Errorf("progress event missing phase field: %v", payload)
			}
			if _, ok := payload["message"]; !ok {
				t.Errorf("progress event missing message field: %v", payload)
			}
		case models.DeepEventProviderResult:
			sawProviderResult = true
			for _, field := range []string{"provider", "status", "confidence", "claimCount"} {
				if _, ok := payload[field]; !ok {
					t.Errorf("provider_result event missing bounded field %q: %v", field, payload)
				}
			}
			if provider, _ := payload["provider"].(string); provider == "image" {
				t.Errorf(`"image" must never appear as a provider in provider_result (FR-025); got %v`, payload)
			}
		case models.DeepEventRouterSelected:
			sawRouterSelected = true
		}
	}
	if !sawProgress {
		t.Error("expected at least one progress event re-encoded to {phase, message}")
	}
	if !sawRouterSelected {
		t.Error("expected a router_selected event")
	}
	if !sawProviderResult {
		t.Error("expected at least one provider_result event (NGC's provider node is zero-network and always runs)")
	}
}

// seamPythonExecutable locates the interpreter to run the real agent
// service with. DEEP_SEAM_PYTHON overrides everything; otherwise the
// repo's src/agent/.venv is used (Windows and POSIX layouts both checked).
func seamPythonExecutable() (string, error) {
	if p := os.Getenv("DEEP_SEAM_PYTHON"); p != "" {
		return p, nil
	}
	candidates := []string{
		filepath.Join("..", "..", "agent", ".venv", "Scripts", "python.exe"),
		filepath.Join("..", "..", "agent", ".venv", "bin", "python"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			if abs, absErr := filepath.Abs(candidate); absErr == nil {
				return abs, nil
			}
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no python interpreter found under src/agent/.venv; set DEEP_SEAM_PYTHON to override")
}

// seamFreePort asks the OS for an unused loopback TCP port. There is an
// inherent (small) TOCTOU window between closing the probe listener and the
// real process binding it; this is the standard, widely-used pattern for
// test port allocation and is acceptable here.
func seamFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// seamWaitForHealth polls GET /health until it returns 200 or timeout
// elapses.
func seamWaitForHealth(baseURL string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// seamSeedArtifact writes a tiny real PNG to disk and inserts the
// DeepIdentificationArtifact row DeepIdentificationPipelineRunner.Run's
// loadImages reads back (contracts/agent-internal-contract.md §2 images).
func seamSeedArtifact(t *testing.T, db *gorm.DB, jobID, userID uint, role models.DeepArtifactRole) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 180, G: 150, B: 60, A: 255})
	img.Set(1, 1, color.RGBA{R: 60, G: 60, B: 180, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode tiny png fixture: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, string(role)+".png")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write artifact file: %v", err)
	}
	artifact := &models.DeepIdentificationArtifact{
		JobID:    jobID,
		UserID:   userID,
		Role:     role,
		Origin:   models.DeepArtifactOriginUploaded,
		FilePath: path,
		MimeType: "image/png",
		ByteSize: int64(buf.Len()),
	}
	if err := db.Create(artifact).Error; err != nil {
		t.Fatalf("seed artifact: %v", err)
	}
}

func seamMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
