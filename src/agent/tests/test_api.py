"""Basic tests for the agent service.

These tests verify endpoint contracts and request validation.
Integration tests requiring a live LLM belong in a separate suite.
"""

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def test_health():
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "ok"
    assert data["service"] == "agent"


def test_search_coins_rejects_invalid_body():
    resp = client.post("/api/search/coins", json={})
    assert resp.status_code == 422


def test_search_coins_rejects_missing_message():
    resp = client.post(
        "/api/search/coins",
        json={
            "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
            "user": {"user_id": 1},
        },
    )
    assert resp.status_code == 422


def test_search_shows_rejects_invalid_body():
    resp = client.post("/api/search/shows", json={})
    assert resp.status_code == 422


def test_analyze_stub():
    resp = client.post(
        "/api/analyze",
        json={
            "llm": {"provider": "ollama", "ollama_url": "http://localhost:11434", "model": "llava"},
            "coin": {"id": 1, "name": "Test Coin"},
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert "message" in data


def test_portfolio_review_rejects_invalid_body():
    resp = client.post("/api/portfolio/review", json={})
    assert resp.status_code == 422


def test_intake_draft_rejects_invalid_body():
    resp = client.post("/api/intake/draft", json={})
    assert resp.status_code == 422
