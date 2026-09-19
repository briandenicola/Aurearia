package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
)

const deepAnalysisTruncationLimitation = "Some evidence was omitted to fit the persisted result limit."

type DeepAnalysisHandoffProjection struct {
	Bytes    []byte
	Metadata DeepAnalysisHandoffTruncation
}

type persistedDeepHandoffEvidenceRef struct {
	Provider   string `json:"provider"`
	ClaimIndex *int   `json:"claim_index"`
}

type persistedDeepHandoffField struct {
	Value        string                            `json:"value"`
	Confidence   float64                           `json:"confidence"`
	EvidenceRefs []persistedDeepHandoffEvidenceRef `json:"evidence_refs"`
}

type persistedDeepHandoffDisagreement struct {
	Field      string                            `json:"field"`
	ClaimRefs  []persistedDeepHandoffEvidenceRef `json:"claim_refs"`
	Resolution string                            `json:"resolution"`
}

type persistedDeepHandoffReport struct {
	Narrative           string                                 `json:"narrative"`
	ProposedFields      map[string]persistedDeepHandoffField   `json:"proposed_fields"`
	Disagreements       []persistedDeepHandoffDisagreement     `json:"disagreements"`
	UnresolvedQuestions []string                               `json:"unresolved_questions"`
	Coverage            []DeepAnalysisHandoffCoverage          `json:"coverage"`
	Attributions        []struct {
		Provider   string  `json:"provider"`
		Text       string  `json:"text"`
		Identifier *string `json:"identifier"`
	} `json:"attributions"`
	ImageHypothesis    json.RawMessage `json:"image_hypothesis"`
	PartialSuccess     bool            `json:"partial_success"`
	QuickLookupOutcome string          `json:"quickLookupOutcome"`
}

func decodeStrictPersistedJSON(raw string, destination any) error {
	if strings.TrimSpace(raw) == "" {
		return ErrInvalidCopilotFrame
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return ErrInvalidCopilotFrame
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidCopilotFrame
	}
	return nil
}

