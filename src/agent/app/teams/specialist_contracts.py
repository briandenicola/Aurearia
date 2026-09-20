"""Typed specialist results for Coin Copilot market tools.

One schema, validated once, here. Go persists and streams the result as data
and the browser renders it; neither re-implements this schema. What is enforced
at this boundary is what actually protects the owner:

- source URLs are https, credential-free and resolve to a configured host
- provider text is untrusted data, screened for injection and token shapes
- results are bounded in size and item count
- outcomes are typed, so a failed source can never look like "no matches"
"""

from __future__ import annotations

import asyncio
import json
import logging
import re
from collections.abc import Awaitable, Callable, Mapping, Sequence
from dataclasses import dataclass
from datetime import date, datetime, timedelta, timezone
from decimal import Decimal
from statistics import median
from typing import Annotated, Any, Literal
from urllib.parse import urlsplit, urlunsplit

import httpx
from anthropic import APIError as AnthropicAPIError
from ollama import ResponseError as OllamaResponseError
from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    PlainSerializer,
    StringConstraints,
    field_validator,
    model_validator,
)

logger = logging.getLogger(__name__)

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
JSONDecimal = Annotated[
    Decimal,
    PlainSerializer(lambda value: float(value), return_type=float, when_used="json"),
]

MAX_ITEMS = 10

