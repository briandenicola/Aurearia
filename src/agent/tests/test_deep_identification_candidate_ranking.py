"""Candidate ranking tests (T121/T122/T123).

Covers the shared, deterministic `candidate_ranking.rank_candidates` used by
Numista/Nomisma (T121, new behavior), and the OCRE legend-token widening
(T122) that draws scoring-only tokens from the hypothesis when quick
evidence carries no `label_text`. Also asserts `ocre_scoring.go` is left
untouched by this diff (ADR 0010, RD-4).
"""

import asyncio
from pathlib import Path

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.teams.deep_identification.candidate_ranking import rank_candidates
from app.teams.deep_identification.providers import numista, ocre


def _field(value: str) -> HypothesisField:
    return HypothesisField(value=value, confidence=0.8)


def _entry(provider: str) -> DeepProviderCatalogEntry:
    return DeepProviderCatalogEntry(provider=provider, automatable=True, call_budget=3)


# --- T123: ranking determinism / reproducibility / no-op on empty signal ---


def test_ranking_is_deterministic_for_identical_inputs():
    candidates = [
        {"title": "Denarius, laureate head right"},
        {"title": "Denarius, Pax standing left holding branch"},
        {"title": "Denarius, generic type"},
    ]
    hypothesis = CoinHypothesis(
        reverseInscription=_field("PAX AVGVSTI"),
        reverseDescription=_field("Pax standing left holding branch and sceptre"),
    )

    first = rank_candidates(candidates, hypothesis, ("title",))
    second = rank_candidates(candidates, hypothesis, ("title",))

    assert first == second


def test_ranking_never_triggers_an_upstream_call():
    """Ranking operates only on results already in hand — zero additional
    upstream calls versus the unranked path (FR-039).
    """
    tools_calls: list[str] = []

    class Tools:
        async def numista_search(self, query, limit=5):
            tools_calls.append(query)
            return {
                "status": "ok",
                "attribution": "Source: Numista",
                "candidates": [
                    {"canonicalUrl": "https://en.numista.com/catalogue/pieces1.html",
                     "title": "Denarius, generic", "denomination": "Denarius", "issuer": "Maximinus I"},
                    {"canonicalUrl": "https://en.numista.com/catalogue/pieces2.html",
                     "title": "Denarius, Pax standing left", "denomination": "Denarius", "issuer": "Maximinus I"},
                ],
            }

    hypothesis = CoinHypothesis(reverseDescription=_field("Pax standing left"))
    result = asyncio.run(
        numista.run(
            _entry("numista"), Tools(), QuickEvidence(numista_query="Maximinus denarius"),
            notes="", hypothesis=hypothesis,
        )
    )

    assert len(tools_calls) == 1  # exactly the one search call — ranking added none
    assert result.call_count == 1


def test_empty_hypothesis_leaves_ordering_at_provider_original_order():
    candidates = [{"title": "first"}, {"title": "second"}, {"title": "third"}]

    ranked = rank_candidates(candidates, None, ("title",))

    assert ranked == candidates

    ranked_empty_hypothesis = rank_candidates(candidates, CoinHypothesis(), ("title",))
    assert ranked_empty_hypothesis == candidates


def test_reverse_signal_promotes_the_matching_candidate_over_first_position():
    candidates = [
        {"title": "Denarius, unrelated laureate bust"},
        {"title": "Denarius, Pax standing left holding branch and sceptre"},
    ]
    hypothesis = CoinHypothesis(reverseDescription=_field("Pax standing left holding branch and sceptre"))

    ranked = rank_candidates(candidates, hypothesis, ("title",))

    assert ranked[0] is candidates[1]  # the reverse-matching candidate now ranks first


def test_ties_break_stably_on_provider_original_order():
    candidates = [{"title": "alpha"}, {"title": "beta"}, {"title": "gamma"}]
    # No candidate text overlaps the hypothesis signal tokens at all, but the
    # hypothesis is non-empty — every score is 0, so order must be unchanged.
    hypothesis = CoinHypothesis(reverseInscription=_field("ZZZNOTPRESENT"))

    ranked = rank_candidates(candidates, hypothesis, ("title",))

    assert ranked == candidates


# --- T122: OCRE legend-token widening from the hypothesis ---


def test_ocre_legend_tokens_draw_from_hypothesis_when_quick_evidence_absent():
    tools_calls: list[dict] = []

    class Tools:
        async def ocre_search(self, **kwargs):
            tools_calls.append(kwargs)
            return {"status": "empty", "candidates": []}

    hypothesis = CoinHypothesis(
        obverseInscription=_field("IMP MAXIMINVS PIVS AVG"),
        reverseInscription=_field("PAX AVGVSTI"),
        coin_type=_field("Pax standing left"),
    )
    quick_evidence = QuickEvidence(coin_fields={"ruler": "Maximinus I"})  # no label_text

    asyncio.run(ocre.run(_entry("ocre"), Tools(), quick_evidence, notes="", hypothesis=hypothesis))

    assert len(tools_calls) == 1
    tokens = tools_calls[0]["legend_tokens"]
    assert tokens  # non-empty — the Maximinus defect this task fixes
    joined = " ".join(tokens)
    assert "maximinvs" in joined
    assert "pax" in joined


def test_ocre_legend_tokens_prefer_quick_evidence_label_text_when_present():
    tools_calls: list[dict] = []

    class Tools:
        async def ocre_search(self, **kwargs):
            tools_calls.append(kwargs)
            return {"status": "empty", "candidates": []}

    hypothesis = CoinHypothesis(reverseInscription=_field("HYPOTHESIS ONLY TOKEN"))
    quick_evidence = QuickEvidence(label_text="LABEL TEXT WINS", coin_fields={"ruler": "Maximinus I"})

    asyncio.run(ocre.run(_entry("ocre"), Tools(), quick_evidence, notes="", hypothesis=hypothesis))

    tokens = tools_calls[0]["legend_tokens"]
    joined = " ".join(tokens)
    assert "label" in joined or "text" in joined
    assert "hypothesis" not in joined


def test_ocre_legend_tokens_respect_the_twelve_token_cap():
    hypothesis = CoinHypothesis(
        obverseInscription=_field(" ".join(f"tok{i}" for i in range(10))),
        reverseInscription=_field(" ".join(f"rev{i}" for i in range(10))),
    )

    tokens = ocre._legend_tokens(None, hypothesis)

    assert len(tokens) <= 12


def test_ocre_scoring_go_is_untouched_by_this_diff():
    """T122/RD-4/ADR-0010: the scoring math itself must not be modified by
    this batch. This is a structural guard, not a behavioral one — it
    confirms the file's ADR-anchored symbols are all still present exactly
    as documented, i.e. nothing here removed or renamed them.
    """
    repo_root = Path(__file__).resolve().parents[3]
    scoring_path = repo_root / "src" / "api" / "services" / "ocre_scoring.go"
    assert scoring_path.exists()
    content = scoring_path.read_text(encoding="utf-8")
    for symbol in ("ocreLegendMatches", "ocreLegendBonusPer", "ocreLegendBonusMax", "sort.SliceStable"):
        assert symbol in content
