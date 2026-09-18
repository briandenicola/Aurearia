"""Strict normalized contracts for Coin Copilot specialist market tools."""

from __future__ import annotations

import ipaddress
import re
from datetime import date, datetime, timedelta, timezone
from decimal import Decimal
from statistics import median
from typing import Annotated, Literal
from urllib.parse import urlsplit

from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    StringConstraints,
    field_validator,
    model_validator,
)

SpecialistCapability = Literal[
    "market_search",
    "auction_search",
    "price_trends",
    "similar_lots",
]
SpecialistOutcome = Literal["complete", "partial", "no_match", "unavailable"]
ProviderStatus = Literal[
    "success",
    "no_match",
    "timeout",
    "failure",
    "unavailable",
    "malformed",
]
ProviderWarningCode = Literal[
    "provider_timeout",
    "provider_unavailable",
    "provider_failure",
    "provider_malformed",
]
Confidence = Literal["high", "medium", "low"]
VerificationState = Literal["verified", "partial"]

BoundedProvider = Annotated[str, StringConstraints(min_length=1, max_length=64)]
BoundedTitle = Annotated[str, StringConstraints(min_length=1, max_length=300)]
BoundedDescription = Annotated[str, StringConstraints(min_length=1, max_length=500)]
BoundedText = Annotated[str, StringConstraints(min_length=1, max_length=300)]
BoundedWarning = Annotated[str, StringConstraints(min_length=1, max_length=500)]
BoundedSourceURL = Annotated[str, StringConstraints(min_length=1, max_length=2048)]
Currency = Annotated[str, StringConstraints(pattern=r"^[A-Z]{3}$")]
SHA256Digest = Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")]

_DEGRADED_PROVIDER_STATUSES = {"timeout", "failure", "unavailable", "malformed"}
_NON_PUBLIC_HOSTS = {"localhost", "metadata.google.internal"}
_REGISTERED_SOURCE_HOSTS = {
    "cng_dealer_search": frozenset({"cngcoins.com"}),
    "numisbids": frozenset({"numisbids.com"}),
}
_CAPABILITY_PROVIDERS = {
    "market_search": frozenset({"cng_dealer_search", "market_search_secondary"}),
    "auction_search": frozenset({"numisbids", "auction_search_secondary"}),
    "price_trends": frozenset({"numisbids", "price_trends_secondary"}),
    "similar_lots": frozenset({"numisbids", "similar_lots_secondary"}),
}
_TOKEN_RE = re.compile(
    r"(?i)(?:bearer\s+[A-Za-z0-9._~+/\-=]{12,}|"
    r"(?:api[_-]?key|secret|password)\s*[:=]\s*[\"']?[A-Za-z0-9._~+/\-=]{8,}|"
    r"token\s*[:=]\s*[\"']?[A-Za-z0-9._~+/\-=]{16,})"
)
_INSTRUCTION_RE = re.compile(
    r"(?ix)\b(?:"
    r"ignore\s+(?:all\s+)?(?:previous|prior|above)\s+(?:instructions?|rules?|prompts?)"
    r"|ignore\s+(?:the\s+)?(?:system|developer|hidden)\s+(?:instructions?|message|prompts?)"
    r"|(?:reveal|show|print|disclose|expose)\s+(?:the\s+)?"
    r"(?:(?:system|hidden|developer)\s+prompts?|credentials?|"
    r"(?:api|access|execution)\s+(?:keys?|tokens?)|passwords?|secrets?)"
    r"|(?:call|invoke|execute|run|trigger|use)\s+(?:the\s+)?"
    r"(?:tool\s+)?(?:search_my_collection|get_coin|collection_summary|"
    r"top_coins_by_value|portfolio_review|gap_analysis|market_search|"
    r"auction_search|price_trends|similar_lots)"
    r"(?:\s+(?:tool|function|capability))?"
    r"|(?:call|invoke|execute|run|trigger)\s+(?:the\s+)?(?:tool|function|capability)"
    r")\b"
)
_UNTRUSTED_EVIDENCE_TEXT_FIELDS = {
    "title",
    "description",
    "dealer_name",
    "ruler",
    "denomination",
    "era",
    "material",
    "auction_house",
    "sale_name",
    "lot_number",
    "lot_status",
    "matched_attributes",
    "material_differences",
}
_SOURCE_BACKED_FIELDS = {
    "title",
    "description",
    "dealer_name",
    "listed_price",
    "currency",
    "availability",
    "ruler",
    "denomination",
    "era",
    "material",
    "auction_house",
    "sale_name",
    "lot_number",
    "sale_date",
    "estimate",
    "current_bid",
    "lot_status",
    "amount",
    "price_basis",
}


