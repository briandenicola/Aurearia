"""OCRE provider node (Feature 345) — automated Deep Analysis provider.

Online Coins of the Roman Empire (numismatics.org) coin-type candidates are
sourced through the Go-hosted `ocre_search` internal tool, which is the only
OCRE/Nomisma-SPARQL HTTP boundary. This node never issues SPARQL, never
scrapes, and never fetches an image — it decodes bound query signals from the
quick-lookup evidence, calls the tool, and maps the typed response onto the
existing ProviderEvidence/ProviderClaim contract.

Behavioral guarantees:
  * Flag off (`catalog_entry.automatable is False`) → a trivial `not_automated`
    row with ZERO tool calls (FR-004/FR-016).
  * No decodable Roman-Imperial signal → `no_match` with ZERO tool calls
    (US1-AC5) — no SPARQL is emitted for a query that binds nothing.
  * Every transport failure → a typed `failed`/`timed_out` row; the node never
    raises (FR-015).
  * Every claim citation is host-validated (numismatics.org) before emission.

T122 (RD-4/FR-039): `_legend_tokens` is ADR 0010's scoring-only-signal
source. `src/api/services/ocre_scoring.go` (`ocreLegendMatches`,
`ocreLegendBonusPer`, `ocreLegendBonusMax`, the base-score weights, the
`[0,1]` clamp, `sort.SliceStable`) is that ADR's deterministic contract and
is NOT touched by this module — only the *source* of legend tokens is
widened here, to also draw from the hypothesis when quick evidence carries
no `label_text`.
"""

import logging

from app.models.hypothesis import CoinHypothesis
from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.merge import validate_citations
from app.tools.provider_tools import ProviderToolError, ProviderToolsClient

logger = logging.getLogger(__name__)

# Exact, fixed attribution string (contract §6 / FR-019). Must match the Go
# handler's ocreAttribution byte-for-byte (em dash U+2014).
OCRE_ATTRIBUTION = (
    "Coin type data: Online Coins of the Roman Empire (OCRE), "
    "American Numismatic Society — ODbL 1.0."
)

# Quick-evidence coin_fields keys that carry type-bearing signals. All values
# are re-normalized/re-validated into Nomisma id slugs Go-side; free text that
# fails validation is dropped there, never interpolated into SPARQL.
_RULER_KEYS = ("ruler", "issuer", "authority")
_DENOMINATION_KEYS = ("denomination", "denom")
_MINT_KEYS = ("mint", "mintmark")
_MATERIAL_KEYS = ("material", "metal")
_OCRE_ID_KEYS = ("ocre_id", "ocreId", "coin_type", "coinType")


def _first(fields: dict[str, str], keys: tuple[str, ...]) -> str:
    for key in keys:
        value = fields.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""


def _tokenize_legend(text: str, tokens: list[str], seen: set[str]) -> None:
    for raw in text.split():
        token = "".join(ch for ch in raw.lower() if ch.isalnum())
        if token and token not in seen:
            seen.add(token)
            tokens.append(token)
        if len(tokens) >= 12:
            return


def _legend_tokens(
    quick_evidence: QuickEvidence | None, hypothesis: CoinHypothesis | None = None
) -> list[str]:
    """Alphanumeric legend tokens are scoring-only signals (never SPARQL).

    T122/RD-4: `quick_evidence.label_text` still wins when present (it is
    already the router's highest-precedence evidence). When it is absent —
    the entire Maximinus no-quick-evidence scenario — fall back to the
    hypothesis's obverse inscription, reverse inscription, and coin type
    (the reverse-type analogue in the hypothesis vocabulary), which is
    exactly the reverse-legend/type signal RD-4/FR-039 designates for
    ranking/scoring. Same normalization, dedup, and 12-token cap as before.
    """
    tokens: list[str] = []
    seen: set[str] = set()
    if quick_evidence is not None and quick_evidence.label_text:
        _tokenize_legend(quick_evidence.label_text, tokens, seen)
        return tokens
    if hypothesis is not None:
        for field in (hypothesis.obverseInscription, hypothesis.reverseInscription, hypothesis.coin_type):
            if field is not None and len(tokens) < 12:
                _tokenize_legend(field.value, tokens, seen)
    return tokens


def _not_automated(catalog_entry: DeepProviderCatalogEntry) -> ProviderEvidence:
    return ProviderEvidence(
        provider="ocre",
        status="not_automated",
        automatable=False,
        call_count=0,
        link_out=catalog_entry.link_out or "",
        attribution="",
    )


