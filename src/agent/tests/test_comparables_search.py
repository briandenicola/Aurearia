"""Route and service tests for Quick Identify comparables search (#779)."""

from datetime import UTC, datetime
from decimal import Decimal

import pytest
from fastapi.testclient import TestClient

from app.main import app
from app.models.requests import ComparablesSearchRequest
from app.models.responses import ComparableListing, ComparablesSearchResponse
from app.teams.coin_search import search_comparables
from app.teams.specialist_contracts import DealerListing, ProviderAttempt, SpecialistResult

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}
OBSERVED_AT = datetime(2026, 9, 29, 17, 0, tzinfo=UTC)


def valid_payload(**overrides) -> dict:
    payload = {
        "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
        "query": "Domitian denarius",
        "search_terms": "Domitian denarius",
        "limit": 5,
        "dealer_search_sources": ["vcoins.com"],
    }
    payload.update(overrides)
    return payload


def dealer_listing(
    url: str,
    price: Decimal | None,
    currency: str | None,
    *,
    verification_state: str = "verified",
    availability: str = "available",
) -> DealerListing:
    return DealerListing(
        kind="dealer_listing",
        source_url=url,
        canonical_source_id=url,
        provider="vcoins.com",
        observed_at=OBSERVED_AT,
        confidence="high",
        verification_state=verification_state,
        title="Domitian Denarius",
        dealer_name="VCoins Dealer",
        listed_price=price,
        currency=currency,
        availability=availability,
    )


def test_comparables_rejects_unknown_fields():
    resp = client.post("/api/search/comparables", json=valid_payload(currency="USD"), headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_comparables_requires_internal_token():
    resp = client.post("/api/search/comparables", json=valid_payload())
    assert resp.status_code == 401


def test_comparables_rejects_invalid_body():
    resp = client.post("/api/search/comparables", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_comparables_rejects_blank_query():
    resp = client.post("/api/search/comparables", json=valid_payload(query=""), headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_comparables_returns_typed_response(monkeypatch):
    async def fake_search(_request):
        return ComparablesSearchResponse(
            listings=[
                ComparableListing(
                    source_url="https://dealer.example/item",
                    source_name="VCoins Dealer",
                    title="Domitian Denarius",
                    price=225,
                    currency="USD",
                    availability="available",
                    verification_state="verified",
                )
            ],
            warnings=[],
            partial=False,
        )

    monkeypatch.setattr("app.routes.search_comparables", fake_search)

    resp = client.post("/api/search/comparables", json=valid_payload(), headers=AUTH_HEADERS)

    assert resp.status_code == 200
    data = resp.json()
    assert data["listings"][0]["price"] == 225
    assert data["listings"][0]["availability"] == "available"
    assert data["partial"] is False


@pytest.mark.asyncio
async def test_search_comparables_maps_dealer_listings(monkeypatch):
    captured: dict = {}

    async def fake_market_search(query, **kwargs):
        captured["query"] = query
        captured["kwargs"] = kwargs
        return SpecialistResult(
            capability="market_search",
            outcome="complete",
            items=[
                dealer_listing("https://dealer.example/a", Decimal("180.00"), "USD"),
                dealer_listing("https://dealer.example/b", None, None),
            ],
            provider_attempts=[
                ProviderAttempt(
                    provider="vcoins.com",
                    status="success",
                    observed_at=OBSERVED_AT,
                    accepted_items=2,
                )
            ],
            warnings=[],
        )

    monkeypatch.setattr("app.teams.coin_search.run_market_search", fake_market_search)

    response = await search_comparables(ComparablesSearchRequest.model_validate(valid_payload()))

    assert captured["query"]["search_terms"] == "Domitian denarius"
    assert captured["kwargs"]["source_hosts"] == {"vcoins.com"}
    assert [listing.price for listing in response.listings] == [180.0, None]
    assert response.listings[0].source_name == "VCoins Dealer"
    assert response.listings[0].verification_state == "verified"
    assert response.partial is False


@pytest.mark.asyncio
async def test_search_comparables_reports_unverified_and_sold_state(monkeypatch):
    """Go drops these, so the mapper must never flatten them to "verified"."""

    async def fake_market_search(_query, **_kwargs):
        return SpecialistResult(
            capability="market_search",
            outcome="complete",
            items=[
                dealer_listing(
                    "https://dealer.example/never-fetched",
                    Decimal("300.00"),
                    "USD",
                    verification_state="partial",
                    availability="unknown",
                ),
                dealer_listing(
                    "https://dealer.example/gone",
                    Decimal("200.00"),
                    "USD",
                    availability="sold",
                ),
            ],
            provider_attempts=[],
            warnings=[],
        )

    monkeypatch.setattr("app.teams.coin_search.run_market_search", fake_market_search)

    response = await search_comparables(ComparablesSearchRequest.model_validate(valid_payload()))

    assert [listing.verification_state for listing in response.listings] == ["partial", "verified"]
    assert [listing.availability for listing in response.listings] == ["unknown", "sold"]


@pytest.mark.asyncio
async def test_search_comparables_fails_open_when_search_raises(monkeypatch):
    async def exploding_market_search(_query, **_kwargs):
        raise RuntimeError("dealer unreachable")

    monkeypatch.setattr("app.teams.coin_search.run_market_search", exploding_market_search)

    response = await search_comparables(ComparablesSearchRequest.model_validate(valid_payload()))

    assert response.listings == []
    assert response.partial is True
    assert response.warnings == ["Comparables search could not complete."]


@pytest.mark.asyncio
async def test_search_comparables_marks_partial_outcomes(monkeypatch):
    async def fake_market_search(_query, **_kwargs):
        return SpecialistResult(
            capability="market_search",
            outcome="partial",
            items=[dealer_listing("https://dealer.example/a", Decimal("180.00"), "USD")],
            provider_attempts=[
                ProviderAttempt(
                    provider="vcoins.com",
                    status="success",
                    observed_at=OBSERVED_AT,
                    accepted_items=1,
                )
            ],
            warnings=["Some listings are only partially verified; current availability may be unknown."],
        )

    monkeypatch.setattr("app.teams.coin_search.run_market_search", fake_market_search)

    response = await search_comparables(ComparablesSearchRequest.model_validate(valid_payload()))

    assert response.partial is True
    assert response.warnings
