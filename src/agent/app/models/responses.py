"""Response models returned to the Go API proxy."""

from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints


class StrictResponseModel(BaseModel):
    """Base response model with contract drift detection."""

    model_config = ConfigDict(extra="forbid")


MAX_SET_BUILDER_SLOTS_RESPONSE = 300


class CandidateReference(BaseModel):
    """A potential structured catalog reference extracted from listing text."""

    catalog: str
    volume: str = ""
    number: str
    uri: str = ""


class CoinSuggestion(BaseModel):
    """A verified coin listing found by the search pipeline."""

    name: str
    description: str = ""
    category: str = ""
    era: str = ""
    ruler: str = ""
    material: str = ""
    denomination: str = ""
    est_price: str = ""
    image_url: str = ""
    source_url: str  # Required — must be a verified live URL
    source_name: str = ""
    candidate_references: list[CandidateReference] = Field(
        default_factory=list,
        serialization_alias="candidateReferences",
    )


class CoinShow(BaseModel):
    """A verified upcoming coin show."""

    name: str
    dates: str = ""
    location: str = ""
    venue: str = ""
    url: str = ""
    description: str = ""
    entry_fee: str = ""
    notable_dealers: list[str] = []


class ValueEstimate(BaseModel):
    """AI-generated value estimate for a coin."""

    estimated_value: float = 0
    confidence: str = "low"  # "low", "medium", "high"
    reasoning: str = ""
    comparables: list[dict] = []


class AgentResponse(BaseModel):
    """Unified response from any agent team."""

    message: str = ""
    suggestions: list[CoinSuggestion] = []
    shows: list[CoinShow] = []
    estimate: ValueEstimate | None = None
    analysis: str = ""


class GradeResponse(BaseModel):
    """Coin grading report returned to the Go API proxy."""

    report: str = ""


class AvailabilityVerdict(BaseModel):
    """AI-determined availability verdict for a single URL."""

    url: Annotated[str, StringConstraints(min_length=1, max_length=2048)]
    coin_name: Annotated[str, StringConstraints(max_length=300)] = ""
    status: Literal["available", "unavailable", "unknown"]
    reason: Annotated[str, StringConstraints(max_length=1000)] = ""
    confidence: Literal["low", "medium", "high"] = "medium"


class AvailabilityCheckResponse(BaseModel):
    """Response from the availability check endpoint."""

    results: list[AvailabilityVerdict] = []


class MarketSignalResponse(StrictResponseModel):
    """Structured price-trend signal for a specific tracked auction lot, derived
    from a live auction-results web search. Always HTTP 200 — `degraded` signals
    the caller should fall back to historical-only data, never an exception.
    """

    trend_direction: Literal["rising", "stable", "declining", "unknown"] = "unknown"
    price_low: float | None = Field(default=None, ge=0)
    price_high: float | None = Field(default=None, ge=0)
    currency: Annotated[str, StringConstraints(max_length=3)] = "USD"
    sample_size: int = Field(default=0, ge=0)
    rationale: Annotated[str, StringConstraints(max_length=1000)] = ""
    sources: list[Annotated[str, StringConstraints(max_length=2048)]] = Field(default_factory=list, max_length=5)
    degraded: bool = False


# Wishlist search alert discovery DTOs.
# Contract anchor: specs/337-wishlist-search-alerts/contracts/agent-discovery-contract.md
class AlertDiscoveryProvenance(StrictResponseModel):
    field: Annotated[str, StringConstraints(min_length=1, max_length=100)]
    value: Annotated[str, StringConstraints(min_length=1, max_length=4000)]
    source_url: Annotated[str, StringConstraints(min_length=1, max_length=2048)]
    observed_at: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    confidence: Literal["high", "medium", "low"]
    verification_state: Literal["verified", "partial", "unverified"]
    notes: Annotated[str, StringConstraints(max_length=1000)] = ""


class AlertDiscoveryCandidate(StrictResponseModel):
    source_url: Annotated[str, StringConstraints(min_length=1, max_length=2048)]
    source_name: Annotated[str, StringConstraints(max_length=500)] = ""
    title: Annotated[str, StringConstraints(min_length=1, max_length=500)]
    observed_price: float | None = Field(default=None, ge=0)
    observed_currency: Annotated[str, StringConstraints(max_length=3)] = ""
    reason_for_match: Annotated[str, StringConstraints(min_length=1, max_length=4000)]
    last_seen_at: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    provenance_status: Literal["verified", "partial", "unverified"]
    fields: dict[str, str] = Field(default_factory=dict, max_length=50)
    provenance: list[AlertDiscoveryProvenance] = Field(default_factory=list, min_length=1)