_DEGRADED_PROVIDER_STATUSES = {"timeout", "failure", "unavailable", "malformed"}
_NON_PUBLIC_HOSTS = {"localhost", "metadata.google.internal"}
_REGISTERED_SOURCE_HOSTS = {
    "numisbids": frozenset({"numisbids.com"}),
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
# Provider text that reaches the model or the owner, screened for injection.
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

# Failures of an outside service (network, HTTP, model API). Anything else a
# provider raises is a defect and must propagate instead of being reported to
# the model as an unavailable source.
_EXTERNAL_PROVIDER_ERRORS = (httpx.HTTPError, OSError, AnthropicAPIError, OllamaResponseError)


def _log_degraded_provider(capability: str, provider: str, status: str, exc: BaseException) -> None:
    logger.warning(
        "Specialist provider degraded capability=%s provider=%s status=%s error_type=%s",
        capability,
        provider,
        status,
        type(exc).__name__,
    )


class ProviderUnavailableError(RuntimeError):
    """A configured specialist provider is not available for this execution."""


class ProviderMalformedError(RuntimeError):
    """A specialist provider returned a response that cannot be normalized."""


CancellationCheck = Callable[[], Awaitable[bool]]


async def raise_if_cancelled(check: CancellationCheck | None) -> None:
    if check and await check():
        raise asyncio.CancelledError


@dataclass(frozen=True)
class ProviderRunner:
    """One configured source: a callable returning raw candidate mappings."""

    provider: str
    run: Callable[[str, int], Awaitable[Sequence[Mapping[str, Any]]]]
    allowed_hosts: frozenset[str] | None = None


class StrictSpecialistModel(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    @field_validator("*", mode="after")
    @classmethod
    def reject_token_shaped_text(cls, value: Any) -> Any:
        values = value if isinstance(value, list) else [value]
        if any(isinstance(item, str) and _TOKEN_RE.search(item) for item in values):
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
        import ipaddress

        address = ipaddress.ip_address(normalized_host)
    except ValueError:
        return value
    if not address.is_global:
        raise ValueError("source URL address is not public")
    return value


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
    hosts = allowed_hosts if allowed_hosts is not None else _REGISTERED_SOURCE_HOSTS.get(provider)
    if hosts is None:
        raise ValueError("evidence provider is not registered")
    host = (urlsplit(validated).hostname or "").rstrip(".").lower()
    if not any(host == allowed or host.endswith(f".{allowed}") for allowed in hosts):
        raise ValueError("source URL host is not configured")
    return validated


class SpecialistQuery(StrictSpecialistModel):
    query: Annotated[str, StringConstraints(min_length=1, max_length=500)]
    limit: int = Field(default=5, ge=1, le=MAX_ITEMS)


class ProviderAttempt(StrictSpecialistModel):
    provider: BoundedProvider
    status: ProviderStatus
    observed_at: datetime
    accepted_items: int = Field(ge=0, le=MAX_ITEMS)
    warning_code: ProviderWarningCode | None = None

    @field_validator("observed_at")
    @classmethod
    def validate_observed_at(cls, value: datetime) -> datetime:
        return _validate_utc(value)

    @model_validator(mode="after")
    def validate_status_fields(self) -> ProviderAttempt:
        degraded = self.status in _DEGRADED_PROVIDER_STATUSES
        if degraded and self.warning_code != _WARNING_CODES[self.status]:
            raise ValueError("degraded provider attempts require a matching warning code")
        if not degraded and self.warning_code is not None:
            raise ValueError("healthy provider attempts cannot carry a warning code")
        return self


class CandidateReference(StrictSpecialistModel):
    catalog: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    number: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    volume: Annotated[str, StringConstraints(max_length=64)] | None = None
    uri: BoundedSourceURL | None = None

    @field_validator("uri")
    @classmethod
    def validate_uri(cls, value: str | None) -> str | None:
        return None if value is None else _validate_source_url(value)


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
    image_url: BoundedSourceURL | None = None
    candidate_references: list[CandidateReference] = Field(default_factory=list, max_length=5)

    @field_validator("source_url", "canonical_source_id", "image_url")
    @classmethod
    def validate_source_urls(cls, value: str | None) -> str | None:
        return None if value is None else _validate_source_url(value)

    @field_validator("observed_at")
    @classmethod
    def validate_observed_at(cls, value: datetime) -> datetime:
        return _validate_utc(value)

    @model_validator(mode="after")
    def reject_instruction_shaped_text(self) -> EvidenceItem:
        for field in _UNTRUSTED_EVIDENCE_TEXT_FIELDS.intersection(type(self).model_fields):
            value = getattr(self, field)
            values = value if isinstance(value, list) else [value]
            if any(isinstance(text, str) and _INSTRUCTION_RE.search(text) for text in values):
                raise ValueError(f"instruction-shaped content is forbidden in {field}")
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
    similarity_score: float = Field(ge=0, le=1)
    matched_attributes: list[BoundedText] = Field(min_length=1, max_length=20)
    material_differences: list[BoundedText] = Field(default_factory=list, max_length=20)


SpecialistEvidence = Annotated[
    DealerListing | AuctionLot | SaleObservation | SimilarLot,
    Field(discriminator="kind"),
]


class PriceTrendSummary(StrictSpecialistModel):
    state: Literal["rising", "stable", "declining", "unknown"]
    sample_size: int = Field(ge=0, le=MAX_ITEMS)
    date_from: date | None = None
    date_to: date | None = None
    currency: Currency | None = None
    price_basis: Literal["hammer", "realized_including_premium"] | None = None
    low: JSONDecimal | None = Field(default=None, ge=0)
    median: JSONDecimal | None = Field(default=None, ge=0)
    high: JSONDecimal | None = Field(default=None, ge=0)
    confidence: Confidence
    limitations: list[BoundedWarning] = Field(default_factory=list, max_length=10)
    supporting_source_ids: list[BoundedSourceURL] = Field(default_factory=list, max_length=MAX_ITEMS)


class TruncationMetadata(StrictSpecialistModel):
    truncated: bool
    omitted_items: int = Field(default=0, ge=0)


class SpecialistResult(StrictSpecialistModel):
    schema_version: Literal[1] = 1
    capability: SpecialistCapability
    outcome: SpecialistOutcome
    items: list[SpecialistEvidence] = Field(default_factory=list, max_length=MAX_ITEMS)
    trend: PriceTrendSummary | None = None
    provider_attempts: list[ProviderAttempt] = Field(default_factory=list, max_length=MAX_ITEMS)
    warnings: list[BoundedWarning] = Field(default_factory=list, max_length=10)
    truncation: TruncationMetadata = TruncationMetadata(truncated=False)

    @model_validator(mode="after")
    def validate_envelope(self) -> SpecialistResult:
        expected_kind = _CAPABILITY_KINDS[self.capability]
        if any(item.kind != expected_kind for item in self.items):
            raise ValueError("evidence item kind does not match capability")
        if self.capability != "price_trends" and self.trend is not None:
            raise ValueError("trend is only valid for price_trends")
        # A failed source must never read as "nothing matched".
        if self.outcome in {"complete", "partial"} and not self.items:
            raise ValueError("complete and partial outcomes require evidence")
        if self.outcome in {"no_match", "unavailable"} and self.items:
            raise ValueError("no_match and unavailable outcomes cannot contain evidence")
        degraded = any(attempt.status in _DEGRADED_PROVIDER_STATUSES for attempt in self.provider_attempts)
        if self.outcome == "unavailable" and not degraded:
            raise ValueError("unavailable requires a degraded provider attempt")
        if self.outcome == "no_match" and degraded:
            raise ValueError("no_match cannot follow a degraded provider attempt")
        return self


_CAPABILITY_KINDS: dict[str, str] = {
    "market_search": "dealer_listing",
    "auction_search": "auction_lot",
    "price_trends": "sale_observation",
    "similar_lots": "similar_lot",
}


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


def _parse_verification(candidate: Mapping[str, Any]) -> tuple[Confidence, VerificationState]:
    verification_state = str(candidate.get("verificationState") or "verified").strip().lower()
    confidence = str(candidate.get("confidence") or "high").strip().lower()
    if verification_state not in {"verified", "partial"} or confidence not in {"high", "medium", "low"}:
        raise ValueError("candidate verification metadata is invalid")
    return confidence, verification_state  # type: ignore[return-value]


def _parse_image_url(candidate: Mapping[str, Any]) -> str | None:
    raw = str(candidate.get("imageUrl") or candidate.get("image_url") or "").strip()
    if not raw:
        return None
    try:
        return _validate_source_url(raw)
    except ValueError:
        return None


def _parse_candidate_references(candidate: Mapping[str, Any]) -> list[CandidateReference]:
    raw = candidate.get("candidateReferences") or candidate.get("candidate_references") or []
    if not isinstance(raw, Sequence) or isinstance(raw, (str, bytes)):
        return []
    references: list[CandidateReference] = []
    for entry in raw[:5]:
        if not isinstance(entry, Mapping):
            continue
        catalog = _clean_optional_text(entry.get("catalog"), maximum=64)
        number = _clean_optional_text(entry.get("number"), maximum=64)
        if not catalog or not number:
            continue
        uri = _clean_optional_text(entry.get("uri"), maximum=2048)
        try:
            references.append(
                CandidateReference(
                    catalog=catalog,
                    number=number,
                    volume=_clean_optional_text(entry.get("volume"), maximum=64),
                    uri=uri or None,
                )
            )
        except ValueError:
            continue
    return references


def _shared_evidence_values(candidate: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "description": _clean_optional_text(candidate.get("description"), maximum=500),
        "image_url": _parse_image_url(candidate),
        "candidate_references": _parse_candidate_references(candidate),
    }


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
    confidence, verification_state = _parse_verification(candidate)

    price_value = candidate.get("listed_price")
    if price_value is None:
        price_value = candidate.get("estPrice") or candidate.get("price")
    raw_availability = str(candidate.get("availability") or "").strip().lower()
    availability = {
        "available": "available",
        "in stock": "available",
        "sold": "sold",
        "sold out": "sold",
        "unknown": "unknown",
    }.get(raw_availability)

    return DealerListing(
        kind="dealer_listing",
        source_url=source_url,
        canonical_source_id=canonical_source_identity(source_url),
        provider=provider,
        observed_at=observed_at,
        confidence=confidence,
        verification_state=verification_state,
        title=title,
        dealer_name=_clean_optional_text(candidate.get("sourceName") or candidate.get("dealer_name")),
        listed_price=_parse_decimal(price_value),
        currency=_parse_currency(candidate.get("currency") or price_value),
        availability=availability,
        ruler=_clean_optional_text(candidate.get("ruler")),
        denomination=_clean_optional_text(candidate.get("denomination")),
        era=_clean_optional_text(candidate.get("era")),
        material=_clean_optional_text(candidate.get("material")),
        **_shared_evidence_values(candidate),
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
    confidence, verification_state = _parse_verification(candidate)

    return AuctionLot(
        kind="auction_lot",
        source_url=source_url,
        canonical_source_id=canonical_source_identity(source_url),
        provider=provider,
        observed_at=observed_at,
        confidence=confidence,
        verification_state=verification_state,
        title=title,
        auction_house=_clean_optional_text(candidate.get("auctionHouse") or candidate.get("auction_house")),
        sale_name=_clean_optional_text(candidate.get("saleName") or candidate.get("sale_name")),
        lot_number=_clean_optional_text(candidate.get("lotNumber") or candidate.get("lot_number"), maximum=100),
        sale_date=_parse_sale_date(candidate.get("saleDate") or candidate.get("sale_date")),
        estimate=_parse_decimal(candidate.get("estimate")),
        current_bid=_parse_decimal(candidate.get("currentBid") or candidate.get("current_bid")),
        currency=_parse_currency(candidate.get("currency")),
        lot_status=_clean_optional_text(candidate.get("lotStatus") or candidate.get("lot_status"), maximum=64),
        ruler=_clean_optional_text(candidate.get("ruler")),
        denomination=_clean_optional_text(candidate.get("denomination")),
        era=_clean_optional_text(candidate.get("era")),
        material=_clean_optional_text(candidate.get("material")),
        **_shared_evidence_values(candidate),
    )


def adapt_sale_observation(
    candidate: Mapping[str, Any],
    *,
    provider: str,
    observed_at: datetime,
    allowed_hosts: frozenset[str] | None = None,
) -> SaleObservation:
    """Normalize one completed-sale observation without conversion or inference."""
    source_url = validate_registered_source_url(
        provider,
        str(candidate.get("url") or candidate.get("sourceUrl") or candidate.get("source_url") or "").strip(),
        allowed_hosts,
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
    confidence, verification_state = _parse_verification(candidate)

    return SaleObservation(
        kind="sale_observation",
        source_url=source_url,
        canonical_source_id=canonical_source_identity(source_url),
        provider=provider,
        observed_at=observed_at,
        confidence=confidence,
        verification_state=verification_state,
        title=title,
        sale_date=sale_date,
        amount=amount,
        currency=currency,
        price_basis=price_basis,
        **_shared_evidence_values(candidate),
    )


def _merge_duplicate(existing: EvidenceItem, candidate: EvidenceItem) -> EvidenceItem:
    """Keep the richer of two observations of the same source."""
    updates = {
        field: getattr(candidate, field)
        for field in type(existing).model_fields
        if getattr(existing, field, None) in (None, [], "") and getattr(candidate, field, None) not in (None, [], "")
    }
    return existing.model_copy(update=updates) if updates else existing


def _deduplicate_items(items: Sequence[EvidenceItem]) -> tuple[list[EvidenceItem], list[str]]:
    merged: dict[str, EvidenceItem] = {}
    warnings: list[str] = []
    for item in items:
        existing = merged.get(item.canonical_source_id)
        if existing is None:
            merged[item.canonical_source_id] = item
            continue
        merged[item.canonical_source_id] = _merge_duplicate(existing, item)
        warnings.append("Conflicting duplicate observations of one source were merged.")
    return list(merged.values()), warnings[:1]


def _sale_strength(item: SaleObservation) -> tuple[int, int]:
    return (
        1 if item.verification_state == "verified" else 0,
        {"high": 2, "medium": 1, "low": 0}[item.confidence],
    )


def _deduplicate_sales(items: Sequence[SaleObservation]) -> tuple[list[SaleObservation], list[str]]:
    merged: dict[tuple[str, date, Decimal, str, str], SaleObservation] = {}
    warnings: list[str] = []
    for item in items:
        key = (item.canonical_source_id, item.sale_date, item.amount, item.currency, item.price_basis)
        existing = merged.get(key)
        if existing is None:
            merged[key] = item
            continue
        warnings.append("Conflicting duplicate completed-sale observations were merged.")
        if _sale_strength(item) > _sale_strength(existing):
            merged[key] = item
    return list(merged.values()), warnings[:1]


def _build_price_trend(items: Sequence[SaleObservation]) -> PriceTrendSummary:
    if not items:
        return PriceTrendSummary(
            state="unknown",
            sample_size=0,
            confidence="low",
            limitations=["No completed-sale observations were available."],
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
        limitations.append("Fewer than three completed sales were available.")
    if not enough_dates:
        limitations.append("Completed sales covered fewer than two distinct sale dates.")
    if not enough_coverage:
        limitations.append("Completed sales covered less than 30 days.")
    if not all_verified:
        limitations.append("At least one completed-sale observation was only partially verified.")

    state: Literal["rising", "stable", "declining", "unknown"] = "unknown"
    aggregate: dict[str, Any] = {"currency": None, "price_basis": None, "low": None, "median": None, "high": None}
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
        "high" if state != "unknown" else "medium" if comparable and len(items) >= 3 else "low"
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
    items: Sequence[EvidenceItem],
    provider_attempts: Sequence[ProviderAttempt],
    warnings: Sequence[str],
    omitted_items: int,
    trend: PriceTrendSummary | None = None,
) -> SpecialistResult:
    return SpecialistResult(
        capability=capability,
        outcome=outcome,
        items=list(items),
        trend=trend,
        provider_attempts=list(provider_attempts),
        warnings=list(warnings)[:10],
        truncation=TruncationMetadata(truncated=omitted_items > 0, omitted_items=omitted_items),
    )


def _outcome_for(items: Sequence[Any], degraded: bool) -> SpecialistOutcome:
    if items:
        return "partial" if degraded else "complete"
    return "unavailable" if degraded else "no_match"


_SIMILARITY_STOPWORDS = frozenset(
    {
        "active", "auction", "coin", "coins", "find", "for", "lot", "lots",
        "my", "of", "similar", "the", "to", "with",
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
                image_url=item.image_url,
                candidate_references=list(item.candidate_references),
                similarity_score=min(1.0, 0.35 + (0.1 * len(matched))),
                matched_attributes=matched,
                material_differences=[],
            )
        )

    projected.sort(key=lambda item: (-item.similarity_score, item.canonical_source_id))
    projected_count = len(projected)
    projected = projected[:MAX_ITEMS]
    degraded = any(
        attempt.status in _DEGRADED_PROVIDER_STATUSES for attempt in auction_result.provider_attempts
    )
    attempts = [
        attempt.model_copy(
            update={
                "status": attempt.status
                if attempt.status in _DEGRADED_PROVIDER_STATUSES
                else ("success" if projected else "no_match"),
                "accepted_items": 0 if attempt.status in _DEGRADED_PROVIDER_STATUSES else len(projected),
            }
        )
        for attempt in auction_result.provider_attempts
    ]
    return _finalize_result(
        capability="similar_lots",
        outcome=_outcome_for(projected, degraded),
        items=projected,
        provider_attempts=attempts,
        warnings=auction_result.warnings,
        omitted_items=max(0, projected_count - len(projected)),
    )


async def _collect_provider_items(
    *,
    capability: str,
    provider_runner: ProviderRunner,
    query: SpecialistQuery,
    timestamp: datetime,
    adapt: Callable[[Mapping[str, Any]], Any],
    keep: Callable[[Any], bool],
    cancellation_check: CancellationCheck | None,
) -> tuple[list[Any], ProviderAttempt, list[str]]:
    """Run one configured source and normalize what it returned."""
    warnings: list[str] = []
    try:
        await raise_if_cancelled(cancellation_check)
        raw_candidates = await provider_runner.run(query.query, query.limit)
        await raise_if_cancelled(cancellation_check)
        if isinstance(raw_candidates, str | bytes) or not isinstance(raw_candidates, Sequence):
            raise ProviderMalformedError
        normalized: list[Any] = []
        invalid_count = 0
        for candidate in raw_candidates:
            if not isinstance(candidate, Mapping):
                invalid_count += 1
                continue
            try:
                item = adapt(candidate)
            except (ValueError, TypeError) as exc:
                invalid_count += 1
                logger.info(
                    "Specialist candidate rejected capability=%s provider=%s reason=%s",
                    capability,
                    provider_runner.provider,
                    exc,
                )
                continue
            if keep(item):
                normalized.append(item)
        if invalid_count and not normalized:
            raise ProviderMalformedError
        if invalid_count:
            warnings.append(_SAFE_PROVIDER_WARNINGS["malformed"])
        status: ProviderStatus = "success" if normalized else "no_match"
        return (
            normalized,
            ProviderAttempt(
                provider=provider_runner.provider,
                status=status,
                observed_at=timestamp,
                accepted_items=min(len(normalized), MAX_ITEMS),
            ),
            warnings,
        )
    except (TimeoutError, asyncio.TimeoutError, httpx.TimeoutException) as exc:
        status, error = "timeout", exc
    except ProviderUnavailableError as exc:
        status, error = "unavailable", exc
    except (ProviderMalformedError, ValueError) as exc:
        status, error = "malformed", exc
    except _EXTERNAL_PROVIDER_ERRORS as exc:
        status, error = "failure", exc
    _log_degraded_provider(capability, provider_runner.provider, status, error)
    return (
        [],
        ProviderAttempt(
            provider=provider_runner.provider,
            status=status,
            observed_at=timestamp,
            accepted_items=0,
            warning_code=_WARNING_CODES[status],
        ),
        [_SAFE_PROVIDER_WARNINGS[status]],
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
    accepted: list[EvidenceItem] = []
    warnings: list[str] = []

    for provider_runner in provider_runners:
        def adapt(candidate: Mapping[str, Any], runner: ProviderRunner = provider_runner) -> EvidenceItem:
            adapter = adapt_dealer_candidate if capability == "market_search" else adapt_auction_candidate
            return adapter(
                candidate,
                provider=runner.provider,
                observed_at=timestamp,
                allowed_hosts=runner.allowed_hosts,
            )

        items, attempt, provider_warnings = await _collect_provider_items(
            capability=capability,
            provider_runner=provider_runner,
            query=parsed_query,
            timestamp=timestamp,
            adapt=adapt,
            keep=lambda item: not (isinstance(item, DealerListing) and item.availability == "sold"),
            cancellation_check=cancellation_check,
        )
        accepted.extend(items)
        attempts.append(attempt)
        warnings.extend(provider_warnings)

    deduplicated, duplicate_warnings = _deduplicate_items(accepted)
    warnings.extend(duplicate_warnings)
    limit = min(parsed_query.limit, MAX_ITEMS)
    items = deduplicated[:limit]
    degraded = any(attempt.status in _DEGRADED_PROVIDER_STATUSES for attempt in attempts)
    return _finalize_result(
        capability=capability,
        outcome=_outcome_for(items, degraded),
        items=items,
        provider_attempts=attempts,
        warnings=warnings,
        omitted_items=max(0, len(deduplicated) - limit),
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
        def adapt(candidate: Mapping[str, Any], runner: ProviderRunner = provider_runner) -> SaleObservation:
            return adapt_sale_observation(
                candidate,
                provider=runner.provider,
                observed_at=timestamp,
                allowed_hosts=runner.allowed_hosts,
            )

        items, attempt, provider_warnings = await _collect_provider_items(
            capability="price_trends",
            provider_runner=provider_runner,
            query=parsed_query,
            timestamp=timestamp,
            adapt=adapt,
            keep=lambda _item: True,
            cancellation_check=cancellation_check,
        )
        accepted.extend(items)
        attempts.append(attempt)
        warnings.extend(provider_warnings)

    deduplicated, duplicate_warnings = _deduplicate_sales(accepted)
    warnings.extend(duplicate_warnings)
    limit = min(parsed_query.limit, MAX_ITEMS)
    items = deduplicated[:limit]
    degraded = any(attempt.status in _DEGRADED_PROVIDER_STATUSES for attempt in attempts)
    return _finalize_result(
        capability="price_trends",
        outcome=_outcome_for(items, degraded),
        items=items,
        trend=_build_price_trend(items),
        provider_attempts=attempts,
        warnings=warnings,
        omitted_items=max(0, len(deduplicated) - limit),
    )


def specialist_result_json(result: SpecialistResult) -> str:
    """Canonical JSON for a specialist result."""
    return json.dumps(result.model_dump(mode="json"), ensure_ascii=False, separators=(",", ":"), sort_keys=True)
