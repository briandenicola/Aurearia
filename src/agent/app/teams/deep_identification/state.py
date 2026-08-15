"""Deep-identification pipeline state (contracts/agent-internal-contract.md §6).

`evidence` uses a list-append reducer so each provider-fanout node can
return `{"evidence": [one_row]}` and LangGraph merges them without a
node needing to see (or clobber) results from sibling nodes running
concurrently in the same fan-out step.
"""

import operator
from typing import Annotated, TypedDict

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

    # prepare_evidence (vision) output — free-text image observations used
    # as router/synthesis context; never itself a typed provider claim.
    image_analysis: str

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