class AlertDiscoveryResponse(StrictResponseModel):
    candidates: list[AlertDiscoveryCandidate] = Field(default_factory=list)
    warnings: list[str] = Field(default_factory=list)
    partial: bool = False


class IntakeConfidenceSummary(BaseModel):
    """Confidence rollup for the generated intake draft."""

    overall: Literal["low", "medium", "high"] = "low"
    uncertain_fields: list[str] = Field(
        default_factory=list,
        validation_alias="uncertainFields",
        serialization_alias="uncertainFields",
    )


class IntakeEvidenceItem(BaseModel):
    """Evidence item mapping extracted signal to an output field."""

    type: str = ""
    source: str = ""
    field: str = ""
    value: str = ""
    confidence: Literal["low", "medium", "high"] = "low"
    notes: str = ""


class IntakeDraftResponse(BaseModel):
    """Structured draft output for the intake flow."""

    coin: dict = Field(default_factory=dict)
    confidence_summary: IntakeConfidenceSummary = Field(
        default_factory=IntakeConfidenceSummary,
        validation_alias="confidenceSummary",
        serialization_alias="confidenceSummary",
    )
    evidence: list[IntakeEvidenceItem] = Field(default_factory=list)
    unresolved_fields: list[str] = Field(
        default_factory=list,
        validation_alias="unresolvedFields",
        serialization_alias="unresolvedFields",
    )


# Dynamic Set Builder workflow DTOs.
# Contract anchor: specs/011-dynamic-set-builder-correction-plan.md (Phase 2)
class SetBuilderScopeOption(StrictResponseModel):
    """One candidate scope interpretation offered by the Intent Analyst role."""

    label: Annotated[str, StringConstraints(min_length=1, max_length=200)]
    description: Annotated[str, StringConstraints(max_length=1000)] = ""
    estimated_slot_count: int = Field(default=0, ge=0)
    recommended: bool = False


class SetBuilderSlot(StrictResponseModel):
    """One proposed roster entry. Becomes a TrackerSlot only after Go approval."""

    label: Annotated[str, StringConstraints(min_length=1, max_length=300)]
    criteria: dict[str, str] = Field(default_factory=dict, max_length=50)
    group: Annotated[str, StringConstraints(max_length=200)] = ""
    sort_order: int = 0
    verification_status: Literal["verified", "unverified"] = "unverified"
    source_note: Annotated[str, StringConstraints(max_length=1000)] = ""
    validation_notes: Annotated[str, StringConstraints(max_length=1000)] = ""


class SetBuilderPrematchSummary(StrictResponseModel):
    """Estimated filled/total preview from the Collection Matcher role."""

    estimated_filled: int = Field(default=0, ge=0)
    estimated_total: int = Field(default=0, ge=0)
    notes: Annotated[str, StringConstraints(max_length=1000)] = ""


class SetBuilderProposal(StrictResponseModel):
    """Structured Set Proposal data only — never a created set. FR-003."""

    name: Annotated[str, StringConstraints(min_length=1, max_length=300)]
    slug_hint: Annotated[str, StringConstraints(max_length=300)] = ""
    description: Annotated[str, StringConstraints(max_length=2000)] = ""
    scope_summary: Annotated[str, StringConstraints(max_length=2000)] = ""
    selected_scope: Annotated[str, StringConstraints(max_length=200)] = ""
    group_by: Annotated[str, StringConstraints(max_length=200)] = ""
    scope_options: list[SetBuilderScopeOption] = Field(default_factory=list, max_length=10)
    slots: list[SetBuilderSlot] = Field(default_factory=list, max_length=MAX_SET_BUILDER_SLOTS_RESPONSE)
    prematch_summary: SetBuilderPrematchSummary = Field(default_factory=SetBuilderPrematchSummary)


class SetBuilderResponse(StrictResponseModel):
    """Response from the set-builder workflow. Data only — no side effects.

    `status` mirrors the outcomes required by spec 011 US1: a completed
    proposal, a clarification request for ambiguous/unbounded prompts, or a
    structured failure (including execution-limit termination) instead of a
    fabricated roster.
    """

    status: Literal["completed", "clarification_needed", "rejected", "failed", "limit_reached"]
    proposal: SetBuilderProposal | None = None
    clarification_question: Annotated[str, StringConstraints(max_length=1000)] = ""
    failure_reason: Annotated[str, StringConstraints(max_length=1000)] = ""
    transcript_summary: Annotated[str, StringConstraints(max_length=4000)] = ""
    turns_used: int = Field(default=0, ge=0)


