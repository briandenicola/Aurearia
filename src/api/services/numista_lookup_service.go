package services

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type NumistaSettingsProvider interface {
	GetSetting(key string) string
	GetNumistaSettings() NumistaSettings
}

type NumistaLookupService struct {
	client    NumistaClient
	cache     *NumistaCache
	scorer    NumistaScorer
	telemetry *NumistaTelemetry
	settings  NumistaSettingsProvider
	clock     NumistaClock
}

func NewNumistaLookupService(
	client NumistaClient,
	cache *NumistaCache,
	scorer NumistaScorer,
	telemetry *NumistaTelemetry,
	settings NumistaSettingsProvider,
	clock NumistaClock,
) *NumistaLookupService {
	if clock == nil {
		clock = realNumistaClock{}
	}
	return &NumistaLookupService{
		client: client, cache: cache, scorer: scorer, telemetry: telemetry,
		settings: settings, clock: clock,
	}
}

func (s *NumistaLookupService) Lookup(ctx context.Context, request models.NumistaLookupRequest) (models.NumistaLookupOutcome, error) {
	if err := request.Validate(); err != nil {
		return models.NumistaLookupOutcome{}, err
	}
	start := s.clock.Now()
	outcome := models.NumistaLookupOutcome{
		Status: models.NumistaStatusUnavailable, EffectiveQuery: request.Query,
		Candidates: []models.NumistaCandidate{}, Stage: "broad",
	}
	if strings.TrimSpace(s.settings.GetSetting(SettingNumistaAPIKey)) == "" {
		outcome.Status = models.NumistaStatusUnconfigured
		outcome.GuidanceCode = "numista_configuration_required"
		s.recordLookup(request, outcome, start, nil)
		return outcome, nil
	}

	config := s.settings.GetNumistaSettings()
	candidates, cacheResult, err := s.cache.doSearchOwned(
		ctx, request.Query, config.SearchResultLimit, config.SearchTTL,
		func(loadCtx context.Context) ([]models.NumistaCandidate, func(), error) {
			loadStart := s.clock.Now()
			operationCtx, recorder := withNumistaOperationRecorder(loadCtx)
			candidates, err := s.client.Search(operationCtx, request.Query, config.SearchResultLimit)
			candidates = sanitizeNumistaCandidates(candidates)
			onAccepted := s.prepareProviderOperation(
				request.Path, request.Query, "broad", loadStart, candidates, err, recorder.Result(),
			)
			return candidates, onAccepted, err
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			s.recordCancellation(request.Path, request.Query, "broad", start, cacheResult)
			return models.NumistaLookupOutcome{}, ctx.Err()
		}
		var numistaErr *NumistaError
		if errors.Is(err, context.Canceled) ||
			(errors.As(err, &numistaErr) && numistaErr.Kind == NumistaErrorCancelled) {
			s.recordCancellation(request.Path, request.Query, "broad", start, cacheResult)
			return models.NumistaLookupOutcome{}, context.Canceled
		}
		var expected bool
		outcome.Status, outcome.GuidanceCode, outcome.RetryAfterSeconds, expected = lookupStatusForError(err)
		if cacheResult == nil || cacheResult.Outcome != NumistaCacheOutcomeLoader {
			s.recordLookup(request, outcome, start, cacheResult)
		}
		if !expected {
			return models.NumistaLookupOutcome{}, err
		}
		return outcome, nil
	}
	outcome.Cache = cacheResult.NumistaCacheMetadata
	outcome.Candidates = s.scorer.Rank(request, candidates)
	if len(outcome.Candidates) == 0 {
		outcome.Status = models.NumistaStatusEmpty
		outcome.GuidanceCode = "revise_numista_query"
	} else {
		outcome.Status = models.NumistaStatusSuccess
	}

	if cacheResult.Outcome != NumistaCacheOutcomeLoader {
		s.recordLookup(request, outcome, start, cacheResult)
	}
	return outcome, nil
}

