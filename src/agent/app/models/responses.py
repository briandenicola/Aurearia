"""Response models returned to the Go API proxy."""

import ipaddress
import json
from typing import Annotated, Any, Literal
from urllib.parse import urlsplit

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, field_validator, model_validator

from app.models.hypothesis import CoinHypothesis
from app.models.requests import (
    COPILOT_ALLOWED_TOOLS,
    COPILOT_SPECIALIST_TOOLS,
    MAX_COPILOT_CLARIFICATION_CHOICES,
    MAX_COPILOT_CLARIFICATION_LENGTH,
    MAX_COPILOT_PLAN_ITEMS,
    ChatMessage,
    CopilotBoundedToolResult,
    CopilotClarification,
    CopilotCompletedTool,
    CopilotExecutionID,
    CopilotPlanItem,
    CopilotRunID,
    CopilotToolCallID,
    CopilotUsage,
)
from app.teams.specialist_contracts import SpecialistResult


class StrictResponseModel(BaseModel):
    """Base response model with contract drift detection."""

    model_config = ConfigDict(extra="forbid")


MAX_DEEP_ANALYSIS_HANDOFF_PUBLIC_EVENT_BYTES = 65_536
MAX_DEEP_ANALYSIS_HANDOFF_PERSISTED_RESULT_BYTES = 32_768

DeepAnalysisHandoffOutcome = Literal[
    "accepted",
    "reused_active",
    "reused_result",
    "status",
    "retry_available",
    "missing_images",
    "target_unavailable",
    "not_eligible",
    "unavailable",
    "cancelled",
]
DeepAnalysisHandoffReason = Literal[
    "missing_obverse",
    "missing_reverse",
    "missing_both",
    "duplicate_faces",
    "target_changed",
    "draft_inactive",
    "source_coin_missing",
    "deep_disabled",
    "copilot_disabled",
    "attribution_disabled",
    "model_unsupported",
    "job_at_capacity",
    "queue_full",
    "result_missing",
    "result_expired",
    "stale",
    "cancelled",
]


class DeepAnalysisHandoffTargetResult(StrictResponseModel):
    type: Literal["coin", "draft"]
    id: int = Field(gt=0)
    display_label: Annotated[str, StringConstraints(min_length=1, max_length=300)]


class DeepAnalysisHandoffJob(StrictResponseModel):
    id: int = Field(gt=0)
    source: Literal["intake", "saved_coin", "copilot_draft"]
    status: Literal["queued", "running", "completed", "partial", "failed", "cancelled"]
    reused: bool
    created_at: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    completed_at: Annotated[str, StringConstraints(min_length=1, max_length=64)] | None


class DeepAnalysisHandoffEvidence(StrictResponseModel):
    provider: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    source: Annotated[str, StringConstraints(min_length=1, max_length=200)]
    url: Annotated[str, StringConstraints(min_length=1, max_length=2048)]
    summary: Annotated[str, StringConstraints(min_length=1, max_length=1000)]

    @field_validator("url")
    @classmethod
    def validate_safe_url(cls, value: str) -> str:
        parsed = urlsplit(value)
        if parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password:
            raise ValueError("evidence URL must be credential-free HTTPS")
        host = parsed.hostname.rstrip(".").lower()
        if host == "localhost" or host.endswith((".localhost", ".local")):
            raise ValueError("evidence URL host is not public")
        try:
            address = ipaddress.ip_address(host)
        except ValueError:
            address = None
        if address is not None and not address.is_global:
            raise ValueError("evidence URL host is not public")
        return value


class DeepAnalysisHandoffField(StrictResponseModel):
    name: Annotated[str, StringConstraints(min_length=1, max_length=100)]
    value: Annotated[str, StringConstraints(min_length=1, max_length=2000)]
    confidence: float = Field(ge=0, le=1, allow_inf_nan=False)
    evidence: list[DeepAnalysisHandoffEvidence] = Field(default_factory=list)


class DeepAnalysisHandoffDisagreement(StrictResponseModel):
    field: Annotated[str, StringConstraints(min_length=1, max_length=100)]
    summary: Annotated[str, StringConstraints(min_length=1, max_length=1000)]