class StrictSpecialistModel(BaseModel):
    model_config = ConfigDict(
        extra="forbid",
        str_strip_whitespace=True,
        allow_inf_nan=False,
    )

    @model_validator(mode="before")
    @classmethod
    def reject_token_shaped_content(cls, value: object) -> object:
        def contains_token(candidate: object) -> bool:
            if isinstance(candidate, str):
                return _TOKEN_RE.search(candidate) is not None
            if isinstance(candidate, list):
                return any(contains_token(item) for item in candidate)
            if isinstance(candidate, dict):
                return any(contains_token(item) for item in candidate.values())
            return False

        if contains_token(value):
            raise ValueError("token-shaped content is forbidden")
        return value


def _validate_utc(value: datetime) -> datetime:
    if value.tzinfo is None or value.utcoffset() != timedelta(0):
        raise ValueError("observed_at must be an RFC3339 UTC timestamp")
    return value.astimezone(timezone.utc)


def _validate_source_url(value: str) -> str:
    parsed = urlsplit(value)
    if parsed.scheme != "https":
        raise ValueError("source URL must use https")
    if parsed.username is not None or parsed.password is not None:
        raise ValueError("source URL must not contain user-info")
    host = parsed.hostname
    if not host:
        raise ValueError("source URL must contain a host")
    normalized_host = host.rstrip(".").lower()
    if normalized_host in _NON_PUBLIC_HOSTS or normalized_host.endswith(".localhost"):
        raise ValueError("source URL host is not public")
    try:
        address = ipaddress.ip_address(normalized_host)
    except ValueError:
        return value
    if not address.is_global:
        raise ValueError("source URL address is not public")
    return value


def _validate_registered_source(provider: str, value: str) -> None:
    allowed_hosts = _REGISTERED_SOURCE_HOSTS.get(provider)
    if allowed_hosts is None:
        raise ValueError("evidence provider is not registered")
    host = urlsplit(value).hostname
    if host is None:
        raise ValueError("source URL must contain a host")
    normalized_host = host.rstrip(".").lower()
    if not any(normalized_host == allowed or normalized_host.endswith(f".{allowed}") for allowed in allowed_hosts):
        raise ValueError("source URL host is outside the provider's registered source boundary")


class SpecialistQuery(StrictSpecialistModel):
    query: Annotated[str, StringConstraints(min_length=1, max_length=500)]
    limit: int = Field(default=5, ge=1, le=10)


class ProviderAttempt(StrictSpecialistModel):
    provider: BoundedProvider
    status: ProviderStatus
    observed_at: datetime
    accepted_items: int = Field(ge=0, le=10)
    warning_code: ProviderWarningCode | None = None

    @field_validator("observed_at")
    @classmethod
    def validate_observed_at(cls, value: datetime) -> datetime:
        return _validate_utc(value)

    @model_validator(mode="after")
    def validate_status_fields(self) -> ProviderAttempt:
        if self.status == "success":
            if self.accepted_items == 0:
                raise ValueError("successful provider attempts require accepted_items")
            if self.warning_code is not None:
                raise ValueError("successful provider attempts cannot include warning_code")
        elif self.status == "no_match":
            if self.accepted_items != 0 or self.warning_code is not None:
                raise ValueError("no-match attempts cannot accept items or include warning_code")
        else:
            if self.accepted_items != 0 or self.warning_code is None:
                raise ValueError("degraded provider attempts require a safe warning_code and zero items")
            expected_warning = {
                "timeout": "provider_timeout",
                "failure": "provider_failure",
                "unavailable": "provider_unavailable",
                "malformed": "provider_malformed",
            }[self.status]
            if self.warning_code != expected_warning:
                raise ValueError("provider warning_code must match the degraded status")
        return self


class FieldProvenance(StrictSpecialistModel):
    field: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    source_url: BoundedSourceURL
    observed_at: datetime
    confidence: Confidence
    verification_state: VerificationState

    @field_validator("source_url")
    @classmethod
    def validate_source_url(cls, value: str) -> str:
        return _validate_source_url(value)

    @field_validator("observed_at")
    @classmethod
    def validate_observed_at(cls, value: datetime) -> datetime:
        return _validate_utc(value)