func (s *NumistaLookupService) LookupDetail(
	ctx context.Context,
	path models.NumistaLookupPath,
	id int,
) (models.NumistaCandidate, *models.NumistaCacheMetadata, error) {
	if path != models.NumistaLookupPathDirect && path != models.NumistaLookupPathPhoto {
		return models.NumistaCandidate{}, nil, errors.New("path must be direct or photo")
	}
	if id <= 0 {
		return models.NumistaCandidate{}, nil, errors.New("Numista detail ID must be positive")
	}
	start := s.clock.Now()
	correlation := "detail:" + strconv.Itoa(id)
	if strings.TrimSpace(s.settings.GetSetting(SettingNumistaAPIKey)) == "" {
		err := &NumistaError{Kind: NumistaErrorUnconfigured}
		s.recordDetailReuse(path, correlation, start, nil, models.NumistaStatusUnconfigured, err)
		return models.NumistaCandidate{}, nil, err
	}

	config := s.settings.GetNumistaSettings()
	candidate, cacheResult, err := s.cache.doDetailOwned(
		ctx, id, config.DetailTTL,
		func(loadCtx context.Context) (models.NumistaCandidate, func(), error) {
			loadStart := s.clock.Now()
			operationCtx, recorder := withNumistaOperationRecorder(loadCtx)
			candidate, err := s.client.Detail(operationCtx, id)
			candidates := []models.NumistaCandidate(nil)
			if err == nil {
				candidate.Title = strings.TrimSpace(candidate.Title)
				if candidate.ID != id || candidate.Title == "" {
					err = &NumistaError{Kind: NumistaErrorMalformedResponse}
				} else {
					candidate.CanonicalURL, _ = models.CanonicalNumistaURL(candidate.ID)
					candidate.EnrichmentState = models.NumistaEnrichmentEnriched
					candidates = []models.NumistaCandidate{candidate}
				}
			}
			onAccepted := s.prepareProviderOperation(
				path, correlation, "detail", loadStart, candidates, err, recorder.Result(),
			)
			return candidate, onAccepted, err
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			s.recordCancellation(path, correlation, "detail", start, cacheResult)
			return models.NumistaCandidate{}, nil, ctx.Err()
		}
		if cacheResult == nil || cacheResult.Outcome != NumistaCacheOutcomeLoader {
			status, _, _, _ := lookupStatusForError(err)
			s.recordDetailReuse(path, correlation, start, cacheResult, status, err)
		}
		return models.NumistaCandidate{}, nil, err
	}
	if cacheResult.Outcome != NumistaCacheOutcomeLoader {
		s.recordDetailReuse(path, correlation, start, cacheResult, models.NumistaStatusSuccess, nil)
	}
	return candidate, cacheResult.NumistaCacheMetadata, nil
}

func sanitizeNumistaCandidates(candidates []models.NumistaCandidate) []models.NumistaCandidate {
	sanitized := make([]models.NumistaCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.Title = strings.TrimSpace(candidate.Title)
		if candidate.ID <= 0 || candidate.Title == "" {
			continue
		}
		candidate.CanonicalURL, _ = models.CanonicalNumistaURL(candidate.ID)
		candidate.EnrichmentState = models.NumistaEnrichmentNotRequested
		sanitized = append(sanitized, candidate)
	}
	return sanitized
}

func (s *NumistaLookupService) LegacySearch(ctx context.Context, query string) (models.LegacyNumistaSearchResponse, error) {
	outcome, err := s.Lookup(ctx, models.NumistaLookupRequest{
		Query: query, Path: models.NumistaLookupPathDirect, Evidence: models.NumistaEvidence{},
	})
	if err != nil {
		return models.LegacyNumistaSearchResponse{}, err
	}
	if outcome.Status != models.NumistaStatusSuccess && outcome.Status != models.NumistaStatusEmpty {
		return models.LegacyNumistaSearchResponse{}, &NumistaLookupStatusError{Status: outcome.Status}
	}
	response := models.LegacyNumistaSearchResponse{Types: make([]models.LegacyNumistaType, 0, len(outcome.Candidates))}
	for _, candidate := range outcome.Candidates {
		item := models.LegacyNumistaType{
			ID: candidate.ID, Title: candidate.Title, MinYear: candidate.MinYear, MaxYear: candidate.MaxYear,
			ObverseThumbnail: candidate.ObverseThumbnail, ReverseThumbnail: candidate.ReverseThumbnail,
		}
		if candidate.Issuer != "" {
			item.Issuer = &models.LegacyNumistaIssuer{Name: candidate.Issuer}
		}
		response.Types = append(response.Types, item)
	}
	response.Count = len(response.Types)
	return response, nil
}

type NumistaLookupStatusError struct {
	Status models.NumistaLookupStatus
}

func (e *NumistaLookupStatusError) Error() string { return "Numista lookup unavailable" }

func lookupStatusForError(err error) (models.NumistaLookupStatus, string, *int, bool) {
	var numistaErr *NumistaError
	if !errors.As(err, &numistaErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			return models.NumistaStatusTimeout, "retry_numista_lookup", nil, true
		}
		return models.NumistaStatusUnavailable, "retry_numista_lookup", nil, false
	}
	switch numistaErr.Kind {
	case NumistaErrorInvalidRequest:
		return models.NumistaStatusEmpty, "revise_numista_query", nil, true
	case NumistaErrorUnconfigured, NumistaErrorUnauthorized:
		return models.NumistaStatusUnconfigured, "numista_configuration_required", nil, true
	case NumistaErrorQuotaLimited:
		return models.NumistaStatusQuotaLimited, "numista_quota_limited",
			positiveRetryAfter(numistaErr.RetryAfterSeconds), true
	case NumistaErrorTimeout:
		return models.NumistaStatusTimeout, "retry_numista_lookup", nil, true
	default:
		return models.NumistaStatusUnavailable, "retry_numista_lookup", nil, true
	}
}

func positiveRetryAfter(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	retryAfter := *value
	return &retryAfter
}

