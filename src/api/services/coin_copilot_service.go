package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrCopilotDisabled            = errors.New("coin copilot is disabled")
	ErrCopilotUnavailable         = errors.New("coin copilot is unavailable")
	ErrCopilotNotFound            = errors.New("coin copilot resource not found")
	ErrCopilotInvalidRequest      = errors.New("invalid coin copilot request")
	ErrCopilotIdempotencyConflict = errors.New("idempotency key conflict")
	ErrCopilotCapacity            = errors.New("coin copilot active-run capacity reached")
	ErrCopilotQueueFull           = errors.New("coin copilot queue is full")
	ErrCopilotStateConflict       = errors.New("coin copilot state conflict")
	ErrCopilotThreadHasActiveRun  = errors.New("coin copilot thread has an active run")
)

const copilotDefaultMaxConcurrentTools = 3

type CoinCopilotStartInput struct {
	ThreadID       string
	Goal           string
	AppContext     map[string]any
	IdempotencyKey string
}

type CoinCopilotResumeInput struct {
	Answer                    string
	ExpectedCheckpointVersion int64
	IdempotencyKey            string
}

type CoinCopilotCapability struct {
	Mode                      string  `json:"mode"`
	Enabled                   bool    `json:"enabled"`
	ModelToolCallingSupported bool    `json:"modelToolCallingSupported"`
	Reason                    *string `json:"reason"`
}

type CoinCopilotService struct {
	repo         *repository.CoinCopilotRepository
	settingsSvc  *SettingsService
	proxy        *AgentProxy
	tokenSvc     *InternalTokenService
	broker       *CoinCopilotBroker
	logger       *Logger
	toolsBaseURL string

	wake            chan struct{}
	cancelMu        sync.Mutex
	cancels         map[string]context.CancelFunc
	toolMu          sync.Mutex
	activeToolCalls map[string]*copilotToolCallTracker
}

type copilotToolCallTracker struct {
	lastPersistedCount int
	calls              map[string]copilotTrackedToolCall
}

type copilotTrackedToolCall struct {
	completed bool
	persisted bool
}

func NewCoinCopilotService(repo *repository.CoinCopilotRepository, settingsSvc *SettingsService, proxy *AgentProxy, tokenSvc *InternalTokenService, logger *Logger, toolsBaseURL string) *CoinCopilotService {
	return &CoinCopilotService{
		repo: repo, settingsSvc: settingsSvc, proxy: proxy, tokenSvc: tokenSvc, logger: logger,
		toolsBaseURL: strings.TrimRight(toolsBaseURL, "/"), broker: NewCoinCopilotBroker(3),
		wake: make(chan struct{}, 1), cancels: make(map[string]context.CancelFunc),
		activeToolCalls: make(map[string]*copilotToolCallTracker),
	}
}

func (s *CoinCopilotService) Broker() *CoinCopilotBroker { return s.broker }

func (s *CoinCopilotService) Capability() CoinCopilotCapability {
	settings := s.settingsSvc.GetCoinCopilotSettings()
	if !settings.Enabled {
		reason := "disabled"
		return CoinCopilotCapability{Mode: "legacy", Enabled: false, ModelToolCallingSupported: false, Reason: &reason}
	}
	if !settings.Valid {
		reason := "temporarily_unavailable"
		return CoinCopilotCapability{Mode: "legacy", Enabled: true, ModelToolCallingSupported: false, Reason: &reason}
	}
	llm, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		reason := "provider_unconfigured"
		return CoinCopilotCapability{Mode: "legacy", Enabled: true, ModelToolCallingSupported: false, Reason: &reason}
	}
	if !s.proxy.SupportsCoinCopilot(llm) {
		reason := "model_tool_calling_unsupported"
		return CoinCopilotCapability{Mode: "legacy", Enabled: true, ModelToolCallingSupported: false, Reason: &reason}
	}
	return CoinCopilotCapability{Mode: "copilot", Enabled: true, ModelToolCallingSupported: true}
}

