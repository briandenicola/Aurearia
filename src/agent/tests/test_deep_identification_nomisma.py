"""Nomisma provider node tests (Task G regression — 351-vision-first-deep-identification).

Traced bug: `_build_query` forwarded `quick_evidence.label_text` (bounded to
2000 runes on the wire) to the Go Nomisma internal tool unbounded, while Go's
`HTTPNomismaClient.Search` rejects anything over `nomismaMaxQueryLength`
(200 runes) as `NomismaErrorInvalidRequest` — misreported through the tool's
`default:` branch as `"unavailable"`, which this node then classified as an
upstream failure. Every slabbed coin (long NGC label text) failed Nomisma in
~5ms and was misreported as an outage instead of "we sent a bad query".
"""

import asyncio

from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.teams.deep_identification.providers import nomisma
from app.teams.deep_identification.providers.nomisma import _MAX_QUERY_LENGTH, _build_query


class FakeNomismaTools:
    """Records the exact query string sent and returns a scripted response."""

    def __init__(self, response=None):
        self._response = response or {"status": "ok", "candidates": [], "attribution": "Data: Nomisma.org (CC BY)"}
        self.calls: list[str] = []

    async def nomisma_search(self, query: str, limit: int = 5):
        self.calls.append(query)
        return self._response


def _entry():
    return DeepProviderCatalogEntry(provider="nomisma", automatable=True, call_budget=3)


def test_build_query_bounds_an_over_length_label_text():
    """The over-length label_text a real NGC slab label produces (up to the
    2000-rune QuickEvidence limit) must never reach the Go client unbounded —
    Go's Nomisma client caps queries at 200 runes.
    """
    long_label = "X" * 2000
    quick_evidence = QuickEvidence(label_text=long_label)

    query = _build_query(quick_evidence, notes="")

    assert len(query) <= _MAX_QUERY_LENGTH


def test_over_length_query_never_reaches_the_tools_client():
    long_label = ("MAXIMINUS I THRAX AR DENARIUS ROME MINT " * 40)[:1900]  # well over 200, under the 2000 wire cap
    quick_evidence = QuickEvidence(label_text=long_label)
    tools = FakeNomismaTools()

    asyncio.run(nomisma.run(_entry(), tools, quick_evidence, notes=""))

    assert len(tools.calls) == 1
    assert len(tools.calls[0]) <= _MAX_QUERY_LENGTH


def test_invalid_request_status_is_not_reported_as_an_upstream_failure():
    """Go's mirror of an over-length/malformed query now returns
    `status: "invalid_request"`, distinct from `"unavailable"` (a real
    upstream outage). This node must map it to `no_match`, never `failed`
    with `error_kind="upstream"` — our own bug is not Nomisma's outage.
    """
    tools = FakeNomismaTools(response={"status": "invalid_request", "candidates": []})

    result = asyncio.run(nomisma.run(_entry(), tools, QuickEvidence(label_text="short"), notes=""))

    assert result.status == "no_match"
    assert result.error_kind != "upstream"
    assert result.call_count == 0


def test_genuine_upstream_outage_is_still_reported_as_failed():
    tools = FakeNomismaTools(response={"status": "unavailable", "candidates": []})

    result = asyncio.run(nomisma.run(_entry(), tools, QuickEvidence(label_text="short"), notes=""))

    assert result.status == "failed"
    assert result.error_kind == "upstream"
