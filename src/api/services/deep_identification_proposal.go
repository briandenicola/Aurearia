package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// Errors returned by DeepIdentificationProposalService. Handlers map these
// to the exact status/code vocabulary in
// contracts/deep-identification.openapi.yaml (§ proposal/apply).
var (
	ErrDeepProposalNotFound         = errors.New("deep identification job not found")
	ErrDeepProposalFieldNotAllowed  = errors.New("field is not writable through the allowed coin/draft update paths")
	ErrDeepProposalNotReady         = errors.New("job has no proposal to edit or apply yet")
	ErrDeepProposalAlreadyApplied   = errors.New("proposal has already been applied")
	ErrDeepProposalNoAcceptedFields = errors.New("no accepted fields to apply")
	ErrDeepProposalSourceMissing    = errors.New("source coin no longer exists")
	ErrDeepProposalTargetMismatch   = errors.New("apply target does not match this job's source")
)

// deepProposalCoinFieldAllowlist maps a Proposal.fields JSON key to the
// exact models.Coin struct field name writable through
// CoinService.UpdateCoinWithFields (data-model.md §7). Any field not
// present here is rejected at both PATCH-proposal and apply time - this is
// the only write-surface allowlist for the "target: coin" apply path
// (Principle IV / F012 allowlist precedent, no silent new write surface).
var deepProposalCoinFieldAllowlist = map[string]string{
	"denomination":       "Denomination",
	"ruler":              "Ruler",
	"era":                "Era",
	"dateRange":          "DateRange",
	"mint":               "Mint",
	"material":           "Material",
	"weightGrams":        "WeightGrams",
	"diameterMm":         "DiameterMm",
	"obverseInscription": "ObverseInscription",
	"reverseInscription": "ReverseInscription",
	"obverseDescription": "ObverseDescription",
	"reverseDescription": "ReverseDescription",
	"notes":              "Notes",
}

// deepProposalDraftFieldAllowlist maps a Proposal.fields JSON key to the
// QuickCaptureDraft field it may seed on the "target: draft" apply path.
// QuickCaptureDraft intentionally has no denomination/ruler/mint/material
// columns of its own (identification detail is only recorded on
// models.Coin once promoted), so those keys are simply not writable for a
// draft target - proposing them is fine, applying them to a draft is not.
var deepProposalDraftFieldAllowlist = map[string]string{
	"workingTitle": "WorkingTitle",
	"era":          "Era",
	"dateRange":    "DateRange",
	"notes":        "Notes",
}

// deepProposalClaim mirrors the `Claim` schema (contracts/deep-identification.openapi.yaml).
type deepProposalClaim struct {
	Field      string  `json:"field"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence,omitempty"`
	Citation   string  `json:"citation"`
	Excerpt    string  `json:"excerpt,omitempty"`
}

// deepProposalFieldEntry mirrors one entry of `Proposal.fields` (data-model.md §7).
type deepProposalFieldEntry struct {
	Proposed    any                 `json:"proposed"`
	Confidence  float64             `json:"confidence,omitempty"`
	Evidence    []deepProposalClaim `json:"evidence,omitempty"`
	OwnerEdited bool                `json:"ownerEdited"`
	OwnerValue  any                 `json:"ownerValue"`
	Accepted    *bool               `json:"accepted"`
}

// deepProposalDocument mirrors the `Proposal` schema.
type deepProposalDocument struct {
	SchemaVersion           int                               `json:"schemaVersion"`
	TargetCoinID            *uint                             `json:"targetCoinId,omitempty"`
	Fields                  map[string]*deepProposalFieldEntry `json:"fields"`
	SourceReportGeneratedAt string                            `json:"sourceReportGeneratedAt,omitempty"`
}

// DeepProposalFieldEdit is one caller-supplied edit for PATCH .../proposal.
type DeepProposalFieldEdit struct {
	OwnerValue    any
	OwnerValueSet bool
	Accepted      *bool
}

// DeepApplyResult is returned by Apply (mirrors the `apply` 200 response).
type DeepApplyResult struct {
	JobID         uint
	DraftID       *uint
	CoinID        *uint
	AppliedFields []string
	AppliedAt     time.Time
}

