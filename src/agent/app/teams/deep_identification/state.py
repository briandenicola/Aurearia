"""Deep-identification pipeline state (contracts/agent-internal-contract.md §6).

`evidence` uses a list-append reducer so each provider-fanout node can
return `{"evidence": [one_row]}` and LangGraph merges them without a
node needing to see (or clobber) results from sibling nodes running
concurrently in the same fan-out step.

T102: `tools_base_url`, `internal_token`, and `errors` were previously
declared here but never populated into the state dict nor read by any
node — `run_deep_identification_stream` builds its `ProviderToolsClient`
directly from `request.tools_base_url`/`request.internal_token` (the real,
still-used request fields of the same name), never from this state's own
copies, and nothing ever wrote to `state["errors"]`. All three were
genuinely internal dead declarations (no Go↔Python wire dependency), so
they were removed rather than kept as documented placeholders.
"""

import operator
from typing import Annotated, TypedDict

from app.models.hypothesis import CoinHypothesis
from app.models.requests import DeepIdentifyBounds, DeepIdentifyImage, DeepProviderCatalogEntry, QuickEvidence
from app.models.responses import DeepFaceAnalysis, DisagreementEntry, ProviderEvidence


class RouterSkip(TypedDict):
    """One provider the router did not select this run, with a reason."""

    provider: str
    reason: str


class DeepIdentificationState(TypedDict, total=False):
    """State flowing through the deep-identification graph."""

    job_id: int
    images: list[DeepIdentifyImage]
    notes: str
    obverse_prompt: str
    reverse_prompt: str
    quick_evidence: QuickEvidence | None
    catalog: list[DeepProviderCatalogEntry]
    provider_override: list[str]
    bounds: DeepIdentifyBounds

    # Role-specific Collection AI Analysis evidence retained for review. It
    # is not provider evidence and never mutates saved side-analysis fields.
    face_analyses: list[DeepFaceAnalysis]

    # Typed hypothesis derived from both face narratives, Quick Lookup
    # evidence, and collector notes. It carries no citation and is consumed by
    # routing, query construction, deterministic evaluation, and synthesis.
    hypothesis: CoinHypothesis

    # Which rung of the hypothesis degrade ladder actually produced
    # `hypothesis` above: "structured" | "prose" | "deterministic_fallback"
    # | "no_face_analysis" (see
    # `hypothesis.py::build_hypothesis_from_face_analyses_traced`).
    # Consumed only by the streaming driver's FR-040 `vision_completed`
    # progress message — never a claim/citation source, never persisted to
    # the coin record.
    hypothesis_source: str

    # router output
    selected: list[str]
    skipped: list[RouterSkip]
    router_rationale: str

    # provider_fanout output (list reducer — each node contributes one row)
    evidence: Annotated[list[ProviderEvidence], operator.add]

    # evaluator output
    disagreements: list[DisagreementEntry]
    resolved_count: int

    # synthesizer output
    synthesis: dict | None