# Deep Agentic Coin Identification DTOs (344-deep-agentic-coin-identification).
# Contract anchor: specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md §4-5
ProviderName = Literal["numista", "nomisma", "ngc", "ocre", "rpc"]
ProviderStatus = Literal[
    "contributed", "no_match", "failed", "timed_out", "not_automated", "unavailable", "skipped"
]
ProviderErrorKind = Literal["timeout", "quota", "unconfigured", "upstream", "invalid_response"]


class ProviderClaim(StrictResponseModel):
    """A single typed, citation-backed factual claim from one provider.

    Every claim MUST carry a citation whose host belongs to the emitting
    provider's canonical allowlist (§4, SC-006) — claims failing that check
    are dropped by `merge.validate_citations` before this model is even
    constructed, so an instance of this class is always pre-validated.
    """

    field: Annotated[str, StringConstraints(min_length=1, max_length=100)]
    value: Annotated[str, StringConstraints(min_length=1, max_length=1000)]
    confidence: float = Field(ge=0.0, le=1.0)
    citation: Annotated[str, StringConstraints(min_length=1, max_length=2048)]
    excerpt: Annotated[str, StringConstraints(max_length=500)] = ""


class ProviderEvidence(StrictResponseModel):
    """Typed, never-prose evidence row for a single provider (§4)."""

    provider: ProviderName
    status: ProviderStatus
    automatable: bool
    confidence: float = Field(default=0.0, ge=0.0, le=1.0)
    call_count: int = Field(default=0, ge=0)
    error_kind: ProviderErrorKind | None = None
    link_out: Annotated[str, StringConstraints(max_length=2048)] = ""
    attribution: Annotated[str, StringConstraints(max_length=200)] = ""
    claims: list[ProviderClaim] = Field(default_factory=list, max_length=50)


class EvidenceRef(StrictResponseModel):
    """A reference from a proposed field or disagreement back to one
    provider's evidence (or `provider: "image"` for image-only support).
    """

    provider: Annotated[str, StringConstraints(min_length=1, max_length=20)]
    claim_index: int | None = Field(default=None, ge=0)


class ProposedFieldValue(StrictResponseModel):
    """One proposed coin-field value with its supporting evidence."""

    value: Annotated[str, StringConstraints(min_length=1, max_length=1000)]
    confidence: float = Field(ge=0.0, le=1.0)
    evidence_refs: list[EvidenceRef] = Field(default_factory=list, min_length=1, max_length=20)


class DisagreementEntry(StrictResponseModel):
    """A field where two or more providers disagree — surfaced, never
    silently resolved by precedence (FR-027).
    """

    field: Annotated[str, StringConstraints(min_length=1, max_length=100)]
    claim_refs: list[EvidenceRef] = Field(default_factory=list, min_length=1, max_length=20)
    resolution: Literal["unresolved", "resolved"] = "unresolved"


class ProviderCoverageEntry(StrictResponseModel):
    """One provider's final status, for the run's coverage summary."""

    provider: ProviderName
    status: ProviderStatus


class ProviderAttribution(StrictResponseModel):
    """Visible attribution/license metadata for one provider that actually
    contributed to the report (§6 / FR-019). Present only when that provider
    surfaced ≥1 claim; each provider's text is distinct and never merged.
    """

    provider: ProviderName
    text: Annotated[str, StringConstraints(max_length=200)]
    identifier: str | None = None


class DeepSynthesis(StrictResponseModel):
    """Typed final synthesis output (§5) — the terminal-success SSE frame
    payload. `proposed_fields` keys are re-validated against the coin-field
    allowlist Go-side on ingest; unknown keys are dropped there, not here.
    """

    narrative: Annotated[str, StringConstraints(max_length=8000)] = ""
    proposed_fields: dict[str, ProposedFieldValue] = Field(default_factory=dict, max_length=50)
    disagreements: list[DisagreementEntry] = Field(default_factory=list, max_length=50)
    unresolved_questions: list[Annotated[str, StringConstraints(max_length=500)]] = Field(
        default_factory=list, max_length=20
    )
    coverage: list[ProviderCoverageEntry] = Field(default_factory=list, max_length=10)
    attributions: list[ProviderAttribution] = Field(default_factory=list, max_length=10)
    partial_success: bool = False
