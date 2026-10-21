package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	copilotGradingDefaultDailyCap = 50
	copilotGradingMaxDailyCap     = 500
	copilotGradingMaxMessages     = 20
	copilotGradingMaxMessageChars = 20000
	copilotGradingMaxTools        = 40
	copilotGradingMaxAnswerChars  = 100000
	copilotGradingCallTimeout     = 2 * time.Minute
)

// ErrCopilotGradingRunning reports that a grading cycle is already in progress.
var ErrCopilotGradingRunning = errors.New("copilot grading is already running")

// CopilotGradeTool is one bounded tool fact sent to the judge.
type CopilotGradeTool struct {
	ToolName  string          `json:"tool_name"`
	Result    json.RawMessage `json:"result"`
	Truncated bool            `json:"truncated"`
}

// CopilotGradeProxyRequest asks the agent to grade one completed run.
type CopilotGradeProxyRequest struct {
	LLM      LLMConfig          `json:"llm"`
	Messages []CopilotMessage   `json:"messages"`
	Tools    []CopilotGradeTool `json:"tools"`
	Answer   string             `json:"answer"`
}

// CopilotGradeProxyResponse is the judge's verdict on one run.
type CopilotGradeProxyResponse struct {
	Quality          float64 `json:"quality"`
	AnswersDirectly  bool    `json:"answers_directly"`
	Grounded         bool    `json:"grounded"`
	Accurate         bool    `json:"accurate"`
	CitesSources     string  `json:"cites_sources"`
	NoOfferMenu      bool    `json:"no_offer_menu"`
	NoFalseInability bool    `json:"no_false_inability"`
	Explanation      string  `json:"explanation"`
	JudgeModel       string  `json:"judge_model"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
}

// GradeCopilotRun posts one completed run to the agent's rubric judge.
func (p *AgentProxy) GradeCopilotRun(ctx context.Context, request CopilotGradeProxyRequest) (*CopilotGradeProxyResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal copilot grade request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/copilot/grade", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create copilot grade request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	p.attachInternalCredential(httpReq)
	resp, err := p.requestClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("agent service unreachable: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, agentServiceHTTPError(resp.StatusCode, respBody)
	}
	var result CopilotGradeProxyResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse copilot grade response: %w", err)
	}
	return &result, nil
}

// CopilotRunGrader grades one run; AgentProxy implements it.
type CopilotRunGrader interface {
	GradeCopilotRun(ctx context.Context, request CopilotGradeProxyRequest) (*CopilotGradeProxyResponse, error)
}

// CopilotGradingCycle summarizes the most recent grading cycle.
type CopilotGradingCycle struct {
	TriggerType string     `json:"triggerType"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Graded      int        `json:"graded"`
	Failed      int        `json:"failed"`
	Skipped     string     `json:"skipped,omitempty"`
}

// CopilotGradingStatus is the scheduler status plus the last cycle.
type CopilotGradingStatus struct {
	SchedulerStatus
	LastCycle *CopilotGradingCycle `json:"lastCycle,omitempty"`
}

// CopilotGradingScheduler grades completed Coin Copilot runs once a day.
type CopilotGradingScheduler struct {
	gradeRepo   *repository.CoinCopilotGradeRepository
	copilotRepo *repository.CoinCopilotRepository
	grader      CopilotRunGrader
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	statusMu    sync.RWMutex
	isRunning   bool
	lastCycle   *CopilotGradingCycle
}

