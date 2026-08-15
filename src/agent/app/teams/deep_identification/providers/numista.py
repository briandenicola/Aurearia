"""Numista provider node (T060) — automated, calls provider_tools.py only.

Query text is built deterministically from `quick_evidence`/`notes` (never
chosen freely by an LLM), matching FR "no arbitrary URL/tool selection".
"""

import logging

from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.merge import validate_citations
from app.tools.provider_tools import ProviderToolError, ProviderToolsClient

logger = logging.getLogger(__name__)

_DEFAULT_QUERY = "unidentified ancient coin"


def _build_query(quick_evidence: QuickEvidence | None, notes: str) -> str:
    if quick_evidence:
        if quick_evidence.numista_query:
            return quick_evidence.numista_query
        if quick_evidence.label_text:
            return quick_evidence.label_text
    if notes:
        return notes[:200]
    return _DEFAULT_QUERY


async def run(
    catalog_entry: DeepProviderCatalogEntry,
    tools: ProviderToolsClient,
    quick_evidence: QuickEvidence | None,
    notes: str,
) -> ProviderEvidence:
    query = _build_query(quick_evidence, notes)
    try:
        result = await tools.numista_search(query, limit=5)
    except ProviderToolError:
        return ProviderEvidence(
            provider="numista", status="failed", automatable=True, error_kind="upstream", call_count=1,
        )

    status = result.get("status", "unavailable")
    attribution = str(result.get("attribution") or "Source: Numista")
    candidates = result.get("candidates") or []

    if status == "unconfigured":
        return ProviderEvidence(
            provider="numista", status="failed", automatable=True, error_kind="unconfigured", call_count=1
        )
    if status == "quota_limited":
        return ProviderEvidence(
            provider="numista", status="failed", automatable=True, error_kind="quota", call_count=1
        )
    if status == "timeout":
        return ProviderEvidence(
            provider="numista", status="timed_out", automatable=True, error_kind="timeout", call_count=1
        )
    if status == "unavailable":
        return ProviderEvidence(
            provider="numista", status="failed", automatable=True, error_kind="upstream", call_count=1
        )
    if status == "empty" or not candidates:
        return ProviderEvidence(
            provider="numista", status="no_match", automatable=True, call_count=1, attribution=attribution
        )

    top = candidates[0]
    raw_claims: list[ProviderClaim] = []
    citation = str(top.get("canonicalUrl") or "")
    if citation:
        title = str(top.get("title") or "")
        for field_name, value in (
            ("denomination", top.get("denomination")),
            ("issuer", top.get("issuer")),
            ("mint", top.get("mint")),
            ("material", top.get("material")),
        ):
            if value:
                raw_claims.append(
                    ProviderClaim(
                        field=field_name, value=str(value), confidence=0.7, citation=citation, excerpt=title[:500]
                    )
                )

    claims, dropped = validate_citations("numista", raw_claims)
    if not claims:
        error_kind = "invalid_response" if dropped else None
        status_out = "no_match" if not dropped else "failed"
        return ProviderEvidence(
            provider="numista", status=status_out, automatable=True, error_kind=error_kind,
            call_count=1, attribution=attribution,
        )

    return ProviderEvidence(
        provider="numista", status="contributed", automatable=True, confidence=0.7,
        call_count=1, attribution=attribution, claims=claims,
    )
