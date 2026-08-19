"""Tests for the wishlist featured-summary route."""

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


def _payload() -> dict:
    return {
        "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
        "coin": {
            "name": "Trajan Denarius",
            "era": "Roman Imperial",
            "category": "Roman",
            "denomination": "Denarius",
            "ruler": "Trajan",
            "mint": "Rome",
            "obverse_analysis": "",
            "reverse_analysis": "",
            "ai_analysis": "",
        },
        "user_display_name": "Brian",
    }


def test_wishlist_featured_summary_rejects_invalid_body():
    resp = client.post("/api/collection/wishlist-featured-summary", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_wishlist_featured_summary_returns_summary(monkeypatch):
    async def _ok(_request):
        return "A classic Trajan denarius prized for Roman imperial history."

    monkeypatch.setattr("app.routes.generate_wishlist_featured_summary", _ok)
    resp = client.post("/api/collection/wishlist-featured-summary", json=_payload(), headers=AUTH_HEADERS)
    assert resp.status_code == 200
    assert resp.json()["summary"]


def test_wishlist_featured_summary_returns_502_on_empty(monkeypatch):
    async def _blank(_request):
        return "   "

    monkeypatch.setattr("app.routes.generate_wishlist_featured_summary", _blank)
    resp = client.post("/api/collection/wishlist-featured-summary", json=_payload(), headers=AUTH_HEADERS)
    assert resp.status_code == 502


def test_wishlist_featured_summary_returns_502_on_exception(monkeypatch):
    async def _fail(_request):
        raise RuntimeError("boom")

    monkeypatch.setattr("app.routes.generate_wishlist_featured_summary", _fail)
    resp = client.post("/api/collection/wishlist-featured-summary", json=_payload(), headers=AUTH_HEADERS)
    assert resp.status_code == 502
