"""Numista provider node (T060) — automated, calls provider_tools.py only.

Query text is built deterministically by `query_terms.build_query_terms`
(never chosen freely by an LLM), matching FR "no arbitrary URL/tool
selection" (FR-009). When no precedence tier yields usable terms, this node
makes ZERO upstream calls and reports `insufficient_query_evidence`
(FR-011) instead of searching for a placeholder string.
"""

import logging

from app.models.hypothesis import CoinHypothesis
from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.candidate_ranking import rank_candidates
from app.teams.deep_identification.merge import validate_citations
from app.teams.deep_identification.query_terms import build_query_terms
from app.tools.provider_tools import ProviderToolError, ProviderToolsClient

logger = logging.getLogger(__name__)

# Candidate text fields consulted by the reverse-legend/type ranker
# (T121/FR-039) — ranking only, never query input.
_RANK_TEXT_FIELDS = ("title", "denomination", "issuer", "mint", "material")


def _build_query(
    quick_evidence: QuickEvidence | None, notes: str, hypothesis: CoinHypothesis | None = None
) -> str:
    return build_query_terms(quick_evidence, hypothesis, notes)


async def run(
    catalog_entry: DeepProviderCatalogEntry,
    tools: ProviderToolsClient,
    quick_evidence: QuickEvidence | None,
    notes: str,
    hypothesis: CoinHypothesis | None = None,
) -> ProviderEvidence:
    query = _build_query(quick_evidence, notes, hypothesis)
    if not query:
        # No precedence tier yielded usable terms (FR-011) — zero upstream
        # calls, never a search for a placeholder string.
        return ProviderEvidence(
            provider="numista", status="no_match", automatable=True,
            error_kind="insufficient_query_evidence", call_count=0,
        )
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

    # T121/FR-039: rank the already-returned candidates against hypothesis
    # reverse-legend/type signals before picking one — zero additional
    # upstream calls, no LLM choice, ties keep the provider's own order.
    ranked = rank_candidates(candidates, hypothesis, _RANK_TEXT_FIELDS)
    top = ranked[0]
    raw_claims: list[ProviderClaim] = []
    citation = str(top.get("canonicalUrl") or "")
    if citation:
        title = str(top.get("title") or "")
        for field_name, value in (
            ("denomination", top.get("denomination")),
            ("ruler", top.get("issuer")),
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
