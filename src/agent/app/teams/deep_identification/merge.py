"""Deterministic evidence merge + citation validation (§4-5 of the internal
contract, T063/T064).

Two pure, LLM-free functions:

- `validate_citations` drops any claim whose citation host is not on the
  emitting provider's canonical allowlist (SC-006) — the LLM can never
  introduce an arbitrary URL into a persisted claim.
- `sort_claims` produces the single deterministic ordering
  `(field, provider_rank, -confidence, citation)` used before any claim is
  shown to the evaluator/synthesizer LLM, so identical inputs always
  produce the same prompt (and therefore reproducible synthesis runs).
"""

from urllib.parse import urlparse

from app.models.responses import ProviderClaim, ProviderEvidence

# Canonical per-provider citation host allowlist (§4).
CITATION_HOST_ALLOWLIST: dict[str, set[str]] = {
    "numista": {"en.numista.com", "api.numista.com"},
    "nomisma": {"nomisma.org"},
    "ngc": {"www.ngccoin.com"},
    "ocre": {"numismatics.org"},
    "rpc": {"rpc.ashmus.ox.ac.uk"},
}

# Deterministic provider rank used for claim ordering and router truncation.
PROVIDER_RANK = ["numista", "nomisma", "ngc", "ocre", "rpc"]


def _citation_host(citation: str) -> str:
    try:
        return (urlparse(citation).hostname or "").lower()
    except ValueError:
        return ""


def validate_citations(provider: str, claims: list[ProviderClaim]) -> tuple[list[ProviderClaim], int]:
    """Return (valid_claims, dropped_count). Never raises — a malformed
    citation is telemetry (`invalid_response`), not a pipeline failure.
    """
    allowlist = CITATION_HOST_ALLOWLIST.get(provider, set())
    valid: list[ProviderClaim] = []
    dropped = 0
    for claim in claims:
        host = _citation_host(claim.citation)
        if host and host in allowlist:
            valid.append(claim)
        else:
            dropped += 1
    return valid, dropped


def _provider_rank(provider: str) -> int:
    try:
        return PROVIDER_RANK.index(provider)
    except ValueError:
        return len(PROVIDER_RANK)


def sort_claims(evidence: list[ProviderEvidence]) -> list[tuple[str, ProviderEvidence, ProviderClaim]]:
    """Flatten every (provider, claim) pair across all evidence rows and
    sort by (field, provider_rank, -confidence, citation) — the single
    deterministic ordering contract §5 requires before an LLM sees claims.
    """
    flattened: list[tuple[str, ProviderEvidence, ProviderClaim]] = [
        (claim.field, row, claim) for row in evidence for claim in row.claims
    ]
    flattened.sort(key=lambda item: (item[0], _provider_rank(item[1].provider), -item[2].confidence, item[2].citation))
    return flattened
