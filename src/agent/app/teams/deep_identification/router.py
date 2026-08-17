"""Router node (contracts/agent-internal-contract.md §6, T044-T046).

A **pure, deterministic** function of `(catalog, provider_override, bounds,
quick_evidence, hypothesis)` selects which *automatable* providers to query
this run, bounded by `bounds.max_providers`. No LLM call is made — this was
previously an LLM-router step (344 FR-022); it is now a plain evidence-driven
selector (spec FR-013/FR-014, ADR 0012). Non-automatable catalog entries
(`ngc`, `ocre`, `rpc` when their flag is off) are never subject to router
reasoning — they always run (trivially, with no network call) and are
represented directly in the final evidence list by their own provider nodes,
so they are excluded from `selected`/`skipped` here (those two lists describe
only automatable candidates the router did or didn't pick).

`provider_override`, when non-empty, replaces selection reasoning entirely:
only catalog-listed automatable providers named in the override are selected,
and no other automatable provider runs. The override can never introduce a
provider absent from the catalog — Go controls the closed candidate list.

Selection is **inclusion by default** (spec FR-015, RD-7): every automatable,
in-bounds provider is selected unless a stated reason says otherwise. The
only evidence-driven skip this router applies is OCRE on a *positive*
non-Roman-Imperial era signal (greek/islamic/byzantine/modern) drawn from the
hypothesis or quick evidence — the mere *absence* of a Roman signal must
never cause a skip, because that is precisely the state an unreadable coin
leaves the pipeline in, and OCRE is the provider most likely to identify a
Roman Imperial coin at exactly the moment the system knows least (RD-7).
"""

from app.models.hypothesis import CoinHypothesis
from app.models.requests import DeepIdentifyBounds, DeepProviderCatalogEntry, QuickEvidence
from app.teams.deep_identification.merge import PROVIDER_RANK
from app.teams.deep_identification.state import RouterSkip

# Non-Roman-Imperial era/category signals (RD-7). Matched case-insensitively
# against the hypothesis era/coin_type fields and quick_evidence's raw
# `category`/`era` coin fields (Go's CoinLookupService populates `category`
# with values like "Greek"/"Byzantine"/"Roman"/"Modern" -
# src/api/services/coin_lookup_service.go:398). A match on any of these is a
# *positive* non-Roman signal; their absence is not evidence of anything.
_NON_ROMAN_ERA_SIGNALS = ("greek", "islamic", "byzantine", "modern")


class RouterDecision:
    """Result of the router step."""

    def __init__(self, selected: list[str], skipped: list[RouterSkip], rationale: str) -> None:
        self.selected = selected
        self.skipped = skipped
        self.rationale = rationale


def _automatable_names(catalog: list[DeepProviderCatalogEntry]) -> list[str]:
    return [entry.provider for entry in catalog if entry.automatable]


def _rank(name: str) -> int:
    try:
        return PROVIDER_RANK.index(name)
    except ValueError:
        return len(PROVIDER_RANK)


def _non_roman_signal(quick_evidence: QuickEvidence | None, hypothesis: CoinHypothesis | None) -> str | None:
    """Return the matched non-Roman-Imperial era/category keyword, or
    `None` when no positive signal is present anywhere. Never treats a
    missing/empty value as a signal.
    """
    candidates: list[str] = []
    if hypothesis is not None:
        if hypothesis.era is not None:
            candidates.append(hypothesis.era.value)
        if hypothesis.coin_type is not None:
            candidates.append(hypothesis.coin_type.value)
    if quick_evidence is not None:
        coin_fields = quick_evidence.coin_fields or {}
        category = coin_fields.get("category")
        if isinstance(category, str):
            candidates.append(category)
        era = coin_fields.get("era")
        if isinstance(era, str):
            candidates.append(era)

    for candidate in candidates:
        normalized = candidate.strip().lower()
        for keyword in _NON_ROMAN_ERA_SIGNALS:
            if keyword in normalized:
                return keyword
    return None


def route(
    catalog: list[DeepProviderCatalogEntry],
    provider_override: list[str],
    bounds: DeepIdentifyBounds | None,
    quick_evidence: QuickEvidence | None = None,
    hypothesis: CoinHypothesis | None = None,
) -> RouterDecision:
    """Select which automatable providers to query this run. Pure and
    deterministic (spec FR-014, SC-006): identical inputs always produce an
    identical `selected` set, `skipped` list (with reasons), and
    `rationale`.
    """
    automatable = _automatable_names(catalog)
    if not automatable:
        return RouterDecision(selected=[], skipped=[], rationale="no automatable providers in catalog")

    budget = max(0, bounds.max_providers if bounds else 0)

    if provider_override:
        # Override wins outright — no reasoning, no free-form provider names.
        allowed_override = [p for p in provider_override if p in automatable]
        selected = sorted(allowed_override, key=_rank)[:budget] if budget else []
        skipped = [
            RouterSkip(provider=p, reason="not requested by provider_override")
            for p in automatable
            if p not in selected
        ]
        return RouterDecision(
            selected=selected, skipped=skipped, rationale="provider_override supplied by caller"
        )

    # Inclusion by default (FR-015, RD-7): every automatable provider is a
    # candidate unless a stated evidence-driven reason excludes it.
    signal = _non_roman_signal(quick_evidence, hypothesis)
    evidence_skips: dict[str, str] = {}
    if signal is not None and "ocre" in automatable:
        evidence_skips["ocre"] = f"non-Roman-Imperial era signal: {signal}"

    candidates = [p for p in automatable if p not in evidence_skips]
    ordered = sorted(candidates, key=_rank)
    selected = ordered[:budget] if budget else []
    selected_set = set(selected)

    skipped: list[RouterSkip] = []
    for p in automatable:
        if p in selected_set:
            continue
        if p in evidence_skips:
            skipped.append(RouterSkip(provider=p, reason=evidence_skips[p]))
        else:
            skipped.append(RouterSkip(provider=p, reason="max_providers limit reached"))

    rationale = f"inclusion by default: selected {len(selected)} of {len(automatable)} automatable providers"
    if evidence_skips:
        reasons = ", ".join(f"{p} ({reason})" for p, reason in sorted(evidence_skips.items()))
        rationale = f"{rationale}; evidence-driven skip: {reasons}"

    return RouterDecision(selected=selected, skipped=skipped, rationale=rationale)
