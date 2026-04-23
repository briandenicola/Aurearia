"""Tests for request/response Pydantic models."""

from app.models.requests import (
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
