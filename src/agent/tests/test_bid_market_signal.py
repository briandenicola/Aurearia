"""Tests for the bid market signal endpoint and team pipeline."""

import logging

import pytest
from fastapi.testclient import TestClient

from app.main import app
from app.teams.bid_market_signal import MarketSignalParseError, parse_market_signal

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


def test_bid_market_signal_rejects_invalid_body():
    resp = client.post("/api/bid-market-signal", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_parse_market_signal_valid_json():
    raw = '''Here is the signal:
```json
{
  "trend_direction": "rising",
  "price_low": 100.0,
  "price_high": 250.0,
  "currency": "USD",
  "sample_size": 6,
  "rationale": "Recent CNG and Roma sales trending upward for this type.",
  "sources": ["https://cngcoins.com/lot/1", "https://romanumismatics.com/lot/2"]
}
```'''
    signal = parse_market_signal(raw)
    assert signal.trend_direction == "rising"
    assert signal.price_low == 100.0
    assert signal.price_high == 250.0
    assert signal.sample_size == 6
    assert signal.degraded is False
    assert len(signal.sources) == 2


def test_parse_market_signal_no_code_fence():
    raw = '{"trend_direction": "stable", "sample_size": 3, "rationale": "Stable market."}'
    signal = parse_market_signal(raw)
    assert signal.trend_direction == "stable"
    assert signal.sample_size == 3
    assert signal.price_low is None


def test_parse_market_signal_invalid_json():
    with pytest.raises(MarketSignalParseError):
        parse_market_signal("This is not JSON at all")


def test_parse_market_signal_rejects_schema_mismatch():
    with pytest.raises(MarketSignalParseError):
        parse_market_signal('{"trend_direction": "sideways"}')


def test_parse_market_signal_empty_input_is_degraded_not_an_error():
    signal = parse_market_signal("")
    assert signal.degraded is True
    assert signal.trend_direction == "unknown"
    assert signal.rationale


def test_parse_market_signal_blank_input_is_degraded_not_an_error():
    signal = parse_market_signal("   \n  ")
    assert signal.degraded is True


def test_bid_market_signal_returns_ok_signal(monkeypatch):
    class DummyGraph:
        async def ainvoke(self, _state):
            return {
                "signal_raw": '''```json
{"trend_direction": "declining", "price_low": 50.0, "price_high": 90.0,
 "currency": "USD", "sample_size": 4, "rationale": "Softening demand.", "sources": []}
```''',
            }

    monkeypatch.setattr(
        "app.routes.create_bid_market_signal_team",
        lambda _llm, coin: DummyGraph(),
    )

    resp = client.post(
        "/api/bid-market-signal",
        json={
            "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
            "coin": {"id": 1, "name": "Roman Denarius", "category": "Roman"},
        },
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["trend_direction"] == "declining"
    assert data["price_low"] == 50.0
    assert data["degraded"] is False


def test_bid_market_signal_degrades_on_graph_exception(monkeypatch, caplog):
    class DummyGraph:
        async def ainvoke(self, _state):
            raise RuntimeError("search backend unavailable")

    monkeypatch.setattr(
        "app.routes.create_bid_market_signal_team",
        lambda _llm, coin: DummyGraph(),
    )

    with caplog.at_level(logging.ERROR):
        resp = client.post(
            "/api/bid-market-signal",
            json={
                "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
                "coin": {"id": 1, "name": "Roman Denarius"},
            },
            headers=AUTH_HEADERS,
        )

    assert resp.status_code == 200
    data = resp.json()
    assert data["degraded"] is True


def test_bid_market_signal_degrades_on_parse_failure(monkeypatch, caplog):
    class DummyGraph:
        async def ainvoke(self, _state):
            return {"signal_raw": "not json at all"}

    monkeypatch.setattr(
        "app.routes.create_bid_market_signal_team",
        lambda _llm, coin: DummyGraph(),
    )

    with caplog.at_level(logging.ERROR):
        resp = client.post(
            "/api/bid-market-signal",
            json={
                "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
                "coin": {"id": 1, "name": "Roman Denarius"},
            },
            headers=AUTH_HEADERS,
        )

    assert resp.status_code == 200
    data = resp.json()
    assert data["degraded"] is True
    assert any("Bid market signal parse failure" in rec.message for rec in caplog.records)


def test_bid_market_signal_ollama_without_searxng_degrades_without_calling_llm(monkeypatch):
    def _fail_if_called(*_args, **_kwargs):
        raise AssertionError("should not call the LLM when Ollama has no SearXNG configured")

    monkeypatch.setattr("app.teams.bid_market_signal.get_chat_model", _fail_if_called)
    monkeypatch.setattr("app.teams.bid_market_signal.create_search_agent", _fail_if_called)

    resp = client.post(
        "/api/bid-market-signal",
        json={
            "llm": {"provider": "ollama", "model": "llama3.1", "ollama_url": "http://localhost:11434"},
            "coin": {"id": 1, "name": "Roman Denarius"},
        },
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["degraded"] is True
