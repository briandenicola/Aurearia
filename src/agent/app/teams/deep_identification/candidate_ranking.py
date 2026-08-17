"""Deterministic, application-owned candidate ranking (T121/T123).

Numista/Nomisma providers previously requested `limit=5` and then took
`candidates[0]` unconditionally, discarding four already-returned
candidates with no ranking whatsoever (`providers/numista.py`,
`providers/nomisma.py`). This module scores the candidate set a provider
has **already returned** against hypothesis signals that are deliberately
excluded from query terms — chiefly reverse legend/type (RD-4, spec
FR-039) — and selects the best-scoring candidate instead of the first.

Guarantees:
  * Zero additional upstream calls, zero additional call budget — this
    operates only on results already in hand (FR-039).
  * Deterministic and reproducible: identical candidates + an identical
    hypothesis always produce an identical ordering (FR-039).
  * Ties break stably on the provider's original candidate order — an LLM
    MUST NOT choose or reorder candidates (FR-009's property extends to
    ranking, FR-039).

OCRE's ranking mechanism is separate, pre-existing, and ADR 0010 governed
(`providers/ocre.py::_legend_tokens` + `src/api/services/ocre_scoring.go`);
this module does not touch it (see T122).
"""

from typing import Any

from app.models.hypothesis import CoinHypothesis


def _tokenize(text: str) -> set[str]:
    tokens: set[str] = set()
    for word in text.split():
        token = "".join(ch for ch in word.lower() if ch.isalnum())
        if token:
            tokens.add(token)
    return tokens


def _hypothesis_signal_tokens(hypothesis: CoinHypothesis | None) -> set[str]:
    """Reverse legend/type plus other hypothesis fields that carry no query
    role (RD-4): ranking/disambiguation signal only, never a query term
    (FR-010) and never a trigger for an additional upstream call (FR-039).
    """
    if hypothesis is None:
        return set()

    tokens: set[str] = set()
    for field in (
        hypothesis.reverseInscription,
        hypothesis.coin_type,
        hypothesis.reverseDescription,
        hypothesis.mint,
        hypothesis.dateRange,
    ):
        if field is not None:
            tokens |= _tokenize(field.value)
    return tokens


def rank_candidates(
    candidates: list[dict[str, Any]],
    hypothesis: CoinHypothesis | None,
    text_fields: tuple[str, ...],
) -> list[dict[str, Any]]:
    """Return `candidates` reordered best-scoring first.

    Scores each candidate by the count of hypothesis reverse-legend/type
    (and other unused-field) tokens found in the candidate's own
    `text_fields` values. Operates only on the list already in hand — zero
    upstream calls. When there is no hypothesis signal at all, the
    original provider order is returned unchanged (nothing to rank on).
    """
    signal_tokens = _hypothesis_signal_tokens(hypothesis)
    if not candidates or not signal_tokens:
        return list(candidates)

    def _score(candidate: dict[str, Any]) -> int:
        candidate_tokens: set[str] = set()
        for field_name in text_fields:
            value = candidate.get(field_name)
            if isinstance(value, str) and value:
                candidate_tokens |= _tokenize(value)
        return len(candidate_tokens & signal_tokens)

    # `sorted` is stable: equal-score candidates keep their original
    # relative (provider-returned) order — the required tie-break rule.
    indexed = list(enumerate(candidates))
    ranked = sorted(indexed, key=lambda pair: (-_score(pair[1]), pair[0]))
    return [candidate for _, candidate in ranked]
