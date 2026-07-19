"""Response models returned to the Go API proxy."""

from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints


class StrictResponseModel(BaseModel):
    """Base response model with contract drift detection."""

    model_config = ConfigDict(extra="forbid")


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