class EvidenceItem(StrictSpecialistModel):
    kind: str
    source_url: BoundedSourceURL
    canonical_source_id: BoundedSourceURL
    provider: BoundedProvider
    observed_at: datetime
    confidence: Confidence
    verification_state: VerificationState
    title: BoundedTitle
    description: BoundedDescription | None = None
    provenance: list[FieldProvenance] = Field(min_length=1, max_length=20)

    @field_validator("source_url", "canonical_source_id")
    @classmethod
    def validate_source_urls(cls, value: str) -> str:
        return _validate_source_url(value)

    @field_validator("observed_at")
    @classmethod
    def validate_observed_at(cls, value: datetime) -> datetime:
        return _validate_utc(value)

    @model_validator(mode="after")
    def validate_provenance(self) -> EvidenceItem:
        model_fields = type(self).model_fields
        _validate_registered_source(self.provider, self.source_url)
        _validate_registered_source(self.provider, self.canonical_source_id)
        for field in _UNTRUSTED_EVIDENCE_TEXT_FIELDS.intersection(model_fields):
            value = getattr(self, field)
            values = value if isinstance(value, list) else [value]
            if any(isinstance(text, str) and _INSTRUCTION_RE.search(text) for text in values):
                raise ValueError(f"instruction-shaped content is forbidden in {field}")
        fields = [entry.field for entry in self.provenance]
        if len(fields) != len(set(fields)):
            raise ValueError("provenance fields must be unique per item")
        if "title" not in fields:
            raise ValueError("title requires provenance")
        for entry in self.provenance:
            if entry.source_url != self.source_url:
                raise ValueError("provenance source_url must match the evidence item")
            if entry.observed_at != self.observed_at:
                raise ValueError("provenance observed_at must match the evidence item")
            if entry.field not in model_fields:
                raise ValueError("provenance field must name a declared evidence field")
            if getattr(self, entry.field) is None:
                raise ValueError("provenance cannot reference an absent field")
        required_provenance = {
            field for field in _SOURCE_BACKED_FIELDS if field in model_fields and getattr(self, field) is not None
        }
        if not required_provenance.issubset(fields):
            raise ValueError("every populated source-backed field requires provenance")
        return self


class DealerListing(EvidenceItem):
    kind: Literal["dealer_listing"]
    dealer_name: BoundedText | None = None
    listed_price: Decimal | None = Field(default=None, ge=0)
    currency: Currency | None = None
    availability: Literal["available", "sold", "unknown"] | None = None
    ruler: BoundedText | None = None
    denomination: BoundedText | None = None
    era: BoundedText | None = None
    material: BoundedText | None = None


class AuctionLot(EvidenceItem):
    kind: Literal["auction_lot"]
    auction_house: BoundedText | None = None
    sale_name: BoundedText | None = None
    lot_number: Annotated[str, StringConstraints(min_length=1, max_length=100)] | None = None
    sale_date: date | None = None
    estimate: Decimal | None = Field(default=None, ge=0)
    current_bid: Decimal | None = Field(default=None, ge=0)
    currency: Currency | None = None
    lot_status: Annotated[str, StringConstraints(min_length=1, max_length=64)] | None = None
    ruler: BoundedText | None = None
    denomination: BoundedText | None = None
    era: BoundedText | None = None
    material: BoundedText | None = None


class SaleObservation(EvidenceItem):
    kind: Literal["sale_observation"]
    sale_date: date
    amount: Decimal = Field(ge=0)
    currency: Currency
    price_basis: Literal["hammer", "realized_including_premium"]


class SimilarLot(EvidenceItem):
    kind: Literal["similar_lot"]
    similarity_score: Decimal = Field(ge=0, le=1)
    matched_attributes: list[BoundedText] = Field(min_length=1, max_length=20)
    material_differences: list[BoundedText] = Field(max_length=20)


SpecialistEvidence = Annotated[
    DealerListing | AuctionLot | SaleObservation | SimilarLot,
    Field(discriminator="kind"),
]