// BuildDeepAnalysisHandoffResultFromJob constructs the complete, validated
// conversational projection from the retained Deep report and proposal. It
// deliberately ignores job notes, artifact paths, owner decisions, and apply
// state; those values are not part of the public handoff contract.
func BuildDeepAnalysisHandoffResultFromJob(
	operation string,
	target *DeepAnalysisHandoffTarget,
	label string,
	job *models.DeepIdentificationJob,
) (DeepAnalysisHandoffResult, error) {
	if job == nil || target == nil || target.ID == 0 || label == "" ||
		job.ID == 0 || !models.IsValidDeepJobSourceBinding(job) ||
		(operation != "request" && operation != "status" && operation != "rerun") {
		return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
	}
	result := DeepAnalysisHandoffResult{
		SchemaVersion: 1,
		Operation:     operation,
		Outcome:       "status",
		InputDigest:   job.InputFingerprint,
		ReviewURL:     fmt.Sprintf("/deep-analysis/%d", job.ID),
		Job: &DeepAnalysisHandoffJob{
			ID: job.ID, Source: string(job.Source), Status: string(job.Status),
			Reused: true, CreatedAt: job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		},
	}
	result.Target = &struct {
		Type         string `json:"type"`
		ID           uint   `json:"id"`
		DisplayLabel string `json:"display_label"`
	}{Type: target.Type, ID: target.ID, DisplayLabel: label}
	if job.CompletedAt != nil {
		completed := job.CompletedAt.UTC().Format("2006-01-02T15:04:05Z")
		result.Job.CompletedAt = &completed
	}
	if job.Status == models.DeepJobStatusQueued || job.Status == models.DeepJobStatusRunning {
		if err := ValidateDeepAnalysisHandoffResult(result); err != nil {
			return DeepAnalysisHandoffResult{}, err
		}
		return result, nil
	}
	if job.Status != models.DeepJobStatusCompleted && job.Status != models.DeepJobStatusPartial {
		return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
	}

	var report persistedDeepHandoffReport
	if err := decodeStrictPersistedJSON(job.ReportJSON, &report); err != nil {
		return DeepAnalysisHandoffResult{}, err
	}
	var proposal deepProposalDocument
	if err := decodeStrictPersistedJSON(job.ProposalJSON, &proposal); err != nil ||
		proposal.SchemaVersion != 1 || proposal.Fields == nil {
		return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
	}

	body := &DeepAnalysisHandoffResultBody{
		State: "complete", Narrative: strings.TrimSpace(report.Narrative),
		PartialSuccess: job.Status == models.DeepJobStatusPartial || job.PartialSuccess || report.PartialSuccess,
		Fields: []DeepAnalysisHandoffField{}, Disagreements: []DeepAnalysisHandoffDisagreement{},
		UnresolvedQuestions: []string{}, Coverage: []DeepAnalysisHandoffCoverage{},
		Attributions: []DeepAnalysisHandoffAttribution{}, Limitations: []string{},
	}
	if body.PartialSuccess {
		body.State = "partial"
	}
	if len(report.ProposedFields) == 0 && !body.PartialSuccess {
		body.State = "no_match"
	}

	fieldNames := make([]string, 0, len(report.ProposedFields))
	for name := range report.ProposedFields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)
	allEvidenceIsImage := len(fieldNames) > 0
	for _, name := range fieldNames {
		field := report.ProposedFields[name]
		if strings.TrimSpace(name) == "" || strings.TrimSpace(field.Value) == "" ||
			math.IsNaN(field.Confidence) || math.IsInf(field.Confidence, 0) ||
			field.Confidence < 0 || field.Confidence > 1 || len(field.EvidenceRefs) == 0 {
			return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
		}
		proposalField := proposal.Fields[name]
		if proposalField != nil {
			proposedValue, valueOK := proposalField.Proposed.(string)
			if !valueOK || proposedValue != field.Value ||
				proposalField.Confidence != field.Confidence {
				return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
			}
		}
		projected := DeepAnalysisHandoffField{
			Name: name, Value: field.Value, Confidence: field.Confidence,
			Evidence: []DeepAnalysisHandoffEvidence{},
		}
		externalProviders := make([]string, 0, len(field.EvidenceRefs))
		for _, reference := range field.EvidenceRefs {
			if reference.Provider == "image" {
				if reference.ClaimIndex != nil {
					return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
				}
				continue
			}
			if !oneOf(reference.Provider, "numista", "nomisma", "ngc", "ocre", "rpc") ||
				reference.ClaimIndex == nil || *reference.ClaimIndex < 0 {
				return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
			}
			allEvidenceIsImage = false
			externalProviders = append(externalProviders, reference.Provider)
		}
		if proposalField != nil {
			for index, evidence := range proposalField.Evidence {
				if index >= len(externalProviders) {
					break
				}
				provider := externalProviders[index]
				if evidence.Field != name || strings.TrimSpace(evidence.Value) == "" ||
					math.IsNaN(evidence.Confidence) || math.IsInf(evidence.Confidence, 0) ||
					evidence.Confidence < 0 || evidence.Confidence > 1 {
					return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
				}
				if !deepCitationHostAllowed(provider, evidence.Citation) {
					body.Limitations = appendUniqueString(body.Limitations, "One or more invalid citations were omitted.")
					continue
				}
				summary := strings.TrimSpace(evidence.Excerpt)
				if summary == "" {
					summary = strings.TrimSpace(evidence.Value)
				}
				projected.Evidence = append(projected.Evidence, DeepAnalysisHandoffEvidence{
					Provider: provider, Source: provider, URL: evidence.Citation, Summary: summary,
				})
			}
		}
		if field.Confidence < 0.5 {
			body.Limitations = appendUniqueString(body.Limitations, "One or more proposed fields have low confidence.")
		}
		body.Fields = append(body.Fields, projected)
	}
	body.ImageOnly = allEvidenceIsImage

	for _, disagreement := range report.Disagreements {
		if strings.TrimSpace(disagreement.Field) == "" ||
			!oneOf(disagreement.Resolution, "unresolved", "resolved") ||
			len(disagreement.ClaimRefs) == 0 {
			return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
		}
		providers := make([]string, 0, len(disagreement.ClaimRefs))
		for _, reference := range disagreement.ClaimRefs {
			if reference.Provider == "image" {
				if reference.ClaimIndex != nil {
					return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
				}
			} else if !oneOf(reference.Provider, "numista", "nomisma", "ngc", "ocre", "rpc") ||
				reference.ClaimIndex == nil || *reference.ClaimIndex < 0 {
				return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
			}
			providers = append(providers, reference.Provider)
		}
		sort.Strings(providers)
		body.Disagreements = append(body.Disagreements, DeepAnalysisHandoffDisagreement{
			Field: disagreement.Field,
			Summary: fmt.Sprintf(
				"Persisted source disagreement (%s): %s",
				strings.Join(providers, ", "), disagreement.Resolution,
			),
		})
	}
	for _, question := range report.UnresolvedQuestions {
		question = strings.TrimSpace(question)
		if question == "" {
			return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
		}
		body.UnresolvedQuestions = append(body.UnresolvedQuestions, question)
	}
	seenCoverage := map[string]bool{}
	for _, coverage := range report.Coverage {
		if !oneOf(coverage.Provider, "numista", "nomisma", "ngc", "ocre", "rpc") ||
			!oneOf(coverage.Status, "pending", "running", "contributed", "no_match", "failed",
				"timed_out", "skipped", "not_automated", "unavailable") ||
			seenCoverage[coverage.Provider] {
			return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
		}
		seenCoverage[coverage.Provider] = true
		body.Coverage = append(body.Coverage, coverage)
		if oneOf(coverage.Status, "failed", "timed_out", "not_automated", "unavailable") {
			body.Limitations = appendUniqueString(body.Limitations, "One or more providers did not contribute.")
		}
	}
	seenAttribution := map[string]bool{}
	for _, attribution := range report.Attributions {
		if !oneOf(attribution.Provider, "numista", "nomisma", "ngc", "ocre", "rpc") ||
			strings.TrimSpace(attribution.Text) == "" || seenAttribution[attribution.Provider] {
			return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
		}
		seenAttribution[attribution.Provider] = true
		body.Attributions = append(body.Attributions, DeepAnalysisHandoffAttribution{
			Provider: attribution.Provider, Label: attribution.Text,
		})
	}
	if body.Narrative == "" && body.State != "no_match" {
		return DeepAnalysisHandoffResult{}, ErrInvalidCopilotFrame
	}
	result.Result = body
	if err := ValidateDeepAnalysisHandoffResult(result); err != nil {
		return DeepAnalysisHandoffResult{}, err
	}
	return result, nil
}

