"""Tests for the availability check endpoint and team pipeline."""

import logging

import pytest
from fastapi.testclient import TestClient

from app.main import app
from app.models.requests import MAX_AVAILABILITY_ITEMS
from app.models.responses import AvailabilityVerdict
from app.teams.availability_check import AvailabilityVerdictParseError, parse_verdicts

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


def test_check_availability_rejects_invalid_body():
    resp = client.post("/api/check-availability", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_check_availability_returns_empty_for_no_items():
    resp = client.post(
        "/api/check-availability",
        json={
            "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
            "items": [],
        },
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["results"] == []


def test_check_availability_rejects_items_over_limit():
    resp = client.post(
        "/api/check-availability",
        json={
            "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
            "items": [{"url": f"https://example.com/{i}"} for i in range(MAX_AVAILABILITY_ITEMS + 1)],
        },
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 422


def test_parse_verdicts_valid_json():
    raw = '''Here are the results:
```json
[
  {
    "url": "https://example.com/coin/1",
    "coin_name": "Roman Denarius",
    "status": "available",
    "reason": "Buy button found",
    "confidence": "high"
  },
  {
    "url": "https://example.com/coin/2",
    "coin_name": "Greek Tetradrachm",
    "status": "unavailable",
    "reason": "Page shows sold indicator",
    "confidence": "high"
  }
]
```'''
    verdicts = parse_verdicts(raw)
    assert len(verdicts) == 2
    assert verdicts[0].status == "available"
    assert verdicts[0].coin_name == "Roman Denarius"
    assert verdicts[1].status == "unavailable"
    assert verdicts[1].confidence == "high"


def test_parse_verdicts_invalid_json():
    with pytest.raises(AvailabilityVerdictParseError):
        parse_verdicts("This is not JSON at all")


def test_parse_verdicts_no_code_fence():
    raw = '[{"url": "https://x.com", "coin_name": "Test", "status": "unknown", "reason": "ambiguous"}]'
    verdicts = parse_verdicts(raw)
    assert len(verdicts) == 1
    assert verdicts[0].status == "unknown"


def test_parse_verdicts_rejects_schema_mismatch():
    raw = '[{"url": "https://x.com", "status": "maybe", "reason": "ambiguous"}]'
    with pytest.raises(AvailabilityVerdictParseError):
        parse_verdicts(raw)


def test_parse_verdicts_rejects_unexpected_urls():
    raw = '[{"url": "https://x.com", "status": "unknown", "reason": "ambiguous"}]'
    with pytest.raises(AvailabilityVerdictParseError):
        parse_verdicts(raw, expected_items=[{"url": "https://expected.com", "coin_name": "Expected"}])


def test_check_availability_logs_and_fallbacks_on_parse_error(monkeypatch, caplog):
    class DummyGraph:
        async def ainvoke(self, _state):
            return {
                "verdicts": '[{"url": "https://example.com/coin/1", "status": "maybe", "reason": "???"}]',
            }

    monkeypatch.setattr("app.routes.create_availability_check_team", lambda _llm: DummyGraph())

    with caplog.at_level(logging.ERROR):
        resp = client.post(
            "/api/check-availability",
            json={
                "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
                "items": [{"url": "https://example.com/coin/1", "coin_name": "Roman Denarius"}],
            },
            headers=AUTH_HEADERS,
        )

    assert resp.status_code == 200
    data = resp.json()
    assert data["results"] == [
        {
            "url": "https://example.com/coin/1",
            "coin_name": "Roman Denarius",
            "status": "unknown",
            "reason": "Agent response failed schema validation",
            "confidence": "low",
        },
    ]
    assert any("Availability verdict parse failure:" in rec.message for rec in caplog.records)


def test_availability_verdict_model():
    v = AvailabilityVerdict(
        url="https://example.com",
        status="available",
        reason="Active listing",
    )
    assert v.confidence == "medium"  # default
    assert v.coin_name == ""  # default