class PriceTrendSummary(StrictSpecialistModel):
    state: Literal["rising", "stable", "declining", "unknown"]
    sample_size: int = Field(ge=0, le=10)
    date_from: date | None = None
    date_to: date | None = None
    currency: Currency | None = None
    price_basis: Literal["hammer", "realized_including_premium"] | None = None
    low: Decimal | None = Field(default=None, ge=0)
    median: Decimal | None = Field(default=None, ge=0)
    high: Decimal | None = Field(default=None, ge=0)
    confidence: Confidence
    limitations: list[BoundedWarning] = Field(default_factory=list, max_length=10)
    supporting_source_ids: list[BoundedSourceURL] = Field(default_factory=list, max_length=10)

    @field_validator("supporting_source_ids")
    @classmethod
    def validate_supporting_source_ids(cls, values: list[str]) -> list[str]:
        if len(values) != len(set(values)):
            raise ValueError("supporting_source_ids must be unique")
        return [_validate_source_url(value) for value in values]

    @model_validator(mode="after")
    def validate_summary(self) -> PriceTrendSummary:
        dates = (self.date_from, self.date_to)
        if (dates[0] is None) != (dates[1] is None):
            raise ValueError("trend date coverage requires both date_from and date_to")
        if dates[0] is not None and dates[1] is not None and dates[0] > dates[1]:
            raise ValueError("date_from cannot be after date_to")

        comparable_fields = (self.currency, self.price_basis, self.low, self.median, self.high)
        if any(value is None for value in comparable_fields) and any(value is not None for value in comparable_fields):
            raise ValueError("trend comparable metadata must be wholly present or absent")
        if self.low is not None and not self.low <= self.median <= self.high:
            raise ValueError("trend prices must satisfy low <= median <= high")
        if self.sample_size != len(self.supporting_source_ids):
            raise ValueError("sample_size must equal supporting source count")
        if self.sample_size == 0 and any(value is not None for value in dates + comparable_fields):
            raise ValueError("empty trends cannot contain sample metadata")
        if self.state != "unknown":
            if self.sample_size < 3 or self.date_from is None or self.date_to is None:
                raise ValueError("directional trends require at least three dated samples")
            if (self.date_to - self.date_from).days < 30:
                raise ValueError("directional trends require at least 30 days of coverage")
            if any(value is None for value in comparable_fields):
                raise ValueError("directional trends require comparable price metadata")
        return self


class TruncationMetadata(StrictSpecialistModel):
    truncated: bool
    original_bytes: int = Field(ge=0)
    persisted_bytes: int = Field(ge=0)
    digest: SHA256Digest
    omitted_items: int = Field(ge=0)

    @model_validator(mode="after")
    def validate_sizes(self) -> TruncationMetadata:
        if self.persisted_bytes > self.original_bytes:
            raise ValueError("persisted_bytes cannot exceed original_bytes")
        if not self.truncated and (self.persisted_bytes != self.original_bytes or self.omitted_items != 0):
            raise ValueError("untruncated results must preserve all bytes and items")
        if self.truncated and (self.persisted_bytes == self.original_bytes and self.omitted_items == 0):
            raise ValueError("truncated results must report an omitted byte or item")
        return self


