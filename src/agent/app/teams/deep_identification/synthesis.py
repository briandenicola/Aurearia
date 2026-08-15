"""Synthesis node (T066) — strict-JSON typed final output (`DeepSynthesis`).

The structured parts of the synthesis (`proposed_fields`, `disagreements`,
`coverage`) are assembled deterministically from already-validated evidence
— never invented by the LLM. The LLM is used only to write the free-text
`narrative` summary, which itself is validated for length by the
`DeepSynthesis` pydantic model on assembly (extra keys the LLM might add
are simply never read).
"""

import logging

from app.models.responses import (
    DeepSynthesis,
    DisagreementEntry,
    EvidenceRef,
    ProposedFieldValue,
    ProviderCoverageEntry,
    ProviderEvidence,
)
from app.safety import with_safety
from app.teams.deep_identification.merge import sort_claims

logger = logging.getLogger(__name__)

NARRATIVE_PROMPT = with_safety("""You are the final report writer for a numismatic deep-identification
pipeline. You will be given the provider evidence gathered for one coin.
Write a short narrative summary (3-6 sentences) covering what was found,
in plain prose. Reference specific findings but do not invent facts beyond
what is given. If evidence is sparse or contradictory, say so plainly.

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
    evidence: list[ProviderEvidence], disagreement_fields: set[str]
) -> dict[str, ProposedFieldValue]:
    grouped: dict[str, list[tuple[ProviderEvidence, "object"]]] = {}
    for field, row, claim in sort_claims(evidence):
        if field in disagreement_fields:
            continue
        grouped.setdefault(field, []).append((row, claim))

    proposed: dict[str, ProposedFieldValue] = {}
    for field, items in grouped.items():
        best_row, best_claim = items[0]
        refs = [
            EvidenceRef(provider=row.provider, claim_index=row.claims.index(claim))
            for row, claim in items
        ]
        proposed[field] = ProposedFieldValue(
            value=best_claim.value, confidence=best_claim.confidence, evidence_refs=refs
        )
    return proposed


def _build_coverage(evidence: list[ProviderEvidence]) -> list[ProviderCoverageEntry]:
    return [ProviderCoverageEntry(provider=row.provider, status=row.status) for row in evidence]


async def synthesize(
    model,
    evidence: list[ProviderEvidence],
    disagreements: list[DisagreementEntry],
    unresolved_questions: list[str],
    partial_success: bool,
) -> DeepSynthesis:
    disagreement_fields = {d.field for d in disagreements}
    proposed_fields = _build_proposed_fields(evidence, disagreement_fields)
    coverage = _build_coverage(evidence)

    contributing = [row for row in evidence if row.status == "contributed"]
    if not contributing:
        narrative = FALLBACK_NARRATIVE_NO_EVIDENCE
    else:
        narrative = await _write_narrative(model, evidence, disagreements) or FALLBACK_NARRATIVE_ON_ERROR

    return DeepSynthesis(
        narrative=narrative[:8000],
        proposed_fields=proposed_fields,
        disagreements=disagreements,
        unresolved_questions=unresolved_questions[:20],
        coverage=coverage,
        partial_success=partial_success,
    )


async def _write_narrative(
    model, evidence: list[ProviderEvidence], disagreements: list[DisagreementEntry]
) -> str | None:
    from langchain_core.messages import HumanMessage, SystemMessage

    from app.llm.retry import ainvoke_with_retry

    summary_lines = []
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
    content = response.content if isinstance(response.content, str) else str(response.content)
    return content.strip() or None