func (s *CoinCopilotService) Start(userID uint, input CoinCopilotStartInput) (*models.CoinCopilotRun, bool, error) {
	settings := s.settingsSvc.GetCoinCopilotSettings()
	if !settings.Enabled {
		return nil, false, ErrCopilotDisabled
	}
	if s.Capability().Mode != "copilot" {
		return nil, false, ErrCopilotUnavailable
	}
	goal := strings.TrimSpace(input.Goal)
	if userID == 0 || goal == "" || len(goal) > CoinCopilotMaxPromptBytes || !validIdempotencyKey(input.IdempotencyKey) {
		return nil, false, ErrCopilotInvalidRequest
	}
	appContext, err := canonicalAppContext(input.AppContext)
	if err != nil {
		return nil, false, ErrCopilotInvalidRequest
	}
	keyHash := sha256Text(input.IdempotencyKey)
	fingerprint := sha256Text(goal + "\n" + appContext + "\n" + input.ThreadID)
	existing, err := s.repo.FindRunByStartKey(userID, keyHash)
	if err == nil {
		if existing.StartRequestFingerprint != fingerprint {
			return nil, false, ErrCopilotIdempotencyConflict
		}
		return existing, true, nil
	}
	if !repository.IsRecordNotFound(err) {
		return nil, false, err
	}
	thread := &models.CoinCopilotThread{UserID: userID}
	if input.ThreadID == "" {
		thread.ID = newCopilotID("cct_")
		thread.Title = goal
		if len(thread.Title) > 200 {
			thread.Title = thread.Title[:200]
		}
	} else {
		thread.ID = input.ThreadID
		owned, _, err := s.repo.GetThread(input.ThreadID, userID)
		if err != nil {
			return nil, false, ErrCopilotNotFound
		}
		thread = owned
	}
	run := &models.CoinCopilotRun{
		ID: newCopilotID("ccr_"), ThreadID: thread.ID, UserID: userID, Status: models.CopilotRunQueued,
		Goal: goal, AppContextJSON: appContext, StartIdempotencyKeyHash: keyHash, StartRequestFingerprint: fingerprint,
		MaxIterations: settings.MaxReasoningIterations, MaxToolCalls: settings.MaxToolCalls,
		MaxConcurrentTools: copilotDefaultMaxConcurrentTools, HardTimeoutSeconds: int(settings.HardTimeout.Seconds()),
		MaxPersistedToolResultBytes: settings.MaxPersistedToolResultBytes,
	}
	admitted, reused, err := s.repo.AdmitRun(thread, run, settings.MaxActivePerUser, settings.QueueDepth)
	switch {
	case errors.Is(err, repository.ErrCopilotStartKeyConflict):
		return nil, false, ErrCopilotIdempotencyConflict
	case errors.Is(err, repository.ErrCopilotOwnerCapacity):
		return nil, false, ErrCopilotCapacity
	case errors.Is(err, repository.ErrCopilotQueueCapacity):
		return nil, false, ErrCopilotQueueFull
	case err != nil:
		return nil, false, err
	case reused:
		return admitted, true, nil
	}
	s.notifyWorkers()
	return admitted, false, nil
}

func (s *CoinCopilotService) GetRun(userID uint, runID string) (*models.CoinCopilotRun, error) {
	run, err := s.repo.GetRun(runID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrCopilotNotFound
		}
		return nil, err
	}
	return run, nil
}

func (s *CoinCopilotService) GetThread(userID uint, threadID string) (*models.CoinCopilotThread, []models.CoinCopilotRun, error) {
	thread, runs, err := s.repo.GetThread(threadID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, nil, ErrCopilotNotFound
		}
		return nil, nil, err
	}
	return thread, runs, nil
}

func (s *CoinCopilotService) DeleteThread(userID uint, threadID string) error {
	err := s.repo.DeleteThread(threadID, userID)
	switch {
	case repository.IsRecordNotFound(err):
		return ErrCopilotNotFound
	case errors.Is(err, repository.ErrCopilotThreadActive):
		return ErrCopilotThreadHasActiveRun
	default:
		return err
	}
}

