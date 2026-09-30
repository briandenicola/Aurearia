"""Shared agent test environment."""

import os

os.environ.setdefault("AGENT_INTERNAL_SERVICE_TOKEN", "test-agent-service-token")
os.environ.setdefault(
    "AGENT_TRUSTED_OUTBOUND_ORIGINS",
    "http://localhost:11434,http://localhost:8080,http://test-api:8080,http://test:8080",
)
os.environ.setdefault("AGENT_ALLOW_LOCAL_OUTBOUND", "true")

import pytest  # noqa: E402


@pytest.fixture(autouse=True)
def _reset_dealer_search_guard():
    """Dealer cool-downs and cached responses are process state; isolate every test."""
    from app.tools.search import DEALER_SEARCH_GUARD

    DEALER_SEARCH_GUARD.reset()
    yield
    DEALER_SEARCH_GUARD.reset()


@pytest.fixture(autouse=True)
def _no_live_listing_relevance(monkeypatch):
    """The title-relevance check calls an LLM; by default it fails open so tests stay offline."""
    from app.teams import listing_relevance

    async def unavailable(*_args, **_kwargs):
        return None

    monkeypatch.setattr(listing_relevance, "classify_titles", unavailable)
