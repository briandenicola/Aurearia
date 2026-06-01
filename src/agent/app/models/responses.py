"""Response models returned to the Go API proxy."""

from typing import Annotated, Literal

from pydantic import BaseModel, Field, StringConstraints


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