func CanonicalDeepAnalysisHandoffResultBytes(result DeepAnalysisHandoffResult) ([]byte, error) {
	normalized, err := cloneDeepAnalysisHandoffResult(result)
	if err != nil {
		return nil, err
	}
	normalized.Truncation = nil
	normalizeDeepAnalysisHandoffResult(&normalized)
	if err := ValidateDeepAnalysisHandoffResult(normalized); err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func ProjectDeepAnalysisHandoffResult(result DeepAnalysisHandoffResult, maximum int) (DeepAnalysisHandoffProjection, error) {
	var projection DeepAnalysisHandoffProjection
	if maximum <= 0 || maximum > DeepAnalysisHandoffMaxPersistedResultBytes {
		return projection, fmt.Errorf("%w: invalid persisted result limit", ErrInvalidCopilotFrame)
	}
	complete, err := CanonicalDeepAnalysisHandoffResultBytes(result)
	if err != nil {
		return projection, err
	}
	digest := sha256.Sum256(complete)
	if result.Outcome == "not_eligible" || result.Outcome == "target_unavailable" {
		if len(complete) > maximum {
			return projection, fmt.Errorf("%w: privacy-safe handoff result exceeds persisted result limit", ErrInvalidCopilotFrame)
		}
		projection.Bytes = complete
		projection.Metadata = DeepAnalysisHandoffTruncation{
			OriginalBytes: len(complete),
			PersistedBytes: len(complete),
			Digest: hex.EncodeToString(digest[:]),
		}
		return projection, nil
	}
	projected, err := cloneDeepAnalysisHandoffResult(result)
	if err != nil {
		return projection, err
	}
	normalizeDeepAnalysisHandoffResult(&projected)
	projected.Truncation = &DeepAnalysisHandoffTruncation{
		OriginalBytes: len(complete),
		Digest:        hex.EncodeToString(digest[:]),
	}

	for {
		encoded, err := marshalDeepAnalysisProjection(&projected)
		if err != nil {
			return projection, err
		}
		if len(encoded) <= maximum {
			projected.Truncation.PersistedBytes = len(encoded)
			encoded, err = marshalDeepAnalysisProjection(&projected)
			if err != nil {
				return projection, err
			}
			if len(encoded) <= maximum && projected.Truncation.PersistedBytes == len(encoded) {
				projection.Bytes = encoded
				projection.Metadata = *projected.Truncation
				return projection, nil
			}
			projected.Truncation.PersistedBytes = len(encoded)
			continue
		}
		if !projected.Truncation.Truncated {
			projected.Truncation.Truncated = true
			projected.Limitations = appendUniqueString(projected.Limitations, deepAnalysisTruncationLimitation)
			if projected.Result != nil {
				projected.Result.Limitations = appendUniqueString(
					projected.Result.Limitations,
					deepAnalysisTruncationLimitation,
				)
			}
			continue
		}
		if omitDeepAnalysisTail(&projected) {
			continue
		}
		return projection, fmt.Errorf("%w: minimal handoff result exceeds persisted result limit", ErrInvalidCopilotFrame)
	}
}

func marshalDeepAnalysisProjection(result *DeepAnalysisHandoffResult) ([]byte, error) {
	for attempts := 0; attempts < 4; attempts++ {
		encoded, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		if result.Truncation.PersistedBytes == len(encoded) {
			return encoded, nil
		}
		result.Truncation.PersistedBytes = len(encoded)
	}
	return json.Marshal(result)
}

func omitDeepAnalysisTail(result *DeepAnalysisHandoffResult) bool {
	body := result.Result
	if body == nil {
		return false
	}
	metadata := result.Truncation
	if count := len(body.UnresolvedQuestions); count > 0 {
		body.UnresolvedQuestions = body.UnresolvedQuestions[:count-1]
		metadata.OmittedQuestions++
		return true
	}
	for index := len(body.Fields) - 1; index >= 0; index-- {
		if count := len(body.Fields[index].Evidence); count > 0 {
			body.Fields[index].Evidence = body.Fields[index].Evidence[:count-1]
			metadata.OmittedEvidence++
			return true
		}
	}
	if count := len(body.Attributions); count > 0 {
		body.Attributions = body.Attributions[:count-1]
		return true
	}
	if count := len(body.Coverage); count > 0 {
		body.Coverage = body.Coverage[:count-1]
		return true
	}
	if count := len(body.Disagreements); count > 0 {
		body.Disagreements = body.Disagreements[:count-1]
		metadata.OmittedDisagreements++
		return true
	}
	if count := len(body.Fields); count > 0 {
		field := body.Fields[count-1]
		metadata.OmittedFields++
		metadata.OmittedEvidence += len(field.Evidence)
		body.Fields = body.Fields[:count-1]
		return true
	}
	return false
}

func normalizeDeepAnalysisHandoffResult(result *DeepAnalysisHandoffResult) {
	if result.Result == nil {
		return
	}
	sort.SliceStable(result.Result.Fields, func(left, right int) bool {
		return result.Result.Fields[left].Name < result.Result.Fields[right].Name
	})
	for index := range result.Result.Fields {
		sort.SliceStable(result.Result.Fields[index].Evidence, func(left, right int) bool {
			leftValue := result.Result.Fields[index].Evidence[left]
			rightValue := result.Result.Fields[index].Evidence[right]
			if leftValue.Provider != rightValue.Provider {
				return leftValue.Provider < rightValue.Provider
			}
			if leftValue.Source != rightValue.Source {
				return leftValue.Source < rightValue.Source
			}
			return leftValue.URL < rightValue.URL
		})
	}
	sort.SliceStable(result.Result.Disagreements, func(left, right int) bool {
		return result.Result.Disagreements[left].Field < result.Result.Disagreements[right].Field
	})
	sort.SliceStable(result.Result.Coverage, func(left, right int) bool {
		return result.Result.Coverage[left].Provider < result.Result.Coverage[right].Provider
	})
	sort.SliceStable(result.Result.Attributions, func(left, right int) bool {
		return result.Result.Attributions[left].Provider < result.Result.Attributions[right].Provider
	})
}

func cloneDeepAnalysisHandoffResult(result DeepAnalysisHandoffResult) (DeepAnalysisHandoffResult, error) {
	var cloned DeepAnalysisHandoffResult
	encoded, err := json.Marshal(result)
	if err != nil {
		return cloned, err
	}
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return cloned, err
	}
	return cloned, nil
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
