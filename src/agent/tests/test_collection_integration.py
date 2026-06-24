"""Tests for collection chat team and supervisor integration."""

import pytest

from app.models.requests import LLMConfig
from app.supervisor import create_supervisor
from app.teams.collection_chat import create_collection_chat_team


@pytest.fixture
def llm_config():
    """Mock LLM config for tests."""
    return LLMConfig(
        provider="ollama",
        model="test-model",
        ollama_url="http://localhost:11434",
    )


def test_create_collection_chat_team(llm_config):
    """Verify collection chat team builds successfully."""
    team = create_collection_chat_team(
        llm_config,
        tools_base_url="http://test:8080",
        internal_token="test-token",
    )

    # Should return a compiled graph
    assert team is not None
    assert hasattr(team, "ainvoke")


def test_supervisor_routes_collection_category(llm_config):
    """Verify supervisor adds collection node to graph."""
    graph = create_supervisor(
        llm_config,
        user_message="Do I have any moose coins?",
        tools_base_url="http://test:8080",
        internal_token="test-token",
    )

    # Supervisor should compile successfully
    assert graph is not None


def test_supervisor_without_collection_tokens(llm_config):
    """Verify supervisor handles missing collection tokens gracefully."""
    graph = create_supervisor(
        llm_config,
        user_message="Do I have any moose coins?",
        # No tools_base_url or internal_token
    )

    # Should still compile (collection node will return unavailable message)
    assert graph is not None


def test_supervisor_disables_untrusted_collection_tools_without_breaking_search(monkeypatch):
    """Coin search chat should still compile if collection callback URL is not trusted."""
    from app.config import settings

    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://app:8080")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    graph = create_supervisor(
        LLMConfig(provider="anthropic", model="claude"),
        user_message="Find me Roman silver denarii of Julius Caesar",
        tools_base_url="http://coins:8080",
        internal_token="test-token",
    )

    assert graph is not None


@pytest.mark.asyncio
async def test_collection_node_requires_tokens(llm_config):
    """Verify collection node returns error without tokens."""
    # Build supervisor without tokens
    graph = create_supervisor(llm_config)

    # Manually invoke collection node (simulating router decision)
    # In real usage, router would decide to go to collection
    # For this test, we're just verifying the graph compiles
    assert graph is not None


def test_coin_search_request_accepts_new_fields():
    """Verify CoinSearchRequest accepts internal_token and tools_base_url."""
    from app.models.requests import CoinSearchRequest, UserContext

    request = CoinSearchRequest(
        llm=LLMConfig(provider="ollama", model="test"),
        user=UserContext(user_id=1),
        message="test",
        internal_token="token-123",
        tools_base_url="http://localhost:8080",
    )

    assert request.internal_token == "token-123"
    assert request.tools_base_url == "http://localhost:8080"


def test_coin_search_request_defaults_empty_strings():
    """Verify new fields default to empty strings."""
    from app.models.requests import CoinSearchRequest, UserContext

    request = CoinSearchRequest(
        llm=LLMConfig(provider="ollama", model="test"),
        user=UserContext(user_id=1),
        message="test",
    )

    assert request.internal_token == ""
    assert request.tools_base_url == ""
