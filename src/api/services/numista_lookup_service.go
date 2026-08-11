package services

import (
	"context"
	"errors"
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
		s.recordLookup(request, outcome, start, false)
		return outcome, nil
	}

	config := s.settings.GetNumistaSettings()
	candidates, cacheMetadata, err := s.cache.DoSearch(
		ctx, request.Query, config.SearchResultLimit, config.SearchTTL,
		func(loadCtx context.Context) ([]models.NumistaCandidate, error) {
			return s.client.Search(loadCtx, request.Query, config.SearchResultLimit)
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			return models.NumistaLookupOutcome{}, ctx.Err()
		}
		var numistaErr *NumistaError
		if errors.Is(err, context.Canceled) ||
			(errors.As(err, &numistaErr) && numistaErr.Kind == NumistaErrorCancelled) {
			return models.NumistaLookupOutcome{}, context.Canceled
		}
		outcome.Status, outcome.GuidanceCode, outcome.RetryAfterSeconds = lookupStatusForError(err)
		s.recordLookup(request, outcome, start, false)
		return outcome, nil
	}
	candidates = sanitizeNumistaCandidates(candidates)
	outcome.Cache = cacheMetadata
	outcome.Candidates = s.scorer.Rank(request, candidates)
	if len(outcome.Candidates) == 0 {
		outcome.Status = models.NumistaStatusEmpty
		outcome.GuidanceCode = "revise_numista_query"
	} else {
		outcome.Status = models.NumistaStatusSuccess
	}

	s.recordLookup(request, outcome, start, cacheMetadata != nil && cacheMetadata.Hit)
	return outcome, nil
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

func lookupStatusForError(err error) (models.NumistaLookupStatus, string, *int) {
	var numistaErr *NumistaError
	if !errors.As(err, &numistaErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			return models.NumistaStatusTimeout, "retry_numista_lookup", nil
		}
		return models.NumistaStatusUnavailable, "retry_numista_lookup", nil
	}
	switch numistaErr.Kind {
	case NumistaErrorUnconfigured:
		return models.NumistaStatusUnconfigured, "numista_configuration_required", nil
	case NumistaErrorQuotaLimited:
		return models.NumistaStatusQuotaLimited, "numista_quota_limited", numistaErr.RetryAfterSeconds
	case NumistaErrorTimeout:
		return models.NumistaStatusTimeout, "retry_numista_lookup", nil
	default:
		return models.NumistaStatusUnavailable, "retry_numista_lookup", nil
	}
}

func (s *NumistaLookupService) recordLookup(
	request models.NumistaLookupRequest,
	outcome models.NumistaLookupOutcome,
	start time.Time,
	cacheHit bool,
) {
	if s.telemetry == nil {
		return
	}
	s.telemetry.Record(NumistaTelemetryEvent{
		OccurredAt: s.clock.Now(), Path: request.Path, Operation: "broad", Status: outcome.Status,
		CacheHit: cacheHit, Refreshed: outcome.Cache != nil && !outcome.Cache.Hit,
		ElapsedMilliseconds: s.clock.Now().Sub(start).Milliseconds(),
		CandidateCount:      len(outcome.Candidates), RetryAfterSeconds: outcome.RetryAfterSeconds,
		CorrelationDigest: NumistaCorrelationDigest(request.Path, request.Query),
	})
}