class DeepAnalysisHandoffCoverage(StrictResponseModel):
    provider: Literal["numista", "nomisma", "ngc", "ocre", "rpc"]
    status: Literal[
        "pending",
        "running",
        "contributed",
        "no_match",
        "failed",
        "timed_out",
        "skipped",
        "not_automated",
        "unavailable",
    ]


class DeepAnalysisHandoffAttribution(StrictResponseModel):
    provider: Literal["numista", "nomisma", "ngc", "ocre", "rpc"]
    label: Annotated[str, StringConstraints(min_length=1, max_length=200)]


class DeepAnalysisHandoffResultBody(StrictResponseModel):
    state: Literal[
        "not_ready",
        "complete",
        "partial",
        "no_match",
        "failed",
        "cancelled",
        "stale",
        "missing_result",
    ]
    narrative: Annotated[str, StringConstraints(max_length=12000)]
    partial_success: bool
    image_only: bool
    fields: list[DeepAnalysisHandoffField] = Field(default_factory=list)
    disagreements: list[DeepAnalysisHandoffDisagreement] = Field(default_factory=list)
    unresolved_questions: list[Annotated[str, StringConstraints(max_length=1000)]] = Field(default_factory=list)
    coverage: list[DeepAnalysisHandoffCoverage] = Field(default_factory=list)
    attributions: list[DeepAnalysisHandoffAttribution] = Field(default_factory=list)
    limitations: list[Annotated[str, StringConstraints(max_length=1000)]] = Field(default_factory=list)

    @model_validator(mode="after")
    def reject_duplicate_entries(self) -> "DeepAnalysisHandoffResultBody":
        for values in (
            [field.name for field in self.fields],
            [coverage.provider for coverage in self.coverage],
            [attribution.provider for attribution in self.attributions],
        ):
            if len(values) != len(set(values)):
                raise ValueError("duplicate handoff result entry")
        return self


class DeepAnalysisHandoffTruncation(StrictResponseModel):
    truncated: bool
    original_bytes: int = Field(ge=0)
    persisted_bytes: int = Field(ge=0, le=MAX_DEEP_ANALYSIS_HANDOFF_PERSISTED_RESULT_BYTES)
    digest: Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")]
    omitted_fields: int = Field(ge=0)
    omitted_evidence: int = Field(ge=0)
    omitted_disagreements: int = Field(ge=0)
    omitted_questions: int = Field(ge=0)


class DeepAnalysisHandoffResult(StrictResponseModel):
    """Strict result; ``outcome`` is the only top-level discriminant."""

    schema_version: Literal[1] | None = None
    operation: Literal["request", "status", "rerun"] | None = None
    outcome: DeepAnalysisHandoffOutcome
    reason: DeepAnalysisHandoffReason | None
    target: DeepAnalysisHandoffTargetResult | None = None
    job: DeepAnalysisHandoffJob | None = None
    input_digest: Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")] | None = None
    review_url: Annotated[str, StringConstraints(pattern=r"^/deep-analysis/[1-9][0-9]*$")] | None = None
    fresh_analysis_available: bool = False
    result: DeepAnalysisHandoffResultBody | None = None
    truncation: DeepAnalysisHandoffTruncation | None = None
    limitations: list[Annotated[str, StringConstraints(max_length=1000)]] = Field(default_factory=list)

    @model_validator(mode="after")
    def validate_outcome_shape(self) -> "DeepAnalysisHandoffResult":
        privacy_outcome = self.outcome in {"not_eligible", "target_unavailable"}
        if privacy_outcome:
            if self.reason is not None or any(
                value is not None
                for value in (
                    self.schema_version,
                    self.operation,
                    self.target,
                    self.job,
                    self.input_digest,
                    self.review_url,
                    self.result,
                    self.truncation,
                )
            ) or self.fresh_analysis_available or self.limitations:
                raise ValueError("privacy-safe outcomes must contain only outcome and null reason")
            return self
        if self.schema_version != 1 or self.operation is None:
            raise ValueError("typed handoff result requires schema_version and operation")
        if self.job is not None and self.review_url != f"/deep-analysis/{self.job.id}":
            raise ValueError("review_url must match job id")
        if self.job is None and self.review_url is not None:
            raise ValueError("review_url requires job")
        return self


