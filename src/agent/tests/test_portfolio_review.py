"""Tests for Team 4 portfolio review pipeline behavior."""

from types import SimpleNamespace

import pytest

from app.models.requests import LLMConfig, PortfolioSummary
from app.teams.portfolio_review import create_portfolio_review_team


@pytest.mark.asyncio
async def test_portfolio_review_does_not_repeat_raw_portfolio_payload(monkeypatch):
    """Only the reader stage should receive the full raw portfolio payload."""
    calls: list[list] = []

    async def _fake_ainvoke(_model, messages, **_kwargs):
        calls.append(messages)
        return SimpleNamespace(content="ok")

    monkeypatch.setattr("app.teams.portfolio_review.ainvoke_with_retry", _fake_ainvoke)

    portfolio = PortfolioSummary(
        total_coins=2,
        total_value=200.0,
        total_invested=150.0,
        categories={"Roman": 2},
    )
    graph = create_portfolio_review_team(
        LLMConfig(provider="anthropic", api_key="k", model="claude-sonnet-5"),
        portfolio=portfolio,
        user_message="How is my portfolio doing?",
    )

    await graph.ainvoke({
        "messages": [],
        "portfolio_summary": "",
        "valuation_commentary": "",
        "final_analysis": "",
        "user_message": "How is my portfolio doing?",
    })

    assert len(calls) == 3
    first_human = str(calls[0][1].content)
    second_human = str(calls[1][1].content)
    third_human = str(calls[2][1].content)

    assert "Here is the portfolio data:" in first_human
    assert "Raw portfolio data:" not in second_human
    assert "Raw data:" not in third_human
