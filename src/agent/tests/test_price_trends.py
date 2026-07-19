"""Refactor-safety test for price_trends.py's search step.

Confirms extracting search_auction_results() out of search_node (done to let
bid_market_signal.py reuse it, see app/teams/bid_market_signal.py) didn't
change search_node's behavior.
"""

from app.models.requests import LLMConfig
from app.teams.price_trends import create_price_trend_team, search_auction_results


class _StubResponse:
    def __init__(self, content: str):
        self.content = content


class _StubSearchModel:
    def __init__(self, content: str):
        self._content = content
        self.last_messages = None

    async def ainvoke(self, messages, **_kwargs):
        self.last_messages = messages
        return _StubResponse(self._content)


async def test_search_auction_results_returns_model_content(monkeypatch):
    stub = _StubSearchModel("Found 5 recent auction results.")
    monkeypatch.setattr("app.teams.price_trends.get_search_model", lambda _config: stub)

    llm_config = LLMConfig(provider="anthropic", api_key="k", model="m")
    result = await search_auction_results(llm_config, "Roman denarius")

    assert result == "Found 5 recent auction results."
    assert stub.last_messages is not None
    assert "Roman denarius" in stub.last_messages[-1].content


async def test_price_trend_team_search_node_still_populates_state(monkeypatch):
    stub = _StubSearchModel("Found 3 recent auction results.")
    monkeypatch.setattr("app.teams.price_trends.get_search_model", lambda _config: stub)
    monkeypatch.setattr("app.teams.price_trends.get_chat_model", lambda _config: stub)

    llm_config = LLMConfig(provider="anthropic", api_key="k", model="m")
    graph = create_price_trend_team(llm_config, user_message="Roman denarius")

    result = await graph.ainvoke(
        {"messages": [], "search_results": "", "analysis": "", "user_message": "Roman denarius"},
    )

    assert result["search_results"] == "Found 3 recent auction results."
    assert result["analysis"] == "Found 3 recent auction results."
