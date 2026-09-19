"""Strict normalized contracts for Coin Copilot specialist market tools."""

from __future__ import annotations

import asyncio
import hashlib
import ipaddress
import json
import re
from collections.abc import Awaitable, Callable, Mapping, Sequence
from dataclasses import dataclass
from datetime import date, datetime, timedelta, timezone
from decimal import Decimal
from statistics import median
from typing import Annotated, Any, Literal
from urllib.parse import urlsplit, urlunsplit

import httpx
from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    PlainSerializer,
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
JSONDecimal = Annotated[
    Decimal,
    PlainSerializer(lambda value: float(value), return_type=float, when_used="json"),
]

_DEGRADED_PROVIDER_STATUSES = {"timeout", "failure", "unavailable", "malformed"}
_NON_PUBLIC_HOSTS = {"localhost", "metadata.google.internal"}
_REGISTERED_SOURCE_HOSTS = {
    "numisbids": frozenset({"numisbids.com"}),
}
_CAPABILITY_PROVIDERS = {
    "market_search": frozenset(
        {"cng_dealer_search", "configured_dealer_search", "market_search_secondary"}
    ),
    "auction_search": frozenset(
        {"numisbids", "configured_auction_search", "auction_search_secondary"}
    ),
    "price_trends": frozenset({"numisbids", "price_trends_secondary"}),
    "similar_lots": frozenset(
        {"numisbids", "configured_auction_search", "similar_lots_secondary"}
    ),
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
_SAFE_PROVIDER_WARNINGS = {
    "timeout": "One configured source timed out; available evidence may be incomplete.",
    "failure": "One configured source could not be reached; available evidence may be incomplete.",
    "unavailable": "One configured source is unavailable; available evidence may be incomplete.",
    "malformed": "One configured source returned unusable data; available evidence may be incomplete.",
}
_WARNING_CODES: dict[str, ProviderWarningCode] = {
    "timeout": "provider_timeout",
    "failure": "provider_failure",
    "unavailable": "provider_unavailable",
    "malformed": "provider_malformed",
}


class ProviderUnavailableError(RuntimeError):
    """A configured specialist provider is not available for this execution."""


class ProviderMalformedError(RuntimeError):
    """A specialist provider returned a response that cannot be normalized."""


ProviderCallable = Callable[[str, int], Awaitable[Sequence[Mapping[str, Any]]]]
CancellationCheck = Callable[[], Awaitable[bool]]


async def raise_if_cancelled(check: CancellationCheck | None) -> None:
    """Stop specialist work as soon as the authoritative run is cancelled."""
    if check is not None and await check():
        raise asyncio.CancelledError


@dataclass(frozen=True)
class ProviderRunner:
    """One fixed provider boundary used by a specialist runner."""

    provider: str
    run: ProviderCallable
    allowed_hosts: frozenset[str] | None = None


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
        if self.provider not in {
            "configured_dealer_search",
            "configured_auction_search",
            "cng_dealer_search",
        }:
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
    listed_price: JSONDecimal | None = Field(default=None, ge=0)
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
    estimate: JSONDecimal | None = Field(default=None, ge=0)
    current_bid: JSONDecimal | None = Field(default=None, ge=0)
    currency: Currency | None = None
    lot_status: Annotated[str, StringConstraints(min_length=1, max_length=64)] | None = None
    ruler: BoundedText | None = None
    denomination: BoundedText | None = None
    era: BoundedText | None = None
    material: BoundedText | None = None


class SaleObservation(EvidenceItem):
    kind: Literal["sale_observation"]
    sale_date: date
    amount: JSONDecimal = Field(ge=0)
    currency: Currency
    price_basis: Literal["hammer", "realized_including_premium"]


class SimilarLot(EvidenceItem):
    kind: Literal["similar_lot"]
    similarity_score: JSONDecimal = Field(ge=0, le=1)
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
    low: JSONDecimal | None = Field(default=None, ge=0)
    median: JSONDecimal | None = Field(default=None, ge=0)
    high: JSONDecimal | None = Field(default=None, ge=0)
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


def canonical_source_identity(url: str) -> str:
    """Return the stable comparison identity without changing the display URL."""
    validated = _validate_source_url(url)
    parsed = urlsplit(validated)
    host = (parsed.hostname or "").rstrip(".").lower()
    port = parsed.port
    netloc = host if port in {None, 443} else f"{host}:{port}"
    return urlunsplit(("https", netloc, parsed.path or "/", parsed.query, ""))


def validate_registered_source_url(
    provider: str,
    url: str,
    allowed_hosts: frozenset[str] | None = None,
) -> str:
    """Validate a source URL against the provider's fixed source boundary."""
    validated = _validate_source_url(url)
    if allowed_hosts is None:
        _validate_registered_source(provider, validated)
    else:
        host = (urlsplit(validated).hostname or "").rstrip(".").lower()
        if not any(host == allowed or host.endswith(f".{allowed}") for allowed in allowed_hosts):
            raise ValueError("source URL host is not configured")
    return validated


def _clean_optional_text(value: object, *, maximum: int = 300) -> str | None:
    if value is None:
        return None
    text = str(value).strip()
    return text[:maximum] if text else None


def _parse_decimal(value: object) -> Decimal | None:
    if value is None or value == "":
        return None
    if isinstance(value, Decimal):
        return value
    if isinstance(value, int | float):
        return Decimal(str(value))
    match = re.search(r"[\d,]+(?:\.\d+)?", str(value))
    if not match:
        return None
    try:
        return Decimal(match.group(0).replace(",", ""))
    except Exception:
        return None


def _parse_currency(value: object) -> str | None:
    if value is None:
        return None
    text = str(value).upper()
    for token in ("USD", "EUR", "GBP", "CHF"):
        if token in text:
            return token
    if "$" in text:
        return "USD"
    return None


def _parse_sale_date(value: object) -> date | None:
    if isinstance(value, datetime):
        return value.date()
    if isinstance(value, date):
        return value
    text = str(value or "").strip()
    if not text:
        return None
    for pattern in ("%Y-%m-%d", "%d %b %Y", "%d %B %Y"):
        try:
            return datetime.strptime(text, pattern).date()
        except ValueError:
            continue
    return None


def _provenance(
    fields: Sequence[str],
    *,
    source_url: str,
    observed_at: datetime,
    confidence: Confidence,
    verification_state: VerificationState,
) -> list[FieldProvenance]:
    return [
        FieldProvenance(
            field=field,
            source_url=source_url,
            observed_at=observed_at,
            confidence=confidence,
            verification_state=verification_state,
        )
        for field in fields
    ]


def adapt_dealer_candidate(
    candidate: Mapping[str, Any],
    *,
    provider: str,
    observed_at: datetime,
    allowed_hosts: frozenset[str] | None = None,
) -> DealerListing:
    """Normalize one provider-observed dealer candidate without enrichment."""
    source_url = validate_registered_source_url(
        provider,
        str(candidate.get("sourceUrl") or candidate.get("source_url") or candidate.get("url") or "").strip(),
        allowed_hosts,
    )
    title = _clean_optional_text(candidate.get("name") or candidate.get("title"))
    if not title:
        raise ValueError("dealer candidate requires a source-backed title")

    price_value = candidate.get("listed_price")
    if price_value is None:
        price_value = candidate.get("estPrice") or candidate.get("price")
    listed_price = _parse_decimal(price_value)
    currency = _parse_currency(candidate.get("currency") or price_value)
    raw_availability = str(candidate.get("availability") or "").strip().lower()
    availability = {
        "available": "available",
        "in stock": "available",
        "sold": "sold",
        "sold out": "sold",
        "unknown": "unknown",
    }.get(raw_availability)

    values = {
        "description": _clean_optional_text(candidate.get("description"), maximum=500),
        "dealer_name": _clean_optional_text(candidate.get("sourceName") or candidate.get("dealer_name")),
        "listed_price": listed_price,
        "currency": currency,
        "availability": availability,
        "ruler": _clean_optional_text(candidate.get("ruler")),
        "denomination": _clean_optional_text(candidate.get("denomination")),
        "era": _clean_optional_text(candidate.get("era")),
        "material": _clean_optional_text(candidate.get("material")),
    }
    proven_fields = ["title", *(field for field, value in values.items() if value is not None)]
    return DealerListing(
        kind="dealer_listing",
        source_url=source_url,
        canonical_source_id=canonical_source_identity(source_url),
        provider=provider,
        observed_at=observed_at,
        confidence="high",
        verification_state="verified",
        title=title,
        provenance=_provenance(
            proven_fields,
            source_url=source_url,
            observed_at=observed_at,
            confidence="high",
            verification_state="verified",
        ),
        **values,
    )


def adapt_auction_candidate(
    candidate: Mapping[str, Any],
    *,
    provider: str,
    observed_at: datetime,
    allowed_hosts: frozenset[str] | None = None,
) -> AuctionLot:
    """Normalize one provider-observed auction candidate without enrichment."""
    source_url = validate_registered_source_url(
        provider,
        str(candidate.get("url") or candidate.get("sourceUrl") or candidate.get("source_url") or "").strip(),
        allowed_hosts,
    )
    title = _clean_optional_text(candidate.get("title") or candidate.get("name"))
    if not title:
        raise ValueError("auction candidate requires a source-backed title")

    estimate = _parse_decimal(candidate.get("estimate"))
    current_bid = _parse_decimal(candidate.get("currentBid") or candidate.get("current_bid"))
    currency = _parse_currency(candidate.get("currency"))
    values = {
        "description": _clean_optional_text(candidate.get("description"), maximum=500),
        "auction_house": _clean_optional_text(candidate.get("auctionHouse") or candidate.get("auction_house")),
        "sale_name": _clean_optional_text(candidate.get("saleName") or candidate.get("sale_name")),
        "lot_number": _clean_optional_text(candidate.get("lotNumber") or candidate.get("lot_number"), maximum=100),
        "sale_date": _parse_sale_date(candidate.get("saleDate") or candidate.get("sale_date")),
        "estimate": estimate,
        "current_bid": current_bid,
        "currency": currency,
        "lot_status": _clean_optional_text(candidate.get("lotStatus") or candidate.get("lot_status"), maximum=64),
        "ruler": _clean_optional_text(candidate.get("ruler")),
        "denomination": _clean_optional_text(candidate.get("denomination")),
        "era": _clean_optional_text(candidate.get("era")),
        "material": _clean_optional_text(candidate.get("material")),
    }
    proven_fields = ["title", *(field for field, value in values.items() if value is not None)]
    return AuctionLot(
        kind="auction_lot",
        source_url=source_url,
        canonical_source_id=canonical_source_identity(source_url),
        provider=provider,
        observed_at=observed_at,
        confidence="high",
        verification_state="verified",
        title=title,
        provenance=_provenance(
            proven_fields,
            source_url=source_url,
            observed_at=observed_at,
            confidence="high",
            verification_state="verified",
        ),
        **values,
    )


def adapt_sale_observation(
    candidate: Mapping[str, Any],
    *,
    provider: str,
    observed_at: datetime,
) -> SaleObservation:
    """Normalize one completed-sale observation without conversion or inference."""
    source_url = validate_registered_source_url(
        provider,
        str(candidate.get("url") or candidate.get("sourceUrl") or candidate.get("source_url") or "").strip(),
    )
    title = _clean_optional_text(candidate.get("title") or candidate.get("name"))
    sale_date = _parse_sale_date(candidate.get("saleDate") or candidate.get("sale_date"))
    amount_value = candidate.get("amount")
    if amount_value is None:
        amount_value = candidate.get("hammerPrice")
    if amount_value is None:
        amount_value = candidate.get("realizedPrice")
    amount = _parse_decimal(amount_value)
    currency = _parse_currency(candidate.get("currency"))
    raw_basis = str(candidate.get("priceBasis") or candidate.get("price_basis") or "").strip().lower()
    price_basis = {
        "hammer": "hammer",
        "realized_including_premium": "realized_including_premium",
        "including premium": "realized_including_premium",
        "realized including premium": "realized_including_premium",
    }.get(raw_basis)
    if not title or sale_date is None or amount is None or currency is None or price_basis is None:
        raise ValueError("sale observation requires complete source-backed sale fields")

    description = _clean_optional_text(candidate.get("description"), maximum=500)
    verification_state = str(candidate.get("verificationState") or "verified").strip().lower()
    confidence = str(candidate.get("confidence") or "high").strip().lower()
    if verification_state not in {"verified", "partial"} or confidence not in {"high", "medium", "low"}:
        raise ValueError("sale observation verification metadata is invalid")
    fields = ["title", "sale_date", "amount", "currency", "price_basis"]
    if description is not None:
        fields.append("description")
    return SaleObservation(
        kind="sale_observation",
        source_url=source_url,
        canonical_source_id=canonical_source_identity(source_url),
        provider=provider,
        observed_at=observed_at,
        confidence=confidence,
        verification_state=verification_state,
        title=title,
        description=description,
        sale_date=sale_date,
        amount=amount,
        currency=currency,
        price_basis=price_basis,
        provenance=_provenance(
            fields,
            source_url=source_url,
            observed_at=observed_at,
            confidence=confidence,
            verification_state=verification_state,
        ),
    )


def _merge_duplicate(
    existing: DealerListing | AuctionLot,
    candidate: DealerListing | AuctionLot,
) -> tuple[DealerListing | AuctionLot, bool]:
    fields = (
        (
            "description",
            "dealer_name",
            "listed_price",
            "currency",
            "availability",
            "ruler",
            "denomination",
            "era",
            "material",
        )
        if isinstance(existing, DealerListing)
        else (
            "description",
            "auction_house",
            "sale_name",
            "lot_number",
            "sale_date",
            "estimate",
            "current_bid",
            "currency",
            "lot_status",
            "ruler",
            "denomination",
            "era",
            "material",
        )
    )
    verification_rank = {"partial": 0, "verified": 1}
    confidence_rank = {"low": 0, "medium": 1, "high": 2}

    def strength(item: DealerListing | AuctionLot) -> tuple[int, int, int]:
        return (
            verification_rank[item.verification_state],
            confidence_rank[item.confidence],
            len(item.provenance),
        )

    base, other = (candidate, existing) if strength(candidate) > strength(existing) else (existing, candidate)
    updates: dict[str, Any] = {}
    conflict = existing.title != candidate.title
    for field in fields:
        base_value = getattr(base, field)
        other_value = getattr(other, field)
        if base_value is None and other_value is not None:
            updates[field] = other_value
        elif base_value is not None and other_value is not None and base_value != other_value:
            conflict = True
    if not updates:
        return base, conflict
    merged = base.model_copy(update=updates)
    proven_fields = [
        field
        for field in type(merged).model_fields
        if field in _SOURCE_BACKED_FIELDS and getattr(merged, field, None) is not None
    ]
    merged = merged.model_copy(
        update={
            "provenance": _provenance(
                proven_fields,
                source_url=merged.source_url,
                observed_at=merged.observed_at,
                confidence=merged.confidence,
                verification_state=merged.verification_state,
            )
        }
    )
    return type(merged).model_validate(merged.model_dump()), conflict


def _deduplicate_items(
    items: Sequence[DealerListing | AuctionLot],
) -> tuple[list[DealerListing | AuctionLot], list[str]]:
    deduplicated: dict[str, DealerListing | AuctionLot] = {}
    warnings: list[str] = []
    for item in items:
        existing = deduplicated.get(item.canonical_source_id)
        if existing is None:
            deduplicated[item.canonical_source_id] = item
            continue
        merged, conflict = _merge_duplicate(existing, item)
        deduplicated[item.canonical_source_id] = merged
        if conflict and "Duplicate source observations contained conflicting facts." not in warnings:
            warnings.append("Duplicate source observations contained conflicting facts.")
    return list(deduplicated.values()), warnings


def _sale_strength(item: SaleObservation) -> tuple[int, int]:
    return (
        {"partial": 0, "verified": 1}[item.verification_state],
        {"low": 0, "medium": 1, "high": 2}[item.confidence],
    )


def _deduplicate_sales(
    items: Sequence[SaleObservation],
) -> tuple[list[SaleObservation], list[str]]:
    deduplicated: dict[str, SaleObservation] = {}
    warnings: list[str] = []
    for item in items:
        existing = deduplicated.get(item.canonical_source_id)
        if existing is None:
            deduplicated[item.canonical_source_id] = item
            continue
        fields_match = (
            existing.title == item.title
            and existing.sale_date == item.sale_date
            and existing.amount == item.amount
            and existing.currency == item.currency
            and existing.price_basis == item.price_basis
        )
        if not fields_match and "Duplicate sale observations contained conflicting facts." not in warnings:
            warnings.append("Duplicate sale observations contained conflicting facts.")
        if _sale_strength(item) > _sale_strength(existing):
            deduplicated[item.canonical_source_id] = item
    return list(deduplicated.values()), warnings


def _build_price_trend(items: Sequence[SaleObservation]) -> PriceTrendSummary:
    if not items:
        return PriceTrendSummary(
            state="unknown",
            sample_size=0,
            confidence="low",
            limitations=["No verified completed-sale observations were available."],
            supporting_source_ids=[],
        )

    dates = [item.sale_date for item in items]
    currencies = {item.currency for item in items}
    price_bases = {item.price_basis for item in items}
    limitations: list[str] = []
    comparable = len(currencies) == 1 and len(price_bases) == 1
    all_verified = all(item.verification_state == "verified" for item in items)
    enough_samples = len(items) >= 3
    enough_dates = len(set(dates)) >= 2
    enough_coverage = (max(dates) - min(dates)).days >= 30

    if len(currencies) > 1:
        limitations.append("Currency observations are not comparable and were not converted.")
    if len(price_bases) > 1:
        limitations.append("Price basis observations kept hammer and premium-inclusive values separate.")
    if not enough_samples:
        limitations.append("Fewer than three verified completed sales were available.")
    if not enough_dates:
        limitations.append("Completed sales covered fewer than two distinct sale dates.")
    if not enough_coverage:
        limitations.append("Completed sales covered less than 30 days.")
    if not all_verified:
        limitations.append("At least one completed-sale observation was only partially verified.")

    state: Literal["rising", "stable", "declining", "unknown"] = "unknown"
    aggregate: dict[str, Any] = {
        "currency": None,
        "price_basis": None,
        "low": None,
        "median": None,
        "high": None,
    }
    if comparable:
        amounts = [item.amount for item in items]
        aggregate = {
            "currency": next(iter(currencies)),
            "price_basis": next(iter(price_bases)),
            "low": min(amounts),
            "median": median(amounts),
            "high": max(amounts),
        }
        if enough_samples and enough_dates and enough_coverage and all_verified:
            by_date: dict[date, list[Decimal]] = {}
            for item in items:
                by_date.setdefault(item.sale_date, []).append(item.amount)
            ordered = [median(by_date[sale_date]) for sale_date in sorted(by_date)]
            state = "rising" if ordered[-1] > ordered[0] else "declining" if ordered[-1] < ordered[0] else "stable"

    confidence: Confidence = (
        "high"
        if state != "unknown"
        else "medium"
        if comparable and len(items) >= 3
        else "low"
    )
    return PriceTrendSummary(
        state=state,
        sample_size=len(items),
        date_from=min(dates),
        date_to=max(dates),
        confidence=confidence,
        limitations=limitations,
        supporting_source_ids=[item.canonical_source_id for item in items],
        **aggregate,
    )


def _finalize_result(
    *,
    capability: SpecialistCapability,
    outcome: SpecialistOutcome,
    items: Sequence[DealerListing | AuctionLot | SaleObservation | SimilarLot],
    provider_attempts: Sequence[ProviderAttempt],
    warnings: Sequence[str],
    omitted_items: int,
    trend: PriceTrendSummary | None = None,
) -> SpecialistResult:
    core = {
        "schema_version": 1,
        "capability": capability,
        "outcome": outcome,
        "items": [item.model_dump(mode="json") for item in items],
        "trend": trend.model_dump(mode="json") if trend is not None else None,
        "provider_attempts": [attempt.model_dump(mode="json") for attempt in provider_attempts],
        "warnings": list(warnings)[:10],
    }
    digest = hashlib.sha256(
        json.dumps(core, separators=(",", ":"), sort_keys=True).encode()
    ).hexdigest()
    size = 0
    for _ in range(5):
        result = SpecialistResult.model_validate(
            {
                **core,
                "truncation": {
                    "truncated": omitted_items > 0,
                    "original_bytes": size,
                    "persisted_bytes": size,
                    "digest": digest,
                    "omitted_items": omitted_items,
                },
            }
        )
        encoded_size = len(result.model_dump_json().encode())
        if encoded_size == size:
            return result
        size = encoded_size
    return result


_SIMILARITY_STOPWORDS = frozenset(
    {
        "active",
        "auction",
        "coin",
        "coins",
        "find",
        "for",
        "lot",
        "lots",
        "my",
        "of",
        "similar",
        "the",
        "to",
        "with",
    }
)


def project_similar_lots(query: str, auction_result: SpecialistResult) -> SpecialistResult:
    """Rank configured auction evidence against explicit query attributes."""
    if auction_result.capability != "auction_search":
        raise ValueError("similar-lot projection requires auction search evidence")

    normalized_query = query.casefold()
    query_terms = {
        term
        for term in re.findall(r"[a-z0-9]+", normalized_query)
        if len(term) >= 3 and term not in _SIMILARITY_STOPWORDS
    }
    projected: list[SimilarLot] = []
    for item in auction_result.items:
        if not isinstance(item, AuctionLot):
            continue
        matched: list[str] = []
        for label, value in (
            ("ruler", item.ruler),
            ("denomination", item.denomination),
            ("era", item.era),
            ("material", item.material),
        ):
            if value and value.casefold() in normalized_query:
                matched.append(f"{label}: {value}")
        title_terms = {
            term
            for term in re.findall(r"[a-z0-9]+", item.title.casefold())
            if len(term) >= 3 and term not in _SIMILARITY_STOPWORDS
        }
        for term in sorted(query_terms.intersection(title_terms))[:5]:
            marker = f"title term: {term}"
            if marker not in matched:
                matched.append(marker)
        if not matched:
            continue
        similarity_score = min(1.0, 0.35 + (0.1 * len(matched)))
        provenance_fields = ["title"]
        if item.description is not None:
            provenance_fields.append("description")
        projected.append(
            SimilarLot(
                kind="similar_lot",
                source_url=item.source_url,
                canonical_source_id=item.canonical_source_id,
                provider=item.provider,
                observed_at=item.observed_at,
                confidence=item.confidence,
                verification_state=item.verification_state,
                title=item.title,
                description=item.description,
                similarity_score=similarity_score,
                matched_attributes=matched,
                material_differences=[],
                provenance=_provenance(
                    provenance_fields,
                    source_url=item.source_url,
                    observed_at=item.observed_at,
                    confidence=item.confidence,
                    verification_state=item.verification_state,
                ),
            )
        )

    projected.sort(key=lambda item: (-item.similarity_score, item.canonical_source_id))
    projected_count = len(projected)
    result_limit = min(projected_count, 10)
    projected = projected[:result_limit]
    degraded = any(
        attempt.status in _DEGRADED_PROVIDER_STATUSES
        for attempt in auction_result.provider_attempts
    )
    if projected:
        outcome: SpecialistOutcome = "partial" if degraded else "complete"
    elif degraded:
        outcome = "unavailable"
    else:
        outcome = "no_match"
    attempts = [
        attempt.model_copy(
            update={
                "status": attempt.status if attempt.status in _DEGRADED_PROVIDER_STATUSES else (
                    "success" if projected else "no_match"
                ),
                "accepted_items": len(projected) if attempt.status not in _DEGRADED_PROVIDER_STATUSES else 0,
                "warning_code": attempt.warning_code if attempt.status in _DEGRADED_PROVIDER_STATUSES else None,
            }
        )
        for attempt in auction_result.provider_attempts
    ]
    return _finalize_result(
        capability="similar_lots",
        outcome=outcome,
        items=projected,
        provider_attempts=attempts,
        warnings=auction_result.warnings,
        omitted_items=max(0, projected_count - result_limit),
    )


async def run_provider_search(
    *,
    capability: Literal["market_search", "auction_search"],
    query: SpecialistQuery | Mapping[str, Any],
    provider_runners: Sequence[ProviderRunner],
    observed_at: datetime | None = None,
    cancellation_check: CancellationCheck | None = None,
) -> SpecialistResult:
    """Run fixed provider adapters and aggregate only normalized evidence."""
    parsed_query = query if isinstance(query, SpecialistQuery) else SpecialistQuery.model_validate(query)
    timestamp = _validate_utc(observed_at or datetime.now(timezone.utc))
    attempts: list[ProviderAttempt] = []
    accepted: list[DealerListing | AuctionLot] = []
    warnings: list[str] = []
    omitted_items = 0

    for provider_runner in provider_runners:
        status: ProviderStatus
        try:
            await raise_if_cancelled(cancellation_check)
            raw_candidates = await provider_runner.run(parsed_query.query, parsed_query.limit)
            await raise_if_cancelled(cancellation_check)
            if isinstance(raw_candidates, str | bytes) or not isinstance(raw_candidates, Sequence):
                raise ProviderMalformedError
            normalized: list[DealerListing | AuctionLot] = []
            invalid_count = 0
            for candidate in raw_candidates:
                if not isinstance(candidate, Mapping):
                    invalid_count += 1
                    continue
                try:
                    item = (
                        adapt_dealer_candidate(
                            candidate,
                            provider=provider_runner.provider,
                            observed_at=timestamp,
                            allowed_hosts=provider_runner.allowed_hosts,
                        )
                        if capability == "market_search"
                        else adapt_auction_candidate(
                            candidate,
                            provider=provider_runner.provider,
                            observed_at=timestamp,
                            allowed_hosts=provider_runner.allowed_hosts,
                        )
                    )
                except (ValueError, TypeError):
                    invalid_count += 1
                    continue
                if isinstance(item, DealerListing) and item.availability != "available":
                    continue
                normalized.append(item)
            if invalid_count and not normalized:
                raise ProviderMalformedError
            if invalid_count:
                warnings.append(_SAFE_PROVIDER_WARNINGS["malformed"])
            accepted.extend(normalized)
            status = "success" if normalized else "no_match"
            attempts.append(
                ProviderAttempt(
                    provider=provider_runner.provider,
                    status=status,
                    observed_at=timestamp,
                    accepted_items=min(len(normalized), 10),
                    warning_code=None,
                )
            )
        except (TimeoutError, asyncio.TimeoutError, httpx.TimeoutException):
            status = "timeout"
        except ProviderUnavailableError:
            status = "unavailable"
        except ProviderMalformedError:
            status = "malformed"
        except ValueError:
            status = "malformed"
        except httpx.TransportError:
            status = "failure"
        except Exception:
            status = "failure"
        if status in _DEGRADED_PROVIDER_STATUSES:
            attempts.append(
                ProviderAttempt(
                    provider=provider_runner.provider,
                    status=status,
                    observed_at=timestamp,
                    accepted_items=0,
                    warning_code=_WARNING_CODES[status],
                )
            )
            warnings.append(_SAFE_PROVIDER_WARNINGS[status])

    deduplicated, duplicate_warnings = _deduplicate_items(accepted)
    warnings.extend(duplicate_warnings)
    result_limit = min(parsed_query.limit, 10)
    omitted_items += max(0, len(deduplicated) - result_limit)
    items = deduplicated[:result_limit]
    degraded = any(attempt.status in _DEGRADED_PROVIDER_STATUSES for attempt in attempts)
    if items:
        outcome: SpecialistOutcome = "partial" if degraded else "complete"
    elif degraded:
        outcome = "unavailable"
    else:
        outcome = "no_match"
    return _finalize_result(
        capability=capability,
        outcome=outcome,
        items=items,
        provider_attempts=attempts,
        warnings=warnings,
        omitted_items=omitted_items,
    )


async def run_price_trend_search(
    *,
    query: SpecialistQuery | Mapping[str, Any],
    provider_runners: Sequence[ProviderRunner],
    observed_at: datetime | None = None,
    cancellation_check: CancellationCheck | None = None,
) -> SpecialistResult:
    """Run fixed completed-sale providers and compute a deterministic trend."""
    parsed_query = query if isinstance(query, SpecialistQuery) else SpecialistQuery.model_validate(query)
    timestamp = _validate_utc(observed_at or datetime.now(timezone.utc))
    attempts: list[ProviderAttempt] = []
    accepted: list[SaleObservation] = []
    warnings: list[str] = []

    for provider_runner in provider_runners:
        status: ProviderStatus
        try:
            await raise_if_cancelled(cancellation_check)
            raw_candidates = await provider_runner.run(parsed_query.query, parsed_query.limit)
            await raise_if_cancelled(cancellation_check)
            if isinstance(raw_candidates, str | bytes) or not isinstance(raw_candidates, Sequence):
                raise ProviderMalformedError
            normalized: list[SaleObservation] = []
            invalid_count = 0
            for candidate in raw_candidates:
                if not isinstance(candidate, Mapping):
                    invalid_count += 1
                    continue
                try:
                    normalized.append(
                        adapt_sale_observation(
                            candidate,
                            provider=provider_runner.provider,
                            observed_at=timestamp,
                        )
                    )
                except (TypeError, ValueError):
                    invalid_count += 1
            if invalid_count and not normalized:
                raise ProviderMalformedError
            if invalid_count:
                warnings.append(_SAFE_PROVIDER_WARNINGS["malformed"])
            accepted.extend(normalized)
            status = "success" if normalized else "no_match"
            attempts.append(
                ProviderAttempt(
                    provider=provider_runner.provider,
                    status=status,
                    observed_at=timestamp,
                    accepted_items=min(len(normalized), 10),
                    warning_code=None,
                )
            )
        except (TimeoutError, asyncio.TimeoutError, httpx.TimeoutException):
            status = "timeout"
        except ProviderUnavailableError:
            status = "unavailable"
        except (ProviderMalformedError, ValueError):
            status = "malformed"
        except httpx.TransportError:
            status = "failure"
        except Exception:
            status = "failure"
        if status in _DEGRADED_PROVIDER_STATUSES:
            attempts.append(
                ProviderAttempt(
                    provider=provider_runner.provider,
                    status=status,
                    observed_at=timestamp,
                    accepted_items=0,
                    warning_code=_WARNING_CODES[status],
                )
            )
            warnings.append(_SAFE_PROVIDER_WARNINGS[status])

    deduplicated, duplicate_warnings = _deduplicate_sales(accepted)
    warnings.extend(duplicate_warnings)
    result_limit = min(parsed_query.limit, 10)
    omitted_items = max(0, len(deduplicated) - result_limit)
    items = deduplicated[:result_limit]
    trend = _build_price_trend(items)
    degraded = any(attempt.status in _DEGRADED_PROVIDER_STATUSES for attempt in attempts)
    if items:
        outcome: SpecialistOutcome = "partial" if degraded else "complete"
    elif degraded:
        outcome = "unavailable"
    else:
        outcome = "no_match"
    return _finalize_result(
        capability="price_trends",
        outcome=outcome,
        items=items,
        trend=trend,
        provider_attempts=attempts,
        warnings=warnings,
        omitted_items=omitted_items,
    )