// NewCopilotGradingScheduler creates the nightly Copilot grading scheduler.
func NewCopilotGradingScheduler(gradeRepo *repository.CoinCopilotGradeRepository, copilotRepo *repository.CoinCopilotRepository, grader CopilotRunGrader, settingsSvc *SettingsService, logger *Logger) *CopilotGradingScheduler {
	return &CopilotGradingScheduler{
		gradeRepo:   gradeRepo,
		copilotRepo: copilotRepo,
		grader:      grader,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the daily loop.
func (s *CopilotGradingScheduler) Start() {
	s.logger.Info("copilot-grading", "Copilot grading scheduler started")
	scheduleLoop{
		category:     "copilot-grading",
		name:         "copilot grading",
		logger:       s.logger,
		stopCh:       s.stopCh,
		initialDelay: 50 * time.Second,
		next:         func(now time.Time, _ bool) time.Time { return s.nextRun(now) },
		run:          s.runCycle,
	}.Run()
}

// Stop signals the scheduler to shut down.
func (s *CopilotGradingScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow grades pending runs immediately, whether or not the schedule is enabled.
func (s *CopilotGradingScheduler) RunNow() error {
	if !s.claim() {
		return ErrCopilotGradingRunning
	}
	s.runClaimed("manual")
	return nil
}

// StartManualRun starts a manual cycle in the background.
func (s *CopilotGradingScheduler) StartManualRun() error {
	if !s.claim() {
		return ErrCopilotGradingRunning
	}
	go s.runClaimed("manual")
	return nil
}

// GetStatus returns scheduler runtime status.
func (s *CopilotGradingScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()
	return SchedulerStatus{
		Name:      "copilot-grading",
		Enabled:   s.settingsSvc.GetSetting(SettingCopilotGradingEnabled) == "true",
		IsRunning: running,
		NextRunIn: s.timeUntilNextRun(),
	}
}

// GetGradingStatus returns status plus the last cycle summary.
func (s *CopilotGradingScheduler) GetGradingStatus() CopilotGradingStatus {
	status := CopilotGradingStatus{SchedulerStatus: s.GetStatus()}
	s.statusMu.RLock()
	if s.lastCycle != nil {
		cycle := *s.lastCycle
		status.LastCycle = &cycle
	}
	s.statusMu.RUnlock()
	return status
}

func (s *CopilotGradingScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	return s.nextRun(now).Sub(now)
}

func (s *CopilotGradingScheduler) nextRun(now time.Time) time.Time {
	h, m := parseStartTime(s.settingsSvc.GetSetting(SettingCopilotGradingStartTime), 5, 0)
	return dailySchedule{Hour: h, Minute: m, Location: scheduleLocation(s.settingsSvc, s.logger)}.slotAfter(now)
}

func (s *CopilotGradingScheduler) dailyCap() int {
	limit, err := strconv.Atoi(s.settingsSvc.GetSetting(SettingCopilotGradingDailyCap))
	if err != nil || limit < 1 {
		return copilotGradingDefaultDailyCap
	}
	return min(limit, copilotGradingMaxDailyCap)
}

func (s *CopilotGradingScheduler) runCycle() {
	if s.settingsSvc.GetSetting(SettingCopilotGradingEnabled) != "true" {
		s.logger.Debug("copilot-grading", "Copilot grading disabled, skipping cycle")
		return
	}
	if !s.claim() {
		return
	}
	s.runClaimed("scheduled")
}

func (s *CopilotGradingScheduler) claim() bool {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if s.isRunning {
		return false
	}
	s.isRunning = true
	return true
}

func (s *CopilotGradingScheduler) runClaimed(triggerType string) {
	cycle := &CopilotGradingCycle{TriggerType: triggerType, StartedAt: time.Now()}
	defer func() {
		completedAt := time.Now()
		cycle.CompletedAt = &completedAt
		s.statusMu.Lock()
		s.isRunning = false
		s.lastCycle = cycle
		s.statusMu.Unlock()
	}()

	apiKey := s.settingsSvc.GetSetting(SettingAnthropicAPIKey)
	if s.settingsSvc.GetSetting(SettingAIProvider) != "anthropic" || apiKey == "" {
		cycle.Skipped = "Copilot grading needs the Anthropic provider and an API key."
		s.logger.Info("copilot-grading", "%s", cycle.Skipped)
		return
	}
	judge := LLMConfig{Provider: "anthropic", APIKey: apiKey, Model: s.settingsSvc.GetSetting(SettingCopilotGradingJudgeModel)}
	if judge.Model == "" {
		judge.Model = settingDefaults[SettingCopilotGradingJudgeModel]
	}

	runs, err := s.gradeRepo.ListUngradedRuns(s.dailyCap())
	if err != nil {
		s.logger.Error("copilot-grading", "Failed to list ungraded Copilot runs: %v", err)
		cycle.Skipped = "Could not list ungraded runs."
		return
	}
	for i := range runs {
		select {
		case <-s.stopCh:
			return
		default:
		}
		if err := s.gradeRun(&runs[i], judge); err != nil {
			cycle.Failed++
			s.logger.Warn("copilot-grading", "Failed to grade Copilot run %s: %v", runs[i].ID, err)
			continue
		}
		cycle.Graded++
	}
	s.logger.Info("copilot-grading", "%s cycle graded %d runs (%d failed) in %s", triggerType, cycle.Graded, cycle.Failed, time.Since(cycle.StartedAt))
}

func (s *CopilotGradingScheduler) gradeRun(run *models.CoinCopilotRun, judge LLMConfig) error {
	request, err := s.gradeRequest(run, judge)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), copilotGradingCallTimeout)
	defer cancel()
	verdict, err := s.grader.GradeCopilotRun(ctx, request)
	if err != nil {
		return err
	}
	return s.gradeRepo.Save(&models.CoinCopilotRunGrade{
		RunID:             run.ID,
		ThreadID:          run.ThreadID,
		UserID:            run.UserID,
		Quality:           verdict.Quality,
		AnswersDirectly:   verdict.AnswersDirectly,
		Grounded:          verdict.Grounded,
		Accurate:          verdict.Accurate,
		CitesSources:      verdict.CitesSources,
		NoOfferMenu:       verdict.NoOfferMenu,
		NoFalseInability:  verdict.NoFalseInability,
		Explanation:       verdict.Explanation,
		JudgeModel:        verdict.JudgeModel,
		JudgeInputTokens:  verdict.InputTokens,
		JudgeOutputTokens: verdict.OutputTokens,
		GradedAt:          time.Now().UTC(),
	})
}

// gradeRequest rebuilds what the judge sees from the run's latest checkpoint:
// the conversation up to the question, the bounded tool facts, and the answer.
func (s *CopilotGradingScheduler) gradeRequest(run *models.CoinCopilotRun, judge LLMConfig) (CopilotGradeProxyRequest, error) {
	checkpoint, err := s.copilotRepo.GetLatestCheckpoint(run.ID, run.UserID)
	if err != nil {
		return CopilotGradeProxyRequest{}, fmt.Errorf("load checkpoint: %w", err)
	}
	var state CopilotCheckpointState
	if err := json.Unmarshal([]byte(checkpoint.StateJSON), &state); err != nil {
		return CopilotGradeProxyRequest{}, fmt.Errorf("decode checkpoint: %w", err)
	}
	messages := state.Messages
	if n := len(messages); n > 0 && messages[n-1].Role == "assistant" {
		messages = messages[:n-1]
	}
	if len(messages) > copilotGradingMaxMessages {
		messages = messages[len(messages)-copilotGradingMaxMessages:]
	}
	if len(messages) == 0 {
		messages = []CopilotMessage{{Role: "user", Content: run.Goal}}
	}
	bounded := make([]CopilotMessage, 0, len(messages))
	for _, message := range messages {
		bounded = append(bounded, CopilotMessage{Role: message.Role, Content: truncateRunes(message.Content, copilotGradingMaxMessageChars)})
	}
	tools := make([]CopilotGradeTool, 0, min(len(state.CompletedTools), copilotGradingMaxTools))
	for _, tool := range state.CompletedTools {
		if len(tools) == copilotGradingMaxTools {
			break
		}
		result := tool.Result
		if len(result) == 0 {
			result = json.RawMessage("null")
		}
		tools = append(tools, CopilotGradeTool{ToolName: tool.ToolName, Result: result, Truncated: tool.Truncated})
	}
	return CopilotGradeProxyRequest{
		LLM:      judge,
		Messages: bounded,
		Tools:    tools,
		Answer:   truncateRunes(run.FinalAnswer, copilotGradingMaxAnswerChars),
	}, nil
}
