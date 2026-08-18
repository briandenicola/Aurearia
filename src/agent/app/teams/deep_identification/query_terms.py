"""Deterministic, application-owned provider query-term builder.

(T034/T035/T037 — contracts/vision-hypothesis.md §2, spec FR-009/FR-010/FR-011/FR-012.)

This is the ONE shared builder used by every automatable provider node
(`providers/numista.py`, `providers/nomisma.py`, `providers/ocre.py`'s
type-bearing bound slots are separate — see below). Query text stays
application-authored: no LLM may choose, rewrite, extend, or reorder it
(FR-009). This property extends to candidate ranking too
(`candidate_ranking.py`).

Precedence, first non-empty tier wins (contract §2):

  1. `quick_evidence.numista_query`
  2. `quick_evidence.label_text`
  3. hypothesis-derived terms — a **fixed** composition order (RD-4):
       `ruler + denomination` -> `ruler` -> `denomination + material`
       -> `obverseInscription`.
     Reverse type and reverse legend are **excluded from query terms
     entirely** (RD-4/FR-010) — they are ranking-only signals consumed by
     `candidate_ranking.py` instead. No second, narrower probe is issued.
  4. `notes[:200]`

When no tier yields non-empty terms, `build_query_terms` returns `""`.
Callers MUST treat that as `insufficient_query_evidence` (FR-011) and make
**zero** upstream calls — never fall back to a placeholder string.
"""

from app.models.hypothesis import CoinHypothesis
from app.models.requests import QuickEvidence

# Bound applied to the owner-notes tier (tier 4) regardless of any
# provider-specific overall bound — matches the pre-existing `notes[:200]`
# behavior every provider already had.
NOTES_MAX_LENGTH = 200


def _value(field) -> str:
    return field.value.strip() if field is not None else ""


def _hypothesis_terms(hypothesis: CoinHypothesis | None) -> str:
    """Fixed composition order (RD-4, spec FR-010). Reverse legend/type are
    never included here — they are ranking-only (FR-039).
    """
    if hypothesis is None:
        return ""

    ruler = _value(hypothesis.ruler)
    denomination = _value(hypothesis.denomination)
    material = _value(hypothesis.material)
    obverse_inscription = _value(hypothesis.obverseInscription)

    if ruler and denomination:
        return f"{ruler} {denomination}"
    if ruler:
        return ruler
    if denomination and material:
        return f"{denomination} {material}"
    if obverse_inscription:
        return obverse_inscription
    return ""


def build_query_terms(
    quick_evidence: QuickEvidence | None,
    hypothesis: CoinHypothesis | None,
    notes: str,
    *,
    max_length: int | None = None,
) -> str:
    """Return the deterministic query string for one provider call, or
    `""` when no precedence tier yields usable terms (FR-011).

    `max_length` bounds the FINAL result regardless of which tier produced
    it — e.g. Nomisma's Go client rejects anything over 200 runes
    (`nomisma_client.go::nomismaMaxQueryLength`), so `nomisma.py` passes
    `max_length=200` here rather than only bounding the notes tier.
    """
    query = ""
    if quick_evidence and quick_evidence.numista_query:
        query = quick_evidence.numista_query
    elif quick_evidence and quick_evidence.label_text:
        query = quick_evidence.label_text
    else:
        query = _hypothesis_terms(hypothesis)
        if not query and notes:
            query = notes[:NOTES_MAX_LENGTH]

    query = query.strip()
    if max_length is not None:
        query = query[:max_length].strip()
    return query
