from unittest.mock import AsyncMock, Mock

import pytest
from langchain_core.messages import AIMessage

from app.models.requests import LLMConfig
from app.teams import coin_search


@pytest.mark.asyncio
@pytest.mark.parametrize("provider,method", [("anthropic", "function_calling"), ("ollama", "json_schema")])
async def test_market_formatter_uses_structured_output_instead_of_freeform_prose(monkeypatch, provider, method):
    payload = {"listings": [{"name": "Fixture denarius", "sourceUrl": "https://dealer.example/coin/1"}]}
    structured = AsyncMock(return_value={"parsed": payload, "parsing_error": None})
    model = AsyncMock()
    model.ainvoke.return_value = AIMessage(content="I cannot output any listings.")
    model.with_structured_output = Mock(return_value=Mock(ainvoke=structured))
    monkeypatch.setattr(coin_search, "get_chat_model", lambda _config: model)
    monkeypatch.setattr("app.llm.provider.get_chat_model", lambda _config: model)

    _, candidates = await coin_search._format_dealer_candidates(
        LLMConfig(provider=provider, api_key="fixture", model="fixture"),
        "denarius", "Source: https://dealer.example/coin/1 Fixture denarius", strict=True,
    )

    assert candidates[0]["sourceUrl"] == "https://dealer.example/coin/1"
    structured.assert_awaited_once()
    model.with_structured_output.assert_called_once_with(
        coin_search.MARKET_LISTINGS_SCHEMA, method=method, include_raw=True,
    )


@pytest.mark.asyncio
@pytest.mark.parametrize("response", [
    {"parsed": None, "parsing_error": ValueError("private provider content")},
    {"parsed": {"listings": "not an array"}, "parsing_error": None},
    {"parsed": {"listings": [None]}, "parsing_error": None},
])
async def test_malformed_structured_output_is_not_reported_as_no_matches(monkeypatch, caplog, response):
    model = Mock(ainvoke=AsyncMock(return_value=response))
    monkeypatch.setattr(coin_search, "get_structured_model", lambda *_args: model)
    with pytest.raises(coin_search.ProviderMalformedError):
        await coin_search._format_dealer_candidates(
            LLMConfig(provider="anthropic", api_key="fixture", model="fixture"), "coin", "evidence", strict=True,
        )
    assert "Market formatter failed structured output validation" in caplog.text
    assert "private provider content" not in caplog.text


@pytest.mark.asyncio
async def test_empty_fetched_page_does_not_discard_search_evidence(monkeypatch):
    url = "https://dealer.example/coin/1"
    monkeypatch.setattr(coin_search, "_search_dealer_pages", AsyncMock(return_value=f"Fixture coin {url}"))
    monkeypatch.setattr(coin_search, "_fetch_dealer_pages", AsyncMock(return_value="Empty search page"))
    monkeypatch.setattr(coin_search, "_format_dealer_candidates", AsyncMock(return_value=("[]", [])))
    fallback = AsyncMock(return_value=[{"name": "Fixture coin", "sourceUrl": url, "verificationState": "partial"}])
    monkeypatch.setattr(coin_search, "_search_result_candidates", fallback)

    result = await coin_search._collect_market_candidates(
        LLMConfig(provider="anthropic", api_key="fixture", model="fixture"), "coin", 5,
        allowed_fetch_hosts={"dealer.example"},
    )
    assert result[0]["sourceUrl"] == url
    fallback.assert_awaited_once()