func (s *NumistaLookupService) recordLookup(
	request models.NumistaLookupRequest,
	outcome models.NumistaLookupOutcome,
	start time.Time,
	cacheResult *NumistaCacheResult,
) {
	if s.telemetry == nil {
		return
	}
	cacheOutcome := NumistaCacheOutcomeBypass
	if cacheResult != nil {
		cacheOutcome = cacheResult.Outcome
	}
	if cacheOutcome == NumistaCacheOutcomeCoalescedWaiter {
		s.telemetry.Record(NumistaTelemetryEvent{
			OccurredAt: s.clock.Now(), Path: request.Path, Operation: "broad",
			CacheOutcome:      cacheOutcome,
			CorrelationDigest: NumistaCorrelationDigest(request.Path, request.Query),
		})
		return
	}
	s.telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: s.clock.Now(), Path: request.Path, Operation: "broad", Status: outcome.Status,
		CacheOutcome:        cacheOutcome,
		ElapsedMilliseconds: s.clock.Now().Sub(start).Milliseconds(),
		CandidateCount:      len(outcome.Candidates), RetryAfterSeconds: outcome.RetryAfterSeconds,
		CorrelationDigest: NumistaCorrelationDigest(request.Path, request.Query),
	})
}

func (s *NumistaLookupService) recordCancellation(
	path models.NumistaLookupPath,
	correlation string,
	operation string,
	start time.Time,
	cacheResult *NumistaCacheResult,
) {
	if s.telemetry == nil {
		return
	}
	cacheOutcome := NumistaCacheOutcomeBypass
	if cacheResult != nil {
		cacheOutcome = cacheResult.Outcome
	}
	if cacheOutcome == NumistaCacheOutcomeLoader {
		cacheOutcome = NumistaCacheOutcomeBypass
	}
	s.telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: s.clock.Now(), Path: path, Operation: operation, CacheOutcome: cacheOutcome,
		ElapsedMilliseconds: boolDuration(cacheOutcome != NumistaCacheOutcomeCoalescedWaiter, s.clock.Now().Sub(start)),
		CorrelationDigest:   NumistaCorrelationDigest(path, correlation), Cancelled: true,
	})
}

func (s *NumistaLookupService) prepareProviderOperation(
	path models.NumistaLookupPath,
	correlation string,
	operation string,
	start time.Time,
	candidates []models.NumistaCandidate,
	err error,
	clientOperation NumistaClientOperation,
) func() {
	if s.telemetry == nil {
		return nil
	}
	status := models.NumistaStatusSuccess
	var retryAfter *int
	if err != nil {
		var numistaErr *NumistaError
		if errors.Is(err, context.Canceled) ||
			(errors.As(err, &numistaErr) && numistaErr.Kind == NumistaErrorCancelled) {
			return nil
		} else {
			status, _, retryAfter, _ = lookupStatusForError(err)
		}
	} else if operation == "broad" && len(candidates) == 0 {
		status = models.NumistaStatusEmpty
	}
	event := NumistaTelemetryEvent{
		OccurredAt: s.clock.Now(), Path: path, Operation: operation, Status: status,
		CacheOutcome:        NumistaCacheOutcomeLoader,
		ElapsedMilliseconds: s.clock.Now().Sub(start).Milliseconds(),
		CandidateCount:      len(candidates),
		RetryCount:          clientOperation.RetryCount,
		RetryAfterSeconds:   retryAfter,
		CorrelationDigest:   NumistaCorrelationDigest(path, correlation),
	}
	if operation == "detail" {
		event.DetailAttemptCount = 1
		if err == nil {
			event.DetailSuccessCount = 1
		} else {
			event.DetailFailureCount = 1
		}
	}
	return func() {
		s.telemetry.Record(event)
	}
}

func (s *NumistaLookupService) recordDetailReuse(
	path models.NumistaLookupPath,
	correlation string,
	start time.Time,
	cacheResult *NumistaCacheResult,
	status models.NumistaLookupStatus,
	err error,
) {
	if s.telemetry == nil {
		return
	}
	cacheOutcome := NumistaCacheOutcomeBypass
	if cacheResult != nil {
		cacheOutcome = cacheResult.Outcome
	}
	if cacheOutcome == NumistaCacheOutcomeCoalescedWaiter {
		s.telemetry.Record(NumistaTelemetryEvent{
			OccurredAt: s.clock.Now(), Path: path, Operation: "detail",
			CacheOutcome: cacheOutcome, CorrelationDigest: NumistaCorrelationDigest(path, correlation),
		})
		return
	}
	var retryAfter *int
	var numistaErr *NumistaError
	if errors.As(err, &numistaErr) {
		retryAfter = positiveRetryAfter(numistaErr.RetryAfterSeconds)
	}
	s.telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: s.clock.Now(), Path: path, Operation: "detail", Status: status,
		CacheOutcome:        cacheOutcome,
		ElapsedMilliseconds: s.clock.Now().Sub(start).Milliseconds(),
		CandidateCount:      boolToCount(err == nil),
		RetryAfterSeconds:   retryAfter,
		CorrelationDigest:   NumistaCorrelationDigest(path, correlation),
	})
}

func boolToCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func boolDuration(include bool, value time.Duration) int64 {
	if !include {
		return 0
	}
	return value.Milliseconds()
}
