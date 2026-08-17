"""Synthesis node (T066) — strict-JSON typed final output (`DeepSynthesis`).

The structured parts of the synthesis (`proposed_fields`, `disagreements`,
`coverage`) are assembled deterministically from already-validated evidence
— never invented by the LLM. The LLM is used only to write the free-text
`narrative` summary, which itself is validated for length by the
`DeepSynthesis` pydantic model on assembly (extra keys the LLM might add
are simply never read).
"""

import logging

from app.llm.content import extract_text_content
from app.models.hypothesis import CoinHypothesis
from app.models.responses import (
    DeepSynthesis,
    DisagreementEntry,
    EvidenceRef,
    ProposedFieldValue,
    ProviderAttribution,
    ProviderCoverageEntry,
    ProviderEvidence,
)
from app.safety import with_safety
from app.teams.deep_identification.merge import sort_claims

logger = logging.getLogger(__name__)

# RD-2 (spec.md, decided 2026-08-17; binds FR-022; tasks.md T004/T061): flat
# bonus applied once per field when a provider claim corroborates the image
# hypothesis on an exact normalized match — never stacked across multiple
# corroborating providers, never LLM-adjusted, never pushed above 1.0.
CORROBORATION_CONFIDENCE_BONUS = 0.10

NARRATIVE_PROMPT = with_safety("""You are the final report writer for a numismatic deep-identification
pipeline. You will be given an image-derived hypothesis for this coin (if
any) and the evidence gathered from each catalogue/authority provider.

Write a short narrative (3-6 sentences), in plain prose, that covers:
- what the images support — ruler, denomination, material, and any other
  legible identity signal, including an NGC certification number/grade
  when one is present;
- what each provider confirmed, refined, or contradicted;
- what remains open or unconfirmed.

Reference specific findings; never invent facts beyond what is given. If a
provider found nothing or could not be queried, say so plainly rather than
omitting it. If everything is sparse or contradictory, say so plainly.

Respond with plain text only — no JSON, no markdown headers, no emojis.""")

FALLBACK_NARRATIVE_NO_EVIDENCE = (
    "No provider evidence could be gathered for this coin. Please review the "
    "image-based analysis and consider retrying once providers are available."
)
FALLBACK_NARRATIVE_ON_ERROR = (
    "A narrative summary could not be generated, but the structured findings "
    "below reflect the evidence gathered from each provider."
)


def _build_proposed_fields(
    evidence: list[ProviderEvidence],
    disagreement_fields: set[str],
    hypothesis: CoinHypothesis | None,
) -> dict[str, ProposedFieldValue]:
    grouped: dict[str, list[tuple[ProviderEvidence, "object"]]] = {}
    for field, row, claim in sort_claims(evidence):
        if field in disagreement_fields:
            continue
        grouped.setdefault(field, []).append((row, claim))

    hypothesis_fields = hypothesis.fields() if hypothesis is not None else {}

    proposed: dict[str, ProposedFieldValue] = {}
    for field, items in grouped.items():
        best_row, best_claim = items[0]
        refs = [
            EvidenceRef(provider=row.provider, claim_index=row.claims.index(claim))
            for row, claim in items
        ]
        value = best_claim.value
        confidence = best_claim.confidence

        hyp = hypothesis_fields.get(field)
        if hyp is not None and hyp.value.strip().lower() == value.strip().lower():
            # FR-022/RD-2: exact normalized match between the image
            # hypothesis and the top provider claim — a flat, deterministic
            # confidence bump applied once regardless of how many providers
            # corroborate (no stacking), never LLM-adjusted, never > 1.0.
            confidence = min(1.0, max(hyp.confidence, confidence) + CORROBORATION_CONFIDENCE_BONUS)
            refs.append(EvidenceRef(provider="image"))

        proposed[field] = ProposedFieldValue(value=value, confidence=confidence, evidence_refs=refs)

    # FR-021: an image-only field — the hypothesis supports it but no
    # provider contributed a (non-contradicted) claim for it — is proposed
    # at the hypothesis's own confidence with a bare `image` evidence ref.
    for field, hyp in hypothesis_fields.items():
        if field in proposed or field in disagreement_fields:
            continue
        proposed[field] = ProposedFieldValue(
            value=hyp.value, confidence=hyp.confidence, evidence_refs=[EvidenceRef(provider="image")]
        )
    return proposed