func (s *CoinCopilotService) Cancel(userID uint, runID string) (*models.CoinCopilotRun, bool, error) {
	run, immediate, err := s.repo.RequestCancel(runID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, false, ErrCopilotNotFound
		}
		if errors.Is(err, repository.ErrCopilotTransitionConflict) {
			current, getErr := s.GetRun(userID, runID)
			return current, false, errors.Join(ErrCopilotStateConflict, getErr)
		}
		return nil, false, err
	}
	if immediate {
		s.tokenSvc.RevokeCopilotExecution(run.ExecutionID)
		s.broker.Publish(runID)
	} else if run.Status == models.CopilotRunCancelRequested {
		s.cancelExecution(run.ExecutionID)
	}
	return run, immediate, nil
}

func (s *CoinCopilotService) Resume(userID uint, runID string, input CoinCopilotResumeInput) (*models.CoinCopilotRun, bool, error) {
	if !s.settingsSvc.GetCoinCopilotSettings().Enabled || s.Capability().Mode != "copilot" {
		return nil, false, ErrCopilotUnavailable
	}
	answer := strings.TrimSpace(input.Answer)
	if answer == "" || len(answer) > CoinCopilotMaxPromptBytes || input.ExpectedCheckpointVersion < 1 || !validIdempotencyKey(input.IdempotencyKey) {
		return nil, false, ErrCopilotInvalidRequest
	}
	keyHash := sha256Text(input.IdempotencyKey)
	fingerprint := sha256Text(fmt.Sprintf("%d\n%s", input.ExpectedCheckpointVersion, answer))
	replayed, err := s.repo.FindResumeRequest(runID, userID, keyHash)
	if err == nil {
		if replayed.RequestFingerprint != fingerprint {
			return nil, false, ErrCopilotIdempotencyConflict
		}
		run, getErr := s.GetRun(userID, runID)
		return run, true, getErr
	}
	if !repository.IsRecordNotFound(err) {
		return nil, false, err
	}
	current, err := s.GetRun(userID, runID)
	if err != nil {
		return nil, false, err
	}
	checkpoint, err := s.repo.GetLatestCheckpoint(runID, userID)
	if err != nil || checkpoint.Version != input.ExpectedCheckpointVersion {
		return nil, false, ErrCopilotStateConflict
	}
	var state CopilotCheckpointState
	if err := json.Unmarshal([]byte(checkpoint.StateJSON), &state); err != nil {
		return nil, false, ErrCopilotStateConflict
	}
	state.Messages = append(state.Messages, CopilotMessage{Role: "user", Content: answer})
	state.PendingClarification = nil
	state.NextAction = "continue"
	stateJSON, _ := json.Marshal(state)
	executionID := newCopilotID("cce_")
	run, err := s.repo.Resume(runID, userID, keyHash, fingerprint, executionID, answer, current.CheckpointVersion, string(stateJSON), CopilotCheckpointDigest(string(stateJSON)))
	if err != nil {
		if replayed, lookupErr := s.repo.FindResumeRequest(runID, userID, keyHash); lookupErr == nil {
			if replayed.RequestFingerprint != fingerprint {
				return nil, false, ErrCopilotIdempotencyConflict
			}
			run, getErr := s.GetRun(userID, runID)
			return run, true, getErr
		}
		if errors.Is(err, repository.ErrCopilotTransitionConflict) {
			return nil, false, ErrCopilotStateConflict
		}
		return nil, false, err
	}
	s.broker.Publish(runID)
	s.notifyWorkers()
	return run, false, nil
}

func (s *CoinCopilotService) ListEventsSince(userID uint, runID string, since int64) ([]models.CoinCopilotEvent, error) {
	if _, err := s.GetRun(userID, runID); err != nil {
		return nil, err
	}
	return s.repo.ListEventsSince(runID, userID, since)
}

func (s *CoinCopilotService) RecoverAndPrune() error {
	settings := s.settingsSvc.GetCoinCopilotSettings()
	recovered, err := s.repo.RecoverStale(time.Now().UTC().Add(-45*time.Second), settings.ResumeWindow)
	if err != nil {
		return err
	}
	expired, err := s.repo.ExpirePaused(time.Now().UTC())
	if err != nil {
		return err
	}
	for _, runID := range append(recovered, expired...) {
		s.broker.Publish(runID)
	}
	return s.repo.Prune(time.Now().UTC().Add(-settings.EventRetention), time.Now().UTC().Add(-settings.CheckpointRetention))
}

