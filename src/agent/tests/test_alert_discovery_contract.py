"""Contract tests for stateless wishlist search alert discovery DTOs."""

import pytest
from pydantic import ValidationError

from app.models.requests import MAX_ALERT_CANDIDATES, AlertDiscoveryRequest
from app.models.responses import AlertDiscoveryCandidate, AlertDiscoveryProvenance
from app.teams.coin_search import _filter_allowed_fetch_urls, _trusted_alert_fetch_hosts, _url_matches_allowed_hosts


def valid_request_payload() -> dict:
    return {
        "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
        "alert": {
            "alert_id": 1,
            "max_candidates": 20,
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


def test_alert_discovery_request_forbids_extra_fields():
    payload = valid_request_payload()
    payload["alert"]["unexpected"] = "drift"

    with pytest.raises(ValidationError):
        AlertDiscoveryRequest(**payload)


def test_alert_discovery_request_caps_max_candidates():
    payload = valid_request_payload()
    payload["alert"]["max_candidates"] = MAX_ALERT_CANDIDATES + 1

    with pytest.raises(ValidationError):
        AlertDiscoveryRequest(**payload)


def test_alert_discovery_request_validates_ranges():
    payload = valid_request_payload()
    payload["alert"]["criteria_snapshot"]["price_min"] = 400

    with pytest.raises(ValidationError):
        AlertDiscoveryRequest(**payload)


def test_alert_discovery_candidate_requires_source_backed_fields():
    provenance = AlertDiscoveryProvenance(
        field="source_url",
        value="https://dealer.example/item",
        source_url="https://dealer.example/item",
        observed_at="2026-06-29T17:00:00Z",
        confidence="high",
        verification_state="verified",
    )

    candidate = AlertDiscoveryCandidate(
        source_url="https://dealer.example/item",
        title="Domitian Denarius",
        reason_for_match="Title matched the alert.",
        last_seen_at="2026-06-29T17:00:00Z",
        provenance_status="verified",
        provenance=[provenance],
    )

    assert candidate.observed_price is None
    assert candidate.source_name == ""


def test_alert_discovery_candidate_rejects_missing_provenance():
    with pytest.raises(ValidationError):
        AlertDiscoveryCandidate(
            source_url="https://dealer.example/item",
            title="Domitian Denarius",
            reason_for_match="Title matched the alert.",
            last_seen_at="2026-06-29T17:00:00Z",
            provenance_status="verified",
            provenance=[],
        )


def test_alert_discovery_fetch_allowlist_rejects_untrusted_source_filter():
    assert _trusted_alert_fetch_hosts(["attacker.example"]) == set()
    assert not _url_matches_allowed_hosts("https://attacker.example/listing", {"vcoins.com"})


def test_alert_discovery_fetch_allowlist_accepts_trusted_dealer_subdomain():
    allowed = _trusted_alert_fetch_hosts(["www.vcoins.com"])

    assert allowed == {"vcoins.com"}
    assert _url_matches_allowed_hosts("https://stores.vcoins.com/item", allowed)


def test_alert_discovery_empty_fetch_allowlist_blocks_all_urls():
    urls = ["https://attacker.example/listing", "https://vcoins.com/item"]

    assert _filter_allowed_fetch_urls(urls, None) == urls
    assert _filter_allowed_fetch_urls(urls, set()) == []
    assert _filter_allowed_fetch_urls(urls, {"vcoins.com"}) == ["https://vcoins.com/item"]
