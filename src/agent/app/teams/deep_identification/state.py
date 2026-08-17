"""Deep-identification pipeline state (contracts/agent-internal-contract.md §6).

`evidence` uses a list-append reducer so each provider-fanout node can
return `{"evidence": [one_row]}` and LangGraph merges them without a
node needing to see (or clobber) results from sibling nodes running
concurrently in the same fan-out step.
"""

import operator
from typing import Annotated, TypedDict

from app.models.hypothesis import CoinHypothesis
from app.models.requests import DeepIdentifyBounds, DeepIdentifyImage, DeepProviderCatalogEntry, QuickEvidence
from app.models.responses import DisagreementEntry, ProviderEvidence


class RouterSkip(TypedDict):
    """One provider the router did not select this run, with a reason."""

    provider: str
    reason: str


class DeepIdentificationState(TypedDict, total=False):
    """State flowing through the deep-identification graph."""

    job_id: int
    images: list[DeepIdentifyImage]
    notes: str
    quick_evidence: QuickEvidence | None
    catalog: list[DeepProviderCatalogEntry]
    provider_override: list[str]
    bounds: DeepIdentifyBounds
    tools_base_url: str
    internal_token: str

    # prepare_evidence (vision) output — the typed hypothesis (Phase 3/4,
    # contracts/vision-hypothesis.md §1). Produced primarily by a real
    # single-vision-LLM-call structured extraction
    # (`app/teams/deep_identification/hypothesis.py::build_hypothesis_from_vision`);
    # degrades through prose extraction and a deterministic quick-evidence
    # adapter on any vision failure, never to an unhandled exception (spec
    # FR-006). It carries no citation and is never converted into a
    # `ProviderClaim`; `"image"` may appear only as an `EvidenceRef.provider`
    # value, never in `ProviderName`, `provider_catalog`, `coverage`, or
    # `attributions` (spec FR-004).
    #
    # The contract specifies FOUR consumers: the router (provider selection
    # signals), provider query construction (deterministic query terms), the
    # evaluator (a first-class claim source alongside provider claims,
    # tagged `source="image"`), and the synthesizer (narrative input plus
    # image-derived proposed fields). As of this batch (Phase 3/4 — the
    # keystone), only the synthesizer reads this key; router/query/evaluator
    # wiring is Phase 5-7 scope and NOT yet done — until it lands, the B2
    # "write-only state field" defect this feature exists to remove is only
    # partially fixed. This corrects an earlier, long-false version of this
    # docstring that claimed the vision output was router/synthesis context
    # only and "never itself a typed provider claim" while the field itself
    # (`image_analysis: str`) had zero readers for the entire life of the
    # feature.
    hypothesis: CoinHypothesis

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
    errors: Annotated[list[str], operator.add]
