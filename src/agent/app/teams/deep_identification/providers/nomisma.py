"""Nomisma provider node (T060) — automated, calls provider_tools.py only.

Nomisma.org's OpenRefine-compatible reconciliation service returns
authority-record matches (mint/ruler/type identifiers), not full catalog
descriptions — surfaced as a single `mint_authority` claim per top match.
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

# Go's Nomisma client rejects any query over this many runes as a
# client-side invalid request (nomisma_client.go::nomismaMaxQueryLength).
# `label_text` is bounded to 2000 runes on the wire (QuickEvidence), so
# passing it through unbounded here reliably produced an over-length query
# for every slabbed coin (label text routinely exceeds 200 runes),
# misreported as an upstream Nomisma outage — this bug is ours, not
# Nomisma's, and must never reach the client at all.
_MAX_QUERY_LENGTH = 200

# Candidate text fields consulted by the reverse-legend/type ranker
# (T121/FR-039) — ranking only, never query input.
_RANK_TEXT_FIELDS = ("label",)


def _build_query(
    quick_evidence: QuickEvidence | None, notes: str, hypothesis: CoinHypothesis | None = None
) -> str:
    return build_query_terms(quick_evidence, hypothesis, notes, max_length=_MAX_QUERY_LENGTH)


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
            provider="nomisma", status="no_match", automatable=True,
            error_kind="insufficient_query_evidence", call_count=0,
        )
    try:
        result = await tools.nomisma_search(query, limit=5)
    except ProviderToolError:
        return ProviderEvidence(
            provider="nomisma", status="failed", automatable=True, error_kind="upstream", call_count=1,
        )

    status = result.get("status", "unavailable")
    attribution = str(result.get("attribution") or "Data: Nomisma.org (CC BY)")
    candidates = result.get("candidates") or []

    if status == "invalid_request":
        # Task G: an over-length/malformed query is OUR bug (client-side),
        # never an upstream Nomisma outage — Go never even issued the HTTP
        # call (call_count=0). `_build_query` above bounds every query to
        # `_MAX_QUERY_LENGTH`, so this should be unreachable in practice;
        # kept as a defensive backstop rather than silently reusing the
        # "unavailable"/upstream-failed status this defect used to produce.
        logger.warning("[deep_identification.nomisma] query rejected as invalid by Go client; this indicates a bug")
        return ProviderEvidence(provider="nomisma", status="no_match", automatable=True, call_count=0)
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
    # T121/FR-039: rank the already-returned matches against hypothesis
    # reverse-legend/type signals before picking one — zero additional
    # upstream calls, no LLM choice, ties keep the provider's own order.
    ranked = rank_candidates(matches, hypothesis, _RANK_TEXT_FIELDS)
    top = ranked[0]
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