func validIdempotencyKey(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}

func canonicalAppContext(value map[string]any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	for key, raw := range value {
		switch key {
		case "route":
			route, ok := raw.(string)
			if !ok || len(route) > 2048 {
				return "", ErrCopilotInvalidRequest
			}
		case "activeCoinId":
			number, ok := raw.(float64)
			if !ok || number < 1 || number != float64(uint(number)) {
				return "", ErrCopilotInvalidRequest
			}
		default:
			return "", ErrCopilotInvalidRequest
		}
	}
	encoded, err := json.Marshal(value)
	return string(encoded), err
}

func sha256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func newCopilotID(prefix string) string {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return prefix + base64.RawURLEncoding.EncodeToString(raw)
}

func (s *CoinCopilotService) notifyWorkers() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *CoinCopilotService) registerExecution(executionID string, cancel context.CancelFunc) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	s.cancels[executionID] = cancel
}

func (s *CoinCopilotService) AuthorizeToolCall(claims *CopilotExecutionClaims, toolCallID, tool string) error {
	if claims == nil || toolCallID == "" || !IsCoinCopilotCallbackTool(tool) {
		return ErrCopilotInvalidRequest
	}
	allowed := false
	for _, candidate := range claims.AllowedTools {
		if candidate == tool {
			allowed = true
			break
		}
	}
	if !allowed {
		return ErrCopilotInvalidRequest
	}
	run, err := s.repo.GetCurrentExecution(claims.RunID, claims.ExecutionID, claims.UserID)
	if err != nil {
		return ErrCopilotStateConflict
	}
	if run.ToolCallCount >= run.MaxToolCalls || run.ExecutionStartedAt == nil ||
		time.Since(*run.ExecutionStartedAt) >= time.Duration(run.HardTimeoutSeconds)*time.Second {
		return ErrCopilotStateConflict
	}
	s.toolMu.Lock()
	defer s.toolMu.Unlock()
	tracker := s.activeToolCalls[claims.ExecutionID]
	if tracker == nil {
		tracker = &copilotToolCallTracker{
			lastPersistedCount: run.ToolCallCount,
			calls:              make(map[string]copilotTrackedToolCall),
		}
		s.activeToolCalls[claims.ExecutionID] = tracker
	}
	if _, exists := tracker.calls[toolCallID]; exists {
		return ErrCopilotIdempotencyConflict
	}
	persistedDelta := max(run.ToolCallCount-tracker.lastPersistedCount, 0)
	for callID, call := range tracker.calls {
		if persistedDelta == 0 {
			break
		}
		if call.completed && !call.persisted {
			call.persisted = true
			tracker.calls[callID] = call
			persistedDelta--
		}
	}
	tracker.lastPersistedCount = max(tracker.lastPersistedCount, run.ToolCallCount)
	inFlight := 0
	unpersistedSuccessful := 0
	for _, call := range tracker.calls {
		if !call.completed {
			inFlight++
		} else if !call.persisted {
			unpersistedSuccessful++
		}
	}
	if inFlight >= run.MaxConcurrentTools || run.ToolCallCount+unpersistedSuccessful+inFlight >= run.MaxToolCalls {
		return ErrCopilotStateConflict
	}
	tracker.calls[toolCallID] = copilotTrackedToolCall{}
	return nil
}

func (s *CoinCopilotService) FinishToolCall(executionID, toolCallID string, success bool) {
	s.toolMu.Lock()
	defer s.toolMu.Unlock()
	tracker := s.activeToolCalls[executionID]
	if tracker == nil {
		return
	}
	if success {
		call := tracker.calls[toolCallID]
		call.completed = true
		tracker.calls[toolCallID] = call
	} else {
		delete(tracker.calls, toolCallID)
	}
}

func (s *CoinCopilotService) clearToolCalls(executionID string) {
	s.toolMu.Lock()
	defer s.toolMu.Unlock()
	delete(s.activeToolCalls, executionID)
}

func (s *CoinCopilotService) unregisterExecution(executionID string) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	delete(s.cancels, executionID)
}

func (s *CoinCopilotService) cancelExecution(executionID string) {
	s.cancelMu.Lock()
	cancel := s.cancels[executionID]
	s.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}