async def run(
    catalog_entry: DeepProviderCatalogEntry,
    tools: ProviderToolsClient | None,
    quick_evidence: QuickEvidence | None,
    notes: str,
    hypothesis: CoinHypothesis | None = None,
) -> ProviderEvidence:
    # (1) Flag-off short circuit — never a tool call (FR-004/FR-016).
    if not catalog_entry.automatable:
        return _not_automated(catalog_entry)

    # (2) Decode bound query signals from the quick-lookup evidence.
    fields = quick_evidence.coin_fields if quick_evidence else {}
    ruler = _first(fields, _RULER_KEYS)
    denomination = _first(fields, _DENOMINATION_KEYS)
    mint = _first(fields, _MINT_KEYS)
    material = _first(fields, _MATERIAL_KEYS)
    ocre_id = _first(fields, _OCRE_ID_KEYS)
    legend_tokens = _legend_tokens(quick_evidence, hypothesis)

    # (3) No type-bearing signal decodes → no_match with ZERO tool calls.
    if not (ruler or denomination or mint or ocre_id):
        return ProviderEvidence(
            provider="ocre", status="no_match", automatable=True, call_count=0, attribution=OCRE_ATTRIBUTION
        )

    # (4) Call the Go internal tool (the only OCRE HTTP boundary).
    if tools is None:
        return ProviderEvidence(
            provider="ocre", status="failed", automatable=True, error_kind="unconfigured", call_count=0
        )
    try:
        result = await tools.ocre_search(
            ruler=ruler,
            denomination=denomination,
            mint=mint,
            material=material,
            legend_tokens=legend_tokens,
            ocre_id=ocre_id,
            limit=5,
        )
    except ProviderToolError:
        # (5) Transport failure → typed row, never raises (FR-015).
        return ProviderEvidence(
            provider="ocre", status="failed", automatable=True, error_kind="upstream", call_count=1
        )

    status = str(result.get("status") or "unavailable")
    attribution = str(result.get("attribution") or OCRE_ATTRIBUTION)
    candidates = result.get("candidates") or []

    if status == "quota_limited":
        return ProviderEvidence(
            provider="ocre", status="no_match", automatable=True, error_kind="quota",
            call_count=1, attribution=attribution,
        )
    if status == "timeout":
        return ProviderEvidence(
            provider="ocre", status="timed_out", automatable=True, error_kind="timeout", call_count=1
        )
    if status == "invalid_response":
        return ProviderEvidence(
            provider="ocre", status="failed", automatable=True, error_kind="invalid_response", call_count=1
        )
    if status in ("unavailable", "cancelled"):
        return ProviderEvidence(
            provider="ocre", status="failed", automatable=True, error_kind="upstream", call_count=1
        )
    if status == "empty" or not candidates:
        return ProviderEvidence(
            provider="ocre", status="no_match", automatable=True, call_count=1, attribution=attribution
        )

    # (4b) Map each surviving candidate to a coin_type claim. Multiple claims
    # on the same field preserve ambiguity (FR-013), never collapsed by trust.
    raw_claims: list[ProviderClaim] = []
    for candidate in candidates:
        type_uri = str(candidate.get("type_uri") or "")
        label = str(candidate.get("label") or "")
        if not type_uri or not label:
            continue
        raw_conf = candidate.get("confidence")
        confidence = float(raw_conf) if isinstance(raw_conf, (int, float)) else 0.5
        confidence = max(0.0, min(1.0, confidence))
        raw_claims.append(
            ProviderClaim(
                field="coin_type",
                value=label,
                confidence=confidence,
                citation=type_uri,
                excerpt=str(candidate.get("explanation") or "")[:500],
            )
        )

    claims, dropped = validate_citations("ocre", raw_claims)
    if not claims:
        # A dropped-but-present set is a malformed upstream response
        # (invalid_response telemetry); an empty set is simply no_match —
        # never a fabricated claim.
        error_kind = "invalid_response" if dropped else None
        status_out = "failed" if dropped else "no_match"
        return ProviderEvidence(
            provider="ocre", status=status_out, automatable=True, error_kind=error_kind,
            call_count=1, attribution=attribution,
        )

    return ProviderEvidence(
        provider="ocre",
        status="contributed",
        automatable=True,
        confidence=max(claim.confidence for claim in claims),
        call_count=1,
        attribution=attribution,
        claims=claims,
    )
