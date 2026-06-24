"""Tests for outbound URL trust boundaries."""

from unittest.mock import AsyncMock, Mock, patch

import httpx
import pytest
from pydantic import ValidationError

from app.config import settings
from app.models.requests import CoinSearchRequest, LLMConfig, UserContext
from app.outbound import safe_get, validate_public_outbound_url
from app.tools.collection_tools import build_collection_tools
from app.tools.search import fetch_dealer_page, verify_url


def test_llm_config_accepts_configured_public_origin(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "https://ollama.example.com")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    config = LLMConfig(provider="ollama", ollama_url="https://ollama.example.com")

    assert config.ollama_url == "https://ollama.example.com"


def test_llm_config_anthropic_ignores_non_ollama_url(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "https://ollama.example.com")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    config = LLMConfig(
        provider="anthropic",
        model="claude-opus-4-8",
        api_key="anthropic-key",
        ollama_url="https://ai.denicolafamily.com",
    )

    assert config.provider == "anthropic"
    assert config.ollama_url == ""


@pytest.mark.parametrize(
    "url",
    [
        "ftp://example.com",
        "http://localhost:11434",
        "http://127.0.0.1:11434",
        "http://169.254.169.254/latest/meta-data",
        "http://192.168.1.20:11434",
        "https://untrusted.example.com",
    ],
)
def test_llm_config_rejects_untrusted_or_private_urls(monkeypatch, url):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "https://ollama.example.com")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    with pytest.raises(ValidationError):
        LLMConfig(provider="ollama", ollama_url=url)