class SpecialistResult(StrictSpecialistModel):
    schema_version: Literal[1] = 1
    capability: SpecialistCapability
    outcome: SpecialistOutcome
    items: list[SpecialistEvidence] = Field(default_factory=list, max_length=10)
    trend: PriceTrendSummary | None = None
    provider_attempts: list[ProviderAttempt] = Field(default_factory=list, max_length=10)
    warnings: list[BoundedWarning] = Field(default_factory=list, max_length=10)
    truncation: TruncationMetadata

    @model_validator(mode="after")
    def validate_envelope(self) -> SpecialistResult:
        expected_kind = {
            "market_search": "dealer_listing",
            "auction_search": "auction_lot",
            "price_trends": "sale_observation",
            "similar_lots": "similar_lot",
        }[self.capability]
        if any(item.kind != expected_kind for item in self.items):
            raise ValueError("evidence item kind does not match capability")

        allowed_providers = _CAPABILITY_PROVIDERS[self.capability]
        if any(item.provider not in allowed_providers for item in self.items):
            raise ValueError("evidence provider is not allowed for capability")
        if any(attempt.provider not in allowed_providers for attempt in self.provider_attempts):
            raise ValueError("provider attempt is not allowed for capability")

        source_ids = [item.canonical_source_id for item in self.items]
        if len(source_ids) != len(set(source_ids)):
            raise ValueError("canonical source identities must be unique")
        providers = [attempt.provider for attempt in self.provider_attempts]
        if len(providers) != len(set(providers)):
            raise ValueError("provider attempts must be unique and ordered")

        if self.outcome in {"complete", "partial"} and not self.items:
            raise ValueError("complete and partial outcomes require evidence")
        if self.outcome in {"no_match", "unavailable"} and self.items:
            raise ValueError("no_match and unavailable outcomes cannot contain evidence")
        degraded = any(attempt.status in _DEGRADED_PROVIDER_STATUSES for attempt in self.provider_attempts)
        if self.outcome == "partial" and not degraded:
            raise ValueError("partial outcomes require a degraded provider attempt")
        if self.outcome == "complete" and degraded:
            raise ValueError("complete outcomes cannot contain degraded provider attempts")
        if self.outcome == "no_match" and any(
            attempt.status not in {"success", "no_match"} for attempt in self.provider_attempts
        ):
            raise ValueError("no_match permits only successful/no-match attempts")
        if self.outcome == "unavailable" and not degraded:
            raise ValueError("unavailable requires a degraded provider attempt")

        if self.capability != "price_trends":
            if self.trend is not None:
                raise ValueError("trend is only valid for price_trends")
        elif self.trend is not None:
            if self.trend.sample_size != len(self.items):
                raise ValueError("trend sample_size must equal evidence item count")
            if set(self.trend.supporting_source_ids) != set(source_ids):
                raise ValueError("trend supporting sources must reference result evidence")
            if self.items:
                observations = [item for item in self.items if isinstance(item, SaleObservation)]
                observed_dates = [item.sale_date for item in observations]
                if self.trend.date_from != min(observed_dates) or self.trend.date_to != max(observed_dates):
                    raise ValueError("trend date coverage must be derived from sale observations")

                currencies = {item.currency for item in observations}
                bases = {item.price_basis for item in observations}
                comparable = len(currencies) == 1 and len(bases) == 1
                aggregate = (
                    self.trend.currency,
                    self.trend.price_basis,
                    self.trend.low,
                    self.trend.median,
                    self.trend.high,
                )
                if comparable:
                    amounts = [item.amount for item in observations]
                    expected = (
                        next(iter(currencies)),
                        next(iter(bases)),
                        min(amounts),
                        median(amounts),
                        max(amounts),
                    )
                    if aggregate != expected:
                        raise ValueError("trend price summary must be derived from comparable sale observations")
                    if (
                        len(observations) >= 3
                        and len(set(observed_dates)) >= 2
                        and (max(observed_dates) - min(observed_dates)).days >= 30
                        and all(item.verification_state == "verified" for item in observations)
                    ):
                        amounts_by_date: dict[date, list[Decimal]] = {}
                        for observation in observations:
                            amounts_by_date.setdefault(observation.sale_date, []).append(observation.amount)
                        ordered_date_amounts = [
                            median(amounts_by_date[sale_date]) for sale_date in sorted(amounts_by_date)
                        ]
                        first_amount = ordered_date_amounts[0]
                        last_amount = ordered_date_amounts[-1]
                        expected_state = (
                            "rising"
                            if last_amount > first_amount
                            else "declining"
                            if last_amount < first_amount
                            else "stable"
                        )
                        if self.trend.state != expected_state:
                            raise ValueError(
                                "trend direction must be derived from chronologically ordered observations"
                            )
                elif any(value is not None for value in aggregate):
                    raise ValueError("incomparable observations cannot produce aggregate price metadata")
                if not comparable and self.trend.state != "unknown":
                    raise ValueError("incomparable observations require an unknown trend state")
            elif any(
                value is not None
                for value in (
                    self.trend.date_from,
                    self.trend.date_to,
                    self.trend.currency,
                    self.trend.price_basis,
                    self.trend.low,
                    self.trend.median,
                    self.trend.high,
                )
            ):
                raise ValueError("trend without evidence cannot contain derived sample metadata")

            if self.trend.state != "unknown":
                if not self.items:
                    raise ValueError("directional trends require evidence")
                if any(item.verification_state != "verified" for item in self.items):
                    raise ValueError("directional trends require verified observations")
        elif self.items:
            if self.trend is None:
                raise ValueError("price_trends evidence requires a trend summary")

        if self.capability == "similar_lots":
            ordered = sorted(
                self.items,
                key=lambda item: (-item.similarity_score, item.canonical_source_id),
            )
            if self.items != ordered:
                raise ValueError("similar lots must use deterministic ranking")
        return self
