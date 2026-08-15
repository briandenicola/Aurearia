"""Nomisma provider node (T060) — automated, calls provider_tools.py only.

Nomisma.org's OpenRefine-compatible reconciliation service returns
authority-record matches (mint/ruler/type identifiers), not full catalog
descriptions — surfaced as a single `mint_authority` claim per top match.
"""

import logging

from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.merge import validate_citations
from app.tools.provider_tools import ProviderToolError, ProviderToolsClient

logger = logging.getLogger(__name__)

_DEFAULT_QUERY = "unidentified ancient coin"


def _build_query(quick_evidence: QuickEvidence | None, notes: str) -> str:
    if quick_evidence and quick_evidence.label_text:
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
        result = await tools.nomisma_search(query, limit=5)
    except ProviderToolError:
        return ProviderEvidence(
            provider="nomisma", status="failed", automatable=True, error_kind="upstream", call_count=1,
        )

    status = result.get("status", "unavailable")
    attribution = str(result.get("attribution") or "Data: Nomisma.org (CC BY)")
    candidates = result.get("candidates") or []

    if status == "unavailable":
        return ProviderEvidence(
            provider="nomisma", status="failed", automatable=True, error_kind="upstream", call_count=1
        )
    if status == "empty" or not candidates:
        return ProviderEvidence(
            provider="nomisma", status="no_match", automatable=True, call_count=1, attribution=attribution
        )

    # Only consider matches Nomisma itself flagged as a plausible reconciliation.
    matches = [c for c in candidates if c.get("match")] or candidates[:1]
    top = matches[0]
    citation = str(top.get("uri") or "")
    raw_claims: list[ProviderClaim] = []
    if citation and top.get("label"):
        score = top.get("score")
        confidence = float(score) if isinstance(score, (int, float)) else 0.5
        confidence = max(0.0, min(1.0, confidence))
        raw_claims.append(
            ProviderClaim(field="mint_authority", value=str(top["label"]), confidence=confidence, citation=citation)
        )

    claims, dropped = validate_citations("nomisma", raw_claims)
    if not claims:
        error_kind = "invalid_response" if dropped else None
        status_out = "no_match" if not dropped else "failed"
        return ProviderEvidence(
            provider="nomisma", status=status_out, automatable=True, error_kind=error_kind,
            call_count=1, attribution=attribution,
        )

    return ProviderEvidence(
        provider="nomisma", status="contributed", automatable=True,
        confidence=claims[0].confidence, call_count=1, attribution=attribution, claims=claims,
    )
