"""Independent QA regression coverage for the wishlist-featured-summary
route (spec 354, Phase 7 / T033), owned by Brutus (Tester/QA).

This file is deliberately separate from
`tests/test_wishlist_featured_summary.py` (which already covers the basic
happy-path/422/502 route wiring) to avoid merge collisions while the team
works concurrently, and to add INTEGRATION-shaped coverage that only mocks
the external LLM boundary (`get_chat_model`) rather than the whole
`generate_wishlist_featured_summary` function — proving the real prompt
construction, trimming, and provider-selection logic actually run, per the
charter's "prioritize integration behavior over brittle mocks" mandate.

Frozen contract under test (plan.md "API Changes" + spec.md FR-025/026):
  POST /api/collection/wishlist-featured-summary
  Request:  { llm: {provider, api_key, model, ollama_url, searxng_url},
              coin: {name, era, category, denomination, ruler, mint,
                      obverse_analysis, reverse_analysis, ai_analysis},
              user_display_name }
  Response: { summary: str }  (bounded <=500 chars, no provider metadata,
              no invented facts, stateless, no DB access)
"""

from fastapi.testclient import TestClient

import app.teams.wishlist_featured_summary as wishlist_summary_module
from app.main import app

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


class _FakeChatModel:
    """Stand-in LLM boundary matching the pattern already established in
    test_deep_identification_graph_topology.py — an object with an async
    `ainvoke` returning something with a `.content` string attribute."""

    def __init__(self, content: str):
        self._content = content

    async def ainvoke(self, messages, **kwargs):
        return type("Resp", (), {"content": self._content})()


def _payload(provider: str = "anthropic", **coin_overrides) -> dict:
    coin = {
        "name": "Trajan Denarius",
        "era": "Roman Imperial",
        "category": "Roman",
        "denomination": "Denarius",
        "ruler": "Trajan",
        "mint": "Rome",
        "obverse_analysis": "Laureate bust of Trajan facing right.",
        "reverse_analysis": "Victory standing, holding wreath and palm.",
        "ai_analysis": "",
    }
    coin.update(coin_overrides)
    llm = {"provider": provider, "api_key": "test-key", "model": "test-model"}
    if provider == "ollama":
        llm = {"provider": "ollama", "ollama_url": "http://localhost:11434", "model": "llama3"}
    return {"llm": llm, "coin": coin, "user_display_name": "Brian"}


def _post(payload: dict):
    return client.post("/api/collection/wishlist-featured-summary", json=payload, headers=AUTH_HEADERS)


# --- Provider abstraction (Anthropic / Ollama) -----------------------------


def test_anthropic_provider_end_to_end_returns_bounded_non_empty_summary(monkeypatch):
    monkeypatch.setattr(
        wishlist_summary_module,
        "get_chat_model",
        lambda config: _FakeChatModel("This Trajan denarius would complement your Roman imperial holdings."),
    )
    resp = _post(_payload(provider="anthropic"))
    assert resp.status_code == 200
    body = resp.json()
    assert body["summary"]
    assert len(body["summary"]) <= 500
    # No provider identifiers should ever leak into client-facing text
    # (Principle V / NFR-005).
    assert "anthropic" not in body["summary"].lower()
    assert "test-key" not in body["summary"]


def test_ollama_provider_end_to_end_returns_bounded_non_empty_summary(monkeypatch):
    monkeypatch.setattr(
        wishlist_summary_module,
        "get_chat_model",
        lambda config: _FakeChatModel("A solid addition to a Trajanic-era Roman collection."),
    )
    resp = _post(_payload(provider="ollama"))
    assert resp.status_code == 200
    body = resp.json()
    assert body["summary"]
    assert len(body["summary"]) <= 500
    assert "ollama" not in body["summary"].lower()
    assert "localhost" not in body["summary"]


def test_ollama_and_anthropic_share_identical_response_shape(monkeypatch):
    """FR-026: the route is provider-agnostic; the response envelope must
    be byte-identical in shape (just `{summary: str}`) regardless of which
    provider the user has configured."""
    monkeypatch.setattr(wishlist_summary_module, "get_chat_model", lambda config: _FakeChatModel("A fine coin."))
    anthropic_keys = set(_post(_payload(provider="anthropic")).json().keys())
    ollama_keys = set(_post(_payload(provider="ollama")).json().keys())
    assert anthropic_keys == ollama_keys == {"summary"}


# --- Bounds / truncation ----------------------------------------------------


def test_overlong_model_response_is_truncated_to_500_chars(monkeypatch):
    overlong = "This coin is remarkable. " * 40  # well over 500 chars
    assert len(overlong) > 500
    monkeypatch.setattr(wishlist_summary_module, "get_chat_model", lambda config: _FakeChatModel(overlong))
    resp = _post(_payload())
    assert resp.status_code == 200
    assert len(resp.json()["summary"]) <= 500


def test_model_response_with_newlines_is_flattened_to_single_line(monkeypatch):
    multiline = "Line one about the coin.\nLine two about the coin.\r\nLine three."
    monkeypatch.setattr(wishlist_summary_module, "get_chat_model", lambda config: _FakeChatModel(multiline))
    resp = _post(_payload())
    assert resp.status_code == 200
    summary = resp.json()["summary"]
    assert "\n" not in summary
    assert "\r" not in summary


def test_whitespace_only_model_response_returns_502_not_a_blank_summary(monkeypatch):
    monkeypatch.setattr(wishlist_summary_module, "get_chat_model", lambda config: _FakeChatModel("   \n\n   "))
    resp = _post(_payload())
    assert resp.status_code == 502
    # SC-005 / D8: an empty AI response must fall back on the Go side, never
    # surface a blank/whitespace summary as if it were a real 200.
    assert "summary" not in resp.json() or not resp.json().get("summary")


# --- Validation / bounds on required fields ---------------------------------


def test_missing_coin_name_is_rejected_with_422():
    payload = _payload()
    del payload["coin"]["name"]
    resp = _post(payload)
    assert resp.status_code == 422


def test_missing_llm_config_is_rejected_with_422():
    payload = _payload()
    del payload["llm"]
    resp = _post(payload)
    assert resp.status_code == 422


def test_unknown_extra_field_is_rejected_strict_schema():
    """StrictRequestModel-derived schemas forbid extra fields — proves this
    route did not silently loosen validation for the new contract."""
    payload = _payload()
    payload["coin"]["unexpected_field"] = "sneaky"
    resp = _post(payload)
    assert resp.status_code == 422


# --- Missing internal auth (Principle V) ------------------------------------


def test_missing_internal_service_token_is_rejected():
    resp = client.post("/api/collection/wishlist-featured-summary", json=_payload())
    assert resp.status_code == 401


# --- No-DB-access / statelessness sanity ------------------------------------


def test_route_module_has_no_database_imports():
    """Principle I: the Python agent is stateless and never touches the DB.
    A crude but effective regression: no sqlalchemy/sqlite/database import
    anywhere in the wishlist-summary module's source."""
    import inspect

    source = inspect.getsource(wishlist_summary_module)
    lowered = source.lower()
    for forbidden in ("sqlalchemy", "sqlite3", "import psycopg", "from app.database", "import database"):
        assert forbidden not in lowered, (
            f"found forbidden DB-access token {forbidden!r} "
            "in wishlist_featured_summary module"
        )
