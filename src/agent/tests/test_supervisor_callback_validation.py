"""Regression tests for optional collection callback wiring."""

from app.models.requests import LLMConfig
from app.supervisor import create_supervisor


class DummyGraph:
    async def ainvoke(self, _state):
        return {"messages": []}


async def dummy_router(_state):
    return "general"


def test_supervisor_keeps_coin_search_available_when_collection_callback_is_invalid(monkeypatch, caplog):
    def reject_collection_tools(*_args, **_kwargs):
        raise ValueError("tools_base_url origin is not trusted")

    monkeypatch.setattr("app.supervisor.create_collection_chat_team", reject_collection_tools)
    monkeypatch.setattr("app.supervisor.create_coin_search_team", lambda *_args, **_kwargs: DummyGraph())
    monkeypatch.setattr("app.supervisor.create_coin_show_team", lambda *_args, **_kwargs: DummyGraph())
    monkeypatch.setattr("app.supervisor.create_auction_search_team", lambda *_args, **_kwargs: DummyGraph())
    monkeypatch.setattr("app.supervisor.create_portfolio_review_team", lambda *_args, **_kwargs: DummyGraph())
    monkeypatch.setattr("app.supervisor.create_gap_analysis_team", lambda *_args, **_kwargs: DummyGraph())
    monkeypatch.setattr("app.supervisor.create_price_trend_team", lambda *_args, **_kwargs: DummyGraph())
    monkeypatch.setattr("app.supervisor.create_similar_lot_team", lambda *_args, **_kwargs: DummyGraph())
    monkeypatch.setattr("app.supervisor.create_router", lambda *_args, **_kwargs: dummy_router)

    graph = create_supervisor(
        LLMConfig(provider="anthropic", api_key="k", model="claude-opus-4-8"),
        user_message="Find Athenian owls for sale",
        tools_base_url="https://unrelated-callback.example.com",
        internal_token="token",
    )

    assert graph is not None
    assert "Collection tools disabled: tools_base_url origin is not trusted" in caplog.text