// DeepIdentificationProposalService owns the confirm-gated report->write
// bridge (US4, FR-031/FR-032/FR-033). It performs no coin/draft write of
// its own: every persisted change happens through CoinService or
// QuickCaptureService, the same two write paths every other part of the
// application uses (Principle IV).
type DeepIdentificationProposalService struct {
	repo     *repository.DeepIdentificationRepository
	coinRepo *repository.CoinRepository
	coinSvc  *CoinService
	qcSvc    *QuickCaptureService
}

func NewDeepIdentificationProposalService(
	repo *repository.DeepIdentificationRepository,
	coinRepo *repository.CoinRepository,
	coinSvc *CoinService,
	qcSvc *QuickCaptureService,
) *DeepIdentificationProposalService {
	return &DeepIdentificationProposalService{repo: repo, coinRepo: coinRepo, coinSvc: coinSvc, qcSvc: qcSvc}
}

func parseDeepProposalDocument(raw string) (*deepProposalDocument, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrDeepProposalNotReady
	}
	var doc deepProposalDocument
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, fmt.Errorf("parse proposal json: %w", err)
	}
	if doc.Fields == nil {
		doc.Fields = map[string]*deepProposalFieldEntry{}
	}
	return &doc, nil
}

// loadTerminalJobWithProposal loads an owner-scoped job and its parsed
// proposal, or a typed error distinguishing "not found" from "no proposal
// yet" from "already applied".
func (s *DeepIdentificationProposalService) loadTerminalJobWithProposal(jobID, userID uint) (*models.DeepIdentificationJob, *deepProposalDocument, error) {
	job, err := s.repo.GetJob(jobID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, nil, ErrDeepProposalNotFound
		}
		return nil, nil, err
	}
	if job.AppliedAt != nil {
		return job, nil, ErrDeepProposalAlreadyApplied
	}
	doc, err := parseDeepProposalDocument(job.ProposalJSON)
	if err != nil {
		return job, nil, err
	}
	return job, doc, nil
}