def test_tools_base_url_rejects_before_network_call(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://app:8080")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    with patch("app.tools.collection_tools.httpx.AsyncClient") as mock_client:
        with pytest.raises(ValueError):
            build_collection_tools("http://127.0.0.1:8080", "token")

    mock_client.assert_not_called()


def test_metadata_ip_is_rejected_even_when_local_dev_allowed(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://169.254.169.254")
    monkeypatch.setattr(settings, "allow_local_outbound", True)

    with pytest.raises(ValidationError):
        LLMConfig(provider="ollama", ollama_url="http://169.254.169.254/latest/meta-data")


@pytest.mark.parametrize(
    "url",
    [
        "http://169.254.169.254/latest/meta-data",
        "http://127.0.0.1:8080/admin",
        "http://10.0.0.5/internal",
        "http://192.168.1.10/internal",
    ],
)
def test_public_outbound_rejects_unsafe_targets(monkeypatch, url):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    with pytest.raises(ValueError):
        validate_public_outbound_url(url)


def test_public_outbound_local_dev_override_requires_trusted_origin(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://localhost:8080")
    monkeypatch.setattr(settings, "allow_local_outbound", True)

    assert validate_public_outbound_url("http://localhost:8080/health") == "http://localhost:8080/health"

    with pytest.raises(ValueError):
        validate_public_outbound_url("http://127.0.0.1:8080/health")


def test_public_outbound_rejects_hostname_resolving_private(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "")
    monkeypatch.setattr(settings, "allow_local_outbound", False)
    monkeypatch.setattr(
        "app.outbound.socket.getaddrinfo",
        lambda *_args, **_kwargs: [(0, 0, 0, "", ("10.0.0.5", 443))],
    )

    with pytest.raises(ValueError, match="resolves to a local or private address"):
        validate_public_outbound_url("https://dealer.example/listing")


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "url",
    [
        "http://169.254.169.254/latest/meta-data",
        "http://127.0.0.1:8080/admin",
        "http://10.0.0.5/internal",
    ],
)
async def test_safe_get_rejects_unsafe_initial_url_before_network(monkeypatch, url):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    with patch("app.outbound.httpx.AsyncClient") as mock_client_cls:
        with pytest.raises(ValueError):
            await safe_get(url)

    mock_client_cls.assert_not_called()


@pytest.mark.asyncio
async def test_safe_get_rejects_private_dns_result_before_network(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "")
    monkeypatch.setattr(settings, "allow_local_outbound", False)
    monkeypatch.setattr(
        "app.outbound.socket.getaddrinfo",
        lambda *_args, **_kwargs: [(0, 0, 0, "", ("10.0.0.5", 443))],
    )

    with patch("app.outbound.httpx.AsyncClient") as mock_client_cls:
        with pytest.raises(ValueError, match="resolves to a local or private address"):
            await safe_get("https://dealer.example/listing")

    mock_client_cls.assert_not_called()


@pytest.mark.asyncio
async def test_safe_get_rejects_redirect_to_metadata_before_following(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "https://example.com")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    redirect_response = httpx.Response(
        302,
        headers={"Location": "http://169.254.169.254/latest/meta-data"},
        request=httpx.Request("GET", "https://example.com/start"),
    )

    with patch("app.outbound.httpx.AsyncClient") as mock_client_cls:
        mock_client = AsyncMock()
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None
        mock_client.get.return_value = redirect_response
        mock_client_cls.return_value = mock_client

        with pytest.raises(ValueError):
            await safe_get("https://example.com/start")

    mock_client.get.assert_called_once()
    assert str(mock_client.get.call_args.args[0]) == "https://example.com/start"


@pytest.mark.asyncio
async def test_verify_url_rejects_private_url_without_network_call(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    with patch("app.outbound.httpx.AsyncClient") as mock_client_cls:
        result = await verify_url.ainvoke({"url": "http://127.0.0.1:8080/admin"})

    mock_client_cls.assert_not_called()
    assert "local or private address" in result


@pytest.mark.asyncio
async def test_fetch_dealer_page_rejects_redirect_to_private_without_following(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "https://example.com")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    redirect_response = httpx.Response(
        302,
        headers={"Location": "http://10.0.0.5/internal"},
        request=httpx.Request("GET", "https://example.com/listing"),
    )

    with patch("app.outbound.httpx.AsyncClient") as mock_client_cls:
        mock_client = AsyncMock()
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None
        mock_client.get.return_value = redirect_response
        mock_client_cls.return_value = mock_client

        result = await fetch_dealer_page.ainvoke({"url": "https://example.com/listing"})

    mock_client.get.assert_called_once()
    assert "local or private address" in result


def test_coin_search_request_accepts_trusted_tools_origin(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://app:8080")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    request = CoinSearchRequest(
        llm=LLMConfig(provider="anthropic", model="claude"),
        user=UserContext(user_id=1),
        message="show my collection",
        tools_base_url="http://app:8080",
    )

    assert request.tools_base_url == "http://app:8080"


def test_coin_search_request_defers_untrusted_tools_origin_validation(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://app:8080")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    request = CoinSearchRequest(
        llm=LLMConfig(
            provider="anthropic",
            api_key="anthropic-key",
            model="claude-opus-4-8",
            ollama_url="https://stale-ollama.example.com",
            searxng_url="https://stale-search.example.com",
        ),
        user=UserContext(user_id=1),
        message="find me Roman silver denarii",
        tools_base_url="http://coins:8080",
    )

    assert request.llm.ollama_url == ""
    assert request.llm.searxng_url == ""
    assert request.tools_base_url == "http://coins:8080"


@pytest.mark.asyncio
async def test_configured_local_dev_origin_allows_network_call(monkeypatch):
    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://localhost:8080")
    monkeypatch.setattr(settings, "allow_local_outbound", True)

    tools = build_collection_tools("http://localhost:8080", "token")
    summary_tool = next(tool for tool in tools if tool.name == "collection_summary")

    with patch("app.tools.collection_tools.httpx.AsyncClient") as mock_client_cls:
        mock_response = Mock()
        mock_response.json.return_value = {"summary": {"total_coins": 1}}
        mock_response.raise_for_status.return_value = None
        mock_client = AsyncMock()
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None
        mock_client.post.return_value = mock_response
        mock_client_cls.return_value = mock_client

        result = await summary_tool.ainvoke({})

    assert "Collection summary" in result
    mock_client.post.assert_called_once()
