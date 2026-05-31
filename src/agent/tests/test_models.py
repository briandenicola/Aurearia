"""Tests for request/response Pydantic models."""

import pytest
from pydantic import ValidationError

from app.models.requests import (
    MAX_AVAILABILITY_ITEMS,
    MAX_HISTORY_MESSAGES,
    MAX_HISTORY_TOTAL_CHARS,
    MAX_IMAGE_BASE64_LENGTH,
    MAX_IMAGE_COUNT,
    AnalyzeRequest,
    AvailabilityCheckRequest,
    CoinData,
    CoinSearchRequest,
    IntakeDraftRequest,
    LLMConfig,
    PortfolioCoin,
    PortfolioSummary,
    UserContext,
)


def test_portfolio_coin_defaults():
    """PortfolioCoin should have sensible defaults for optional fields."""
    coin = PortfolioCoin(name="Denarius")
    assert coin.grade == ""
    assert coin.category == ""
    assert coin.era == ""
    assert coin.purchase_price == 0


def test_portfolio_coin_with_all_fields():
    coin = PortfolioCoin(
        name="Denarius",
        category="Roman",
        era="Republic",
        grade="VF",
        purchase_price=150.0,
    )
    assert coin.grade == "VF"
    assert coin.purchase_price == 150.0


def test_llm_config_required_fields():
    config = LLMConfig(provider="anthropic")
    assert config.provider == "anthropic"
    assert config.api_key == ""
    assert config.model == ""


def test_llm_config_ollama():
    config = LLMConfig(provider="ollama", ollama_url="http://localhost:11434")
    assert config.ollama_url == "http://localhost:11434"


def test_user_context_defaults():
    user = UserContext(user_id=42)
    assert user.user_id == 42
    assert user.zip_code == ""


def test_portfolio_summary_null_maps_become_empty():
    """Go sends null for nil maps — validator should convert to empty dict."""
    summary = PortfolioSummary(categories=None, materials=None)
    assert summary.categories == {}
    assert summary.materials == {}


def test_portfolio_summary_null_lists_become_empty():
    """Go sends null for nil slices — validator should convert to empty list."""
    summary = PortfolioSummary(eras=None, rulers=None, top_coins=None)
    assert summary.eras == []
    assert summary.rulers == []
    assert summary.top_coins == []


def test_coin_search_request_rejects_history_over_limit():
    with pytest.raises(ValidationError):
        CoinSearchRequest(
            llm=LLMConfig(provider="anthropic"),
            user=UserContext(user_id=1),
            message="hello",
            history=[{"role": "user", "content": "x"}] * (MAX_HISTORY_MESSAGES + 1),
        )


def test_coin_search_request_rejects_history_char_over_limit():
    oversized = "x" * (MAX_HISTORY_TOTAL_CHARS + 1)
    with pytest.raises(ValidationError):
        CoinSearchRequest(
            llm=LLMConfig(provider="anthropic"),
            user=UserContext(user_id=1),
            message="hello",
            history=[{"role": "user", "content": oversized}],
        )


def test_analyze_request_rejects_image_count_over_limit():
    with pytest.raises(ValidationError):
        AnalyzeRequest(
            llm=LLMConfig(provider="anthropic"),
            coin=CoinData(id=1, name="Coin"),
            images=["a"] * (MAX_IMAGE_COUNT + 1),
        )


def test_analyze_request_rejects_oversized_base64_image():
    with pytest.raises(ValidationError):
        AnalyzeRequest(
            llm=LLMConfig(provider="anthropic"),
            coin=CoinData(id=1, name="Coin"),
            images=["a" * (MAX_IMAGE_BASE64_LENGTH + 1)],
        )


def test_intake_request_requires_at_least_one_image():
    with pytest.raises(ValidationError):
        IntakeDraftRequest(
            llm=LLMConfig(provider="anthropic"),
            images=[],
        )


def test_availability_check_request_rejects_items_over_limit():
    with pytest.raises(ValidationError):
        AvailabilityCheckRequest(
            llm=LLMConfig(provider="anthropic"),
            items=[{"url": f"https://example.com/{i}"} for i in range(MAX_AVAILABILITY_ITEMS + 1)],
        )


def test_availability_check_request_rejects_duplicate_urls():
    with pytest.raises(ValidationError):
        AvailabilityCheckRequest(
            llm=LLMConfig(provider="anthropic"),
            items=[{"url": "https://example.com/1"}, {"url": "https://example.com/1"}],
        )
