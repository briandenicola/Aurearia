"""Route tests for stateless wishlist search alert discovery."""

from fastapi.testclient import TestClient

from app.main import app
from app.models.responses import AlertDiscoveryCandidate, AlertDiscoveryProvenance, AlertDiscoveryResponse

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


def valid_payload(max_candidates: int = 20) -> dict:
    return {
        "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
        "alert": {
            "alert_id": 1,
            "max_candidates": max_candidates,
            "criteria_snapshot": {
                "name": "Domitian denarius",
                "ruler_or_issuer": "Domitian",
                "coin_type": "Denarius",
                "price_max": 300,
                "currency": "USD",
                "source_filters": ["vcoins.com"],
            },
        },
    }


def test_search_alerts_requires_internal_token():
    resp = client.post("/api/search/alerts", json=valid_payload())
    assert resp.status_code == 401


def test_search_alerts_rejects_invalid_body():
    resp = client.post("/api/search/alerts", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_search_alerts_returns_typed_response(monkeypatch):
    async def fake_discover(_request):
        return AlertDiscoveryResponse(
            candidates=[
                AlertDiscoveryCandidate(
                    source_url="https://dealer.example/item",
                    title="Domitian Denarius",
                    observed_price=225,
                    observed_currency="USD",
                    reason_for_match="Title matched the alert.",
                    last_seen_at="2026-06-29T17:00:00Z",
                    provenance_status="verified",
                    provenance=[
                        AlertDiscoveryProvenance(
                            field="source_url",
                            value="https://dealer.example/item",
                            source_url="https://dealer.example/item",
                            observed_at="2026-06-29T17:00:00Z",
                            confidence="high",
                            verification_state="verified",
                        )
                    ],
                )
            ],
            warnings=[],
            partial=False,
        )

    monkeypatch.setattr("app.routes.discover_alert_candidates", fake_discover)

    resp = client.post("/api/search/alerts", json=valid_payload(), headers=AUTH_HEADERS)

    assert resp.status_code == 200
    data = resp.json()
    assert data["candidates"][0]["source_url"] == "https://dealer.example/item"
    assert data["candidates"][0]["provenance"][0]["verification_state"] == "verified"
    assert data["partial"] is False