def _validate_deep_analysis_handoff_envelope(payload: bytes, maximum: int) -> object:
    if not payload:
        raise ValueError("handoff envelope is empty")
    try:
        value = json.loads(payload)
        canonical = json.dumps(
            value,
            ensure_ascii=False,
            allow_nan=False,
            separators=(",", ":"),
            sort_keys=True,
        ).encode("utf-8")
    except (UnicodeDecodeError, ValueError, TypeError) as exc:
        raise ValueError("invalid handoff JSON") from exc
    if len(canonical) > maximum:
        raise ValueError("handoff envelope exceeds its byte limit")
    return value


def validate_deep_analysis_handoff_public_event_envelope(payload: bytes) -> object:
    return _validate_deep_analysis_handoff_envelope(
        payload,
        MAX_DEEP_ANALYSIS_HANDOFF_PUBLIC_EVENT_BYTES,
    )


def validate_deep_analysis_handoff_persisted_result_envelope(payload: bytes) -> object:
    return _validate_deep_analysis_handoff_envelope(
        payload,
        MAX_DEEP_ANALYSIS_HANDOFF_PERSISTED_RESULT_BYTES,
    )


MAX_SET_BUILDER_SLOTS_RESPONSE = 300
MAX_WISHLIST_FEATURED_SUMMARY_LENGTH = 500
MAX_COPILOT_ANSWER_LENGTH = 100000
MAX_COPILOT_RESULT_SUMMARY_LENGTH = 300


class CopilotCapabilityResponse(StrictResponseModel):
    supported: bool


class CopilotCheckpointState(StrictResponseModel):
    """Complete durable continuation state emitted to Go."""

    schema_version: Literal[1] = 1
    messages: list[ChatMessage] = Field(default_factory=list, max_length=50)
    plan: list[CopilotPlanItem] = Field(default_factory=list, max_length=MAX_COPILOT_PLAN_ITEMS)
    completed_tools: list[CopilotCompletedTool] = Field(default_factory=list, max_length=40)
    pending_clarification: CopilotClarification | None = None
    next_action: Literal["continue", "await_clarification", "finish"] = "continue"
    counters: CopilotUsage

    @model_validator(mode="after")
    def reject_private_or_duplicate_state(self) -> "CopilotCheckpointState":
        seen: set[str] = set()
        if sum(len(message.content) for message in self.messages) > 100000:
            raise ValueError("checkpoint messages exceed the total content limit")
        for tool in self.completed_tools:
            call_id = tool.tool_call_id
            if call_id in seen:
                raise ValueError("completed_tools contains duplicate tool_call_id values")
            seen.add(call_id)
        return self


class CopilotPlanUpdatedPayload(StrictResponseModel):
    plan: list[CopilotPlanItem] = Field(max_length=MAX_COPILOT_PLAN_ITEMS)


class CopilotToolStartedPayload(StrictResponseModel):
    tool_call_id: CopilotToolCallID
    tool_name: str
    step_id: Annotated[str, StringConstraints(min_length=1, max_length=64)]

    @model_validator(mode="after")
    def validate_tool_name(self) -> "CopilotToolStartedPayload":
        if self.tool_name not in COPILOT_ALLOWED_TOOLS:
            raise ValueError("tool_name is not in the Coin Copilot allowlist")
        return self


class CopilotToolCompletedPayload(CopilotToolStartedPayload):
    status: Literal["succeeded", "failed", "cancelled", "rejected"]
    duration_ms: int = Field(ge=0)
    result_summary: Annotated[str, StringConstraints(max_length=MAX_COPILOT_RESULT_SUMMARY_LENGTH)]
    result: SpecialistResult | dict[str, Any]

    @model_validator(mode="after")
    def validate_specialist_result(self) -> "CopilotToolCompletedPayload":
        if self.tool_name == "deep_analysis_handoff":
            try:
                self.result = DeepAnalysisHandoffResult.model_validate(self.result)
            except ValueError as exc:
                try:
                    fallback = CopilotBoundedToolResult.model_validate(self.result)
                except ValueError:
                    raise ValueError("Deep Analysis handoff result is invalid") from exc
                self.result = fallback.model_dump(mode="json")
            return self
        if self.tool_name in COPILOT_SPECIALIST_TOOLS:
            try:
                result = SpecialistResult.model_validate(self.result)
            except ValueError as exc:
                try:
                    fallback = CopilotBoundedToolResult.model_validate(self.result)
                except ValueError:
                    raise ValueError("specialist result is invalid") from exc
                self.result = fallback.model_dump(mode="json")
                return self
            if result.capability != self.tool_name:
                raise ValueError("specialist result capability does not match tool_name")
            self.result = result
        elif isinstance(self.result, SpecialistResult):
            raise ValueError("specialist result requires a specialist tool_name")
        return self


