"""Basic tests for the agent service.

These tests verify endpoint contracts and request validation.
Integration tests requiring a live LLM belong in a separate suite.
"""

from fastapi.testclient import TestClient

from app.config import settings
from app.main import app

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


def test_health():
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "ok"
    assert data["service"] == "agent"


def test_ready():
    resp = client.get("/ready")
    assert resp.status_code == 200
    assert resp.json()["status"] == "ready"


def test_ready_reports_missing_internal_service_credential():
    original = settings.internal_service_token
    settings.internal_service_token = ""
    try:
        resp = client.get("/ready")
    finally:
        settings.internal_service_token = original

    assert resp.status_code == 503
    assert resp.json()["detail"] == (
        "Internal service credential is not configured; set AGENT_INTERNAL_SERVICE_TOKEN"
    )


def test_agent_api_requires_internal_token():
    resp = client.post("/api/search/coins", json={})
    assert resp.status_code == 401


def test_logs_requires_internal_token():
    resp = client.get("/logs")
    assert resp.status_code == 401


def test_log_level_requires_internal_token():
    resp = client.put("/log-level", json={"level": "INFO"})
    assert resp.status_code == 401


def test_internal_token_missing_config_returns_clear_503(monkeypatch):
    monkeypatch.setattr(settings, "internal_service_token", "")

    resp = client.get("/logs", headers=AUTH_HEADERS)

    assert resp.status_code == 503
    assert resp.json() == {"detail": "Internal service credential is not configured"}


def test_configured_internal_token_allows_go_proxy_header(monkeypatch):
    monkeypatch.setattr(settings, "internal_service_token", "go-proxy-token")

    resp = client.get("/logs", headers={"X-Internal-Service-Token": "go-proxy-token"})

    assert resp.status_code == 200
    assert "logs" in resp.json()


def test_logs_allow_go_mediated_internal_token():
    resp = client.get("/logs", headers=AUTH_HEADERS)
    assert resp.status_code == 200
    assert "logs" in resp.json()


def test_search_coins_rejects_invalid_body():
    resp = client.post("/api/search/coins", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_search_coins_rejects_missing_message():
    resp = client.post(
        "/api/search/coins",
        json={
            "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
            "user": {"user_id": 1},
        },
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 422


def test_search_shows_rejects_invalid_body():
    resp = client.post("/api/search/shows", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_analyze_stub():
    resp = client.post(
        "/api/analyze",
        json={
            "llm": {"provider": "ollama", "ollama_url": "http://localhost:11434", "model": "llava"},
            "coin": {"id": 1, "name": "Test Coin"},
        },
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 200
    data = resp.json()
    assert "message" in data


def test_analyze_anthropic_ignores_non_ollama_url():
    resp = client.post(
        "/api/analyze",
        json={
            "llm": {
                "provider": "anthropic",
                "api_key": "k",
                "model": "claude-opus-4-8",
                "ollama_url": "https://ai.denicolafamily.com",
            },
            "coin": {"id": 1, "name": "Test Coin"},
        },
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 200


def test_analyze_raw_format_returns_unformatted_analysis(monkeypatch):
    import app.routes as routes

    captured = {}

    class FakeGraph:
        async def ainvoke(self, _state):
            return {
                "raw_analysis": '{"name":"Julia Domna Denarius"}',
                "formatted_analysis": "Formatted narrative should not be returned",
                "messages": [],
            }

    def fake_create_coin_analysis_team(**kwargs):
        captured.update(kwargs)
        return FakeGraph()

    monkeypatch.setattr(routes, "create_coin_analysis_team", fake_create_coin_analysis_team)

    resp = client.post(
        "/api/analyze",
        json={
            "llm": {"provider": "anthropic", "api_key": "k", "model": "claude-opus-4-8"},
            "coin": {"id": 1, "name": "Lookup Candidate"},
            "format_output": False,
        },
        headers=AUTH_HEADERS,
    )

    assert resp.status_code == 200
    assert resp.json()["analysis"] == '{"name":"Julia Domna Denarius"}'
    assert captured["format_output"] is False


def test_search_coins_anthropic_accepts_stale_urls_and_unrelated_callback(monkeypatch):
    captured = {}

    def fake_create_supervisor(llm, **kwargs):
        captured["llm"] = llm
        captured.update(kwargs)
        return object()

    async def fake_stream_graph_events(_graph, _state, config=None):
        captured["config"] = config
        yield 'data: {"type":"done","message":"ok"}\n\n'

    monkeypatch.setattr("app.routes.create_supervisor", fake_create_supervisor)
    monkeypatch.setattr("app.routes.stream_graph_events", fake_stream_graph_events)
    monkeypatch.setattr(settings, "trusted_outbound_origins", "http://app:8080")
    monkeypatch.setattr(settings, "allow_local_outbound", False)

    resp = client.post(
        "/api/search/coins",
        json={
            "llm": {
                "provider": "anthropic",
                "api_key": "k",
                "model": "claude-opus-4-8",
                "ollama_url": "https://stale-ollama.example.com",
                "searxng_url": "https://stale-search.example.com",
            },
            "user": {"user_id": 1},
            "message": "Find Athenian owls for sale",
            "tools_base_url": "https://unrelated-callback.example.com",
            "internal_token": "token",
        },
        headers=AUTH_HEADERS,
    )

    assert resp.status_code == 200
    assert "ok" in resp.text
    assert captured["llm"].ollama_url == ""
    assert captured["llm"].searxng_url == ""
    assert captured["tools_base_url"] == "https://unrelated-callback.example.com"


def test_portfolio_review_rejects_invalid_body():
    resp = client.post("/api/portfolio/review", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_intake_draft_rejects_invalid_body():
    resp = client.post("/api/intake/draft", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422