def _build_coverage(evidence: list[ProviderEvidence]) -> list[ProviderCoverageEntry]:
    # T063/FR-025: this iterates only `evidence: list[ProviderEvidence]`,
    # which by construction (§4) never contains an `image` row — `image` is
    # not a provider and is carried solely via `EvidenceRef` on proposed
    # fields, never as a `ProviderEvidence`/coverage/attribution entry.
    return [ProviderCoverageEntry(provider=row.provider, status=row.status) for row in evidence]


def _build_attributions(evidence: list[ProviderEvidence]) -> list[ProviderAttribution]:
    """One attribution entry per provider that BOTH carries a non-empty
    attribution string AND surfaced ≥1 claim (FR-019). Fully deterministic,
    no LLM: attribution appears only when the provider actually contributed,
    and each provider's text is kept distinct (never merged). `identifier`
    carries the top claim's canonical citation so the UI can deep-link the
    exact contributed type.
    """
    attributions: list[ProviderAttribution] = []
    for row in evidence:
        if not row.attribution or not row.claims:
            continue
        attributions.append(
            ProviderAttribution(
                provider=row.provider,
                text=row.attribution,
                identifier=row.claims[0].citation or None,
            )
        )
    return attributions


async def synthesize(
    model,
    evidence: list[ProviderEvidence],
    disagreements: list[DisagreementEntry],
    unresolved_questions: list[str],
    partial_success: bool,
    hypothesis: CoinHypothesis | None = None,
) -> DeepSynthesis:
    disagreement_fields = {d.field for d in disagreements}
    proposed_fields = _build_proposed_fields(evidence, disagreement_fields, hypothesis)
    coverage = _build_coverage(evidence)
    attributions = _build_attributions(evidence)

    contributing = [row for row in evidence if row.status == "contributed"]
    hypothesis_supported = hypothesis is not None and not hypothesis.is_empty()
    # FR-020/T059: the no-evidence fallback is reachable ONLY when both the
    # hypothesis and provider evidence are empty. A hypothesis-supported run
    # with zero contributing providers (the exact Maximinus shape — NGC
    # not_automated, everything else no_match/failed) must still get a real
    # narrative, not this fallback.
    if not contributing and not hypothesis_supported:
        narrative = FALLBACK_NARRATIVE_NO_EVIDENCE
    else:
        narrative = (
            await _write_narrative(model, evidence, disagreements, hypothesis) or FALLBACK_NARRATIVE_ON_ERROR
        )

    return DeepSynthesis(
        narrative=narrative[:8000],
        proposed_fields=proposed_fields,
        disagreements=disagreements,
        unresolved_questions=unresolved_questions[:20],
        coverage=coverage,
        attributions=attributions,
        # T030/contracts/vision-hypothesis.md §4: additive, present only
        # when the vision call actually produced something, so a
        # typed-empty hypothesis (e.g. no images, or every rung of the
        # degrade ladder failing) never pollutes persisted reports.
        image_hypothesis=hypothesis if hypothesis is not None and not hypothesis.is_empty() else None,
        partial_success=partial_success,
    )


async def _write_narrative(
    model,
    evidence: list[ProviderEvidence],
    disagreements: list[DisagreementEntry],
    hypothesis: CoinHypothesis | None,
) -> str | None:
    from langchain_core.messages import HumanMessage, SystemMessage

    from app.llm.retry import ainvoke_with_retry

    summary_lines: list[str] = []
    if hypothesis is not None and not hypothesis.is_empty():
        hyp_fields = hypothesis.fields()
        if hyp_fields:
            hyp_summary = "; ".join(f"{field}={value.value}" for field, value in hyp_fields.items())
            summary_lines.append(f"Image hypothesis: {hyp_summary}")
        if hypothesis.observations:
            summary_lines.append(f"Image observations: {hypothesis.observations}")

    for row in evidence:
        if row.status == "contributed":
            claim_summary = "; ".join(f"{c.field}={c.value}" for c in row.claims)
            summary_lines.append(f"{row.provider}: {claim_summary}")
        else:
            summary_lines.append(f"{row.provider}: {row.status}")
    if disagreements:
        summary_lines.append("Disagreements: " + ", ".join(d.field for d in disagreements))

    try:
        response = await ainvoke_with_retry(
            model,
            [SystemMessage(content=NARRATIVE_PROMPT), HumanMessage(content="\n".join(summary_lines))],
        )
    except Exception:
        logger.exception("[deep_identification.synthesis] narrative LLM call failed")
        return None
    return extract_text_content(response.content) or None