class CopilotClarificationPayload(StrictResponseModel):
    question: Annotated[str, StringConstraints(min_length=1, max_length=MAX_COPILOT_CLARIFICATION_LENGTH)]
    input_type: Literal["text", "single_choice", "boolean"]
    choices: list[Annotated[str, StringConstraints(min_length=1, max_length=200)]] = Field(
        default_factory=list,
        max_length=MAX_COPILOT_CLARIFICATION_CHOICES,
    )


class CopilotCompletedPayload(StrictResponseModel):
    answer: Annotated[str, StringConstraints(min_length=1, max_length=MAX_COPILOT_ANSWER_LENGTH)]
    usage: CopilotUsage


class CopilotFailedPayload(StrictResponseModel):
    code: Literal[
        "agent_unavailable",
        "execution_lost",
        "invalid_agent_frame",
        "invalid_tool_call",
        "iteration_limit_exceeded",
        "tool_limit_exceeded",
        "time_limit_exceeded",
        "model_tool_calling_unsupported",
        "resume_window_expired",
        "internal",
    ]
    message: Annotated[str, StringConstraints(min_length=1, max_length=300)]
    retryable: bool
    usage: CopilotUsage


CopilotFramePayload = (
    CopilotPlanUpdatedPayload
    | CopilotToolStartedPayload
    | CopilotToolCompletedPayload
    | CopilotCheckpointState
    | CopilotClarificationPayload
    | CopilotCompletedPayload
    | CopilotFailedPayload
    | CopilotUsage
)


class CopilotExecutionFrame(StrictResponseModel):
    """Typed internal SSE frame consumed and persisted by Go."""

    schema_version: Literal[1] = 1
    run_id: CopilotRunID
    execution_id: CopilotExecutionID
    frame_id: Annotated[str, StringConstraints(min_length=1, max_length=100)]
    type: Literal[
        "plan_updated",
        "tool_started",
        "tool_completed",
        "checkpoint",
        "clarification_required",
        "completed",
        "failed",
        "usage",
    ]
    payload: CopilotFramePayload

    @model_validator(mode="after")
    def validate_payload_type(self) -> "CopilotExecutionFrame":
        expected = {
            "plan_updated": CopilotPlanUpdatedPayload,
            "tool_started": CopilotToolStartedPayload,
            "tool_completed": CopilotToolCompletedPayload,
            "checkpoint": CopilotCheckpointState,
            "clarification_required": CopilotClarificationPayload,
            "completed": CopilotCompletedPayload,
            "failed": CopilotFailedPayload,
            "usage": CopilotUsage,
        }[self.type]
        if not isinstance(self.payload, expected):
            raise ValueError(f"payload does not match frame type {self.type}")
        return self


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


class WishlistFeaturedSummaryResponse(StrictResponseModel):
    """Concise wishlist featured-coin rationale."""

    summary: Annotated[str, StringConstraints(min_length=1, max_length=MAX_WISHLIST_FEATURED_SUMMARY_LENGTH)]


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
ProviderStatus = Literal["contributed", "no_match", "failed", "timed_out", "not_automated", "unavailable", "skipped"]
ProviderErrorKind = Literal[
    "timeout", "quota", "unconfigured", "upstream", "invalid_response", "insufficient_query_evidence"
]


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
    # Additive, optional (contracts/vision-hypothesis.md §4 / spec FR-008):
    # present when the vision call produced anything, so the raw hypothesis
    # is recoverable from the persisted report even where `proposed_fields`
    # only carries the fields that survived corroboration/disagreement
    # filtering. Absent in reports persisted before this feature; Go's
    # report reader unmarshals only `narrative`/`proposed_fields`, so this
    # key is ignored by existing code (additive-safe).
    image_hypothesis: CoinHypothesis | None = None
    partial_success: bool = False