// UpdateProposal applies owner edits (ownerValue/accepted) to a proposal's
// fields (T110). It never touches coin/draft data - only the job's own
// ProposalJSON column - and only accepts field names already present in
// the AI-proposed document, which the synthesizer itself only ever
// populates from the allowlists above (defense-in-depth: a field name
// absent from the allowlist can never appear here either).
func (s *DeepIdentificationProposalService) UpdateProposal(jobID, userID uint, edits map[string]DeepProposalFieldEdit) (*deepProposalDocument, error) {
	job, doc, err := s.loadTerminalJobWithProposal(jobID, userID)
	if err != nil {
		return nil, err
	}
	for name := range edits {
		if _, known := doc.Fields[name]; !known {
			return nil, fmt.Errorf("%w: %q", ErrDeepProposalFieldNotAllowed, name)
		}
	}
	for name, edit := range edits {
		entry := doc.Fields[name]
		if edit.OwnerValueSet {
			entry.OwnerValue = edit.OwnerValue
			entry.OwnerEdited = true
		}
		if edit.Accepted != nil {
			entry.Accepted = edit.Accepted
		}
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.UpdateProposalJSON(job.ID, userID, string(encoded))
	if err != nil {
		return nil, err
	}
	if !ok {
		// Lost a race with a concurrent Apply between load and save.
		return nil, ErrDeepProposalAlreadyApplied
	}
	return doc, nil
}

// resolveFieldValue returns the effective value for a proposal field: the
// owner's edited value when present, else the AI-proposed value (FR-032).
func resolveDeepProposalFieldValue(entry *deepProposalFieldEntry) any {
	if entry.OwnerEdited {
		return entry.OwnerValue
	}
	return entry.Proposed
}

// selectAppliedFieldNames resolves which proposal field names to apply:
// the explicit fieldsFilter if given (every name must exist and be
// accepted), else every field marked accepted=true (contract: "Omit to
// apply every field marked accepted").
func selectDeepAppliedFieldNames(doc *deepProposalDocument, fieldsFilter []string) ([]string, error) {
	if len(fieldsFilter) > 0 {
		for _, name := range fieldsFilter {
			entry, ok := doc.Fields[name]
			if !ok {
				return nil, fmt.Errorf("%w: %q", ErrDeepProposalFieldNotAllowed, name)
			}
			if entry.Accepted == nil || !*entry.Accepted {
				return nil, fmt.Errorf("%w: %q is not accepted", ErrDeepProposalNoAcceptedFields, name)
			}
		}
		out := append([]string(nil), fieldsFilter...)
		sort.Strings(out)
		return out, nil
	}
	var out []string
	for name, entry := range doc.Fields {
		if entry.Accepted != nil && *entry.Accepted {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

// Apply confirms the proposal through an existing Go-owned write path
// (T111, FR-031/FR-033): "draft" seeds a new QuickCaptureDraft via
// QuickCaptureService (existing promote flow finishes the job); "coin"
// patches the saved coin via CoinService.UpdateCoinWithFields with journal
// source "deep_identification". No direct coin/draft write exists in this
// function or anywhere else in the deep-identification package.
func (s *DeepIdentificationProposalService) Apply(jobID, userID uint, target string, fieldsFilter []string) (*DeepApplyResult, error) {
	job, doc, err := s.loadTerminalJobWithProposal(jobID, userID)
	if err != nil {
		return nil, err
	}
	if job.Status != models.DeepJobStatusCompleted && job.Status != models.DeepJobStatusPartial {
		return nil, ErrDeepProposalNotReady
	}
	switch target {
	case "draft":
		if job.Source != models.DeepJobSourceIntake {
			return nil, ErrDeepProposalTargetMismatch
		}
	case "coin":
		if job.Source != models.DeepJobSourceSavedCoin {
			return nil, ErrDeepProposalTargetMismatch
		}
	default:
		return nil, fmt.Errorf("%w: unknown target %q", ErrDeepProposalFieldNotAllowed, target)
	}

	fieldNames, err := selectDeepAppliedFieldNames(doc, fieldsFilter)
	if err != nil {
		return nil, err
	}
	if len(fieldNames) == 0 {
		return nil, ErrDeepProposalNoAcceptedFields
	}

	var draftID, coinID *uint
	switch target {
	case "draft":
		id, err := s.applyToDraft(job, doc, fieldNames)
		if err != nil {
			return nil, err
		}
		draftID = &id
	case "coin":
		id, err := s.applyToCoin(job, userID, doc, fieldNames)
		if err != nil {
			return nil, err
		}
		coinID = &id
	}

	appliedAt := time.Now().UTC()
	won, err := s.repo.ApplyJob(job.ID, userID, coinID, draftID, appliedAt)
	if err != nil {
		return nil, err
	}
	if !won {
		return nil, ErrDeepProposalAlreadyApplied
	}
	return &DeepApplyResult{
		JobID:         job.ID,
		DraftID:       draftID,
		CoinID:        coinID,
		AppliedFields: fieldNames,
		AppliedAt:     appliedAt,
	}, nil
}

func (s *DeepIdentificationProposalService) applyToCoin(job *models.DeepIdentificationJob, userID uint, doc *deepProposalDocument, fieldNames []string) (uint, error) {
	if job.CoinID == nil {
		return 0, ErrDeepProposalSourceMissing
	}
	existing, err := s.coinRepo.FindByID(*job.CoinID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return 0, ErrDeepProposalSourceMissing
		}
		return 0, err
	}
	updates := &models.Coin{}
	goFields := make([]string, 0, len(fieldNames))
	for _, name := range fieldNames {
		goField, ok := deepProposalCoinFieldAllowlist[name]
		if !ok {
			return 0, fmt.Errorf("%w: %q", ErrDeepProposalFieldNotAllowed, name)
		}
		value := resolveDeepProposalFieldValue(doc.Fields[name])
		if err := setCoinFieldFromProposalValue(updates, goField, value); err != nil {
			return 0, err
		}
		goFields = append(goFields, goField)
	}
	if err := s.coinSvc.UpdateCoinWithFields(existing, updates, goFields, userID, "deep_identification", false); err != nil {
		return 0, err
	}
	return existing.ID, nil
}

func (s *DeepIdentificationProposalService) applyToDraft(job *models.DeepIdentificationJob, doc *deepProposalDocument, fieldNames []string) (uint, error) {
	input := CreateQuickCaptureDraftInput{
		UserID: job.UserID,
		Source: "deep_identification",
	}
	for _, name := range fieldNames {
		draftField, ok := deepProposalDraftFieldAllowlist[name]
		if !ok {
			return 0, fmt.Errorf("%w: %q", ErrDeepProposalFieldNotAllowed, name)
		}
		value := deepProposalValueToString(resolveDeepProposalFieldValue(doc.Fields[name]))
		switch draftField {
		case "WorkingTitle":
			input.WorkingTitle = value
		case "Era":
			input.Era = value
		case "DateRange":
			input.DateRange = value
		case "Notes":
			input.Notes = value
		}
	}
	images, err := s.deepJobFaceImages(job)
	if err != nil {
		return 0, err
	}
	input.Images = images

	draft, err := s.qcSvc.CreateDraft(input)
	if err != nil {
		return 0, err
	}
	return draft.ID, nil
}

// deepJobFaceImages loads the job's non-hint (obverse/reverse) artifact
// bytes so the seeded draft carries the same coin-face images the job was
// analyzed from - hint images are never promoted (FR-021/SC-004) and are
// excluded here by construction (only Role=obverse/reverse are read).
func (s *DeepIdentificationProposalService) deepJobFaceImages(job *models.DeepIdentificationJob) ([]QuickCaptureImageUpload, error) {
	artifacts, err := s.repo.ListArtifacts(job.ID)
	if err != nil {
		return nil, err
	}
	var images []QuickCaptureImageUpload
	for _, artifact := range artifacts {
		if artifact.Role == models.DeepArtifactRoleHint || artifact.DeletedAt != nil || artifact.FilePath == "" {
			continue
		}
		data, err := os.ReadFile(artifact.FilePath)
		if err != nil {
			return nil, err
		}
		imageType := "obverse"
		if artifact.Role == models.DeepArtifactRoleReverse {
			imageType = "reverse"
		}
		images = append(images, QuickCaptureImageUpload{
			Filename:  string(artifact.Role) + ".jpg",
			Data:      data,
			ImageType: imageType,
			IsPrimary: artifact.Role == models.DeepArtifactRoleObverse,
		})
	}
	return images, nil
}

// setCoinFieldFromProposalValue assigns value onto the named models.Coin
// field, coercing JSON-decoded types (float64/string) into the field's Go
// type. Only fields in deepProposalCoinFieldAllowlist ever reach here.
func setCoinFieldFromProposalValue(coin *models.Coin, field string, value any) error {
	switch field {
	case "Denomination":
		coin.Denomination = deepProposalValueToString(value)
	case "Ruler":
		coin.Ruler = deepProposalValueToString(value)
	case "Era":
		coin.Era = models.Era(deepProposalValueToString(value))
	case "DateRange":
		coin.DateRange = deepProposalValueToString(value)
	case "Mint":
		coin.Mint = deepProposalValueToString(value)
	case "Material":
		coin.Material = models.Material(deepProposalValueToString(value))
	case "WeightGrams":
		f, err := deepProposalValueToFloat(value)
		if err != nil {
			return err
		}
		coin.WeightGrams = f
	case "DiameterMm":
		f, err := deepProposalValueToFloat(value)
		if err != nil {
			return err
		}
		coin.DiameterMm = f
	case "ObverseInscription":
		coin.ObverseInscription = deepProposalValueToString(value)
	case "ReverseInscription":
		coin.ReverseInscription = deepProposalValueToString(value)
	case "ObverseDescription":
		coin.ObverseDescription = deepProposalValueToString(value)
	case "ReverseDescription":
		coin.ReverseDescription = deepProposalValueToString(value)
	case "Notes":
		coin.Notes = deepProposalValueToString(value)
	default:
		return fmt.Errorf("%w: %q", ErrDeepProposalFieldNotAllowed, field)
	}
	return nil
}

func deepProposalValueToString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

func deepProposalValueToFloat(value any) (*float64, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case float64:
		return &v, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil, nil
		}
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric value %q", v)
		}
		return &f, nil
	default:
		return nil, fmt.Errorf("invalid numeric value %v", v)
	}
}
