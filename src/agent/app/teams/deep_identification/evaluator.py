"""Evaluator node (T065) — contradiction/provenance detection.

The disagreement set itself is always derived deterministically from the
evidence (never from LLM output), so "never silently resolve by
precedence" (FR-027) is a structural guarantee, not a prompting hope: two
providers reporting different normalized values for the same field always
produce a `DisagreementEntry` with `resolution: "unresolved"` listing both
claims. The optional LLM call is used only to produce a short human-facing
rationale/question for the synthesizer — it can never remove or "resolve"
an already-detected disagreement.
"""

import logging

from app.models.responses import DisagreementEntry, EvidenceRef, ProviderEvidence

logger = logging.getLogger(__name__)

SUMMARY_PROMPT = (
    "You are the contradiction reviewer for a numismatic identification pipeline. "
    "You will be given a list of fields where two or more sources disagree. "
    "Write one short, neutral open question per disagreement that a collector "
    "could use to resolve it themselves — never state which source is correct. "
    "Respond with a single JSON array of strings only, one per disagreement, "
    "in the same order given. Do not use emojis."
)


class EvaluationResult:
    def __init__(self, disagreements: list[DisagreementEntry], resolved_count: int, unresolved_questions: list[str]):
        self.disagreements = disagreements
        self.resolved_count = resolved_count
        self.unresolved_questions = unresolved_questions


def _group_claims_by_field(evidence: list[ProviderEvidence]) -> dict[str, list[tuple[str, int, str]]]:
    grouped: dict[str, list[tuple[str, int, str]]] = {}
    for row in evidence:
        for idx, claim in enumerate(row.claims):
            grouped.setdefault(claim.field, []).append((row.provider, idx, claim.value))
    return grouped


def detect_disagreements(evidence: list[ProviderEvidence]) -> tuple[list[DisagreementEntry], int]:
    """Pure, deterministic disagreement detection — no LLM involved."""
    grouped = _group_claims_by_field(evidence)
    disagreements: list[DisagreementEntry] = []
    resolved_count = 0
    for field in sorted(grouped):
        entries = grouped[field]
        distinct_values = {value.strip().lower() for _, _, value in entries}
        if len(distinct_values) > 1:
            disagreements.append(
                DisagreementEntry(
                    field=field,
                    claim_refs=[EvidenceRef(provider=provider, claim_index=idx) for provider, idx, _ in entries],
                    resolution="unresolved",
                )
            )
        elif len(entries) > 1:
            resolved_count += 1
    return disagreements, resolved_count


async def evaluate(model, evidence: list[ProviderEvidence]) -> EvaluationResult:
    disagreements, resolved_count = detect_disagreements(evidence)
    unresolved_questions: list[str] = [f"Sources disagree on {d.field}." for d in disagreements]

    if disagreements and model is not None:
        try:
            unresolved_questions = await _summarize(model, disagreements) or unresolved_questions
        except Exception:
            logger.exception(
                "[deep_identification.evaluator] LLM summary failed — using deterministic fallback text"
            )

    return EvaluationResult(
        disagreements=disagreements,
        resolved_count=resolved_count,
        unresolved_questions=unresolved_questions,
    )


async def _summarize(model, disagreements: list[DisagreementEntry]) -> list[str] | None:
    import json

    from langchain_core.messages import HumanMessage, SystemMessage

    from app.llm.retry import ainvoke_with_retry

    fields = [d.field for d in disagreements]
    response = await ainvoke_with_retry(
        model,
        [SystemMessage(content=SUMMARY_PROMPT), HumanMessage(content=f"Disagreement fields: {json.dumps(fields)}")],
    )
    raw = response.content if isinstance(response.content, str) else str(response.content)
    text = raw.strip()
    start = text.find("```json")
    if start != -1:
        start += len("```json")
        end = text.find("```", start)
        text = text[start:end if end != -1 else None].strip()
    parsed = json.loads(text)
    if not isinstance(parsed, list) or len(parsed) != len(disagreements):
        return None
    return [str(item)[:500] for item in parsed]
