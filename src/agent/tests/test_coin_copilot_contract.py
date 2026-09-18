"""Shared Go/Python Coin Copilot contract fixture validation."""

import hashlib
import json
from copy import deepcopy
from pathlib import Path

import pytest
from pydantic import ValidationError

from app.models.requests import CopilotExecuteRequest
from app.models.responses import CopilotExecutionFrame
from app.teams.specialist_contracts import (
    PriceTrendSummary,
    SpecialistQuery,
    SpecialistResult,
    TruncationMetadata,
)
from app.tools.copilot_collection_tools import bound_tool_result

FIXTURES = Path(__file__).parent / "fixtures" / "coin_copilot"
SPECIALIST_FIXTURES = FIXTURES / "specialists"
INVALID_SPECIALIST_FIXTURES = FIXTURES / "specialists_invalid"


def _load(name: str) -> dict:
    return json.loads((FIXTURES / name).read_text(encoding="utf-8"))


def _load_specialist(name: str) -> dict:
    return json.loads((SPECIALIST_FIXTURES / name).read_text(encoding="utf-8"))


def _load_invalid_specialist(name: str) -> dict:
    return json.loads((INVALID_SPECIALIST_FIXTURES / name).read_text(encoding="utf-8"))


def _load_adversarial_specialist(name: str) -> dict:
    case = _load_invalid_specialist(name)
    payload = deepcopy(_load_specialist(case["base_fixture"]))
    for old, new in case.get("replacements", {}).items():
        payload = json.loads(json.dumps(payload).replace(old, new))
    for change in case.get("changes", []):
        target = payload
        for part in change["path"][:-1]:
            target = target[part]
        target[change["path"][-1]] = change["value"]
    return payload


def _bounded_fallback() -> dict:
    return {
        "truncated": True,
        "original_bytes": 65536,
        "digest": "a" * 64,
        "summary": "Tool result exceeded the persisted-result limit.",
    }


def test_valid_execute_request_fixture_is_strict_and_current():
    request = CopilotExecuteRequest.model_validate(_load("valid_execute_request.json"))
    assert request.schema_version == 1
    assert request.limits.max_iterations == 8
    assert request.limits.max_tool_calls == 12
    assert request.limits.max_concurrent_tools == 3
    assert request.limits.hard_timeout_seconds == 120
    assert request.limits.max_persisted_tool_result_bytes == 32768
    assert request.app_context is not None
    assert request.app_context.active_coin_id is None


def test_concurrent_tool_limit_accepts_five_and_rejects_six():
    payload = _load("valid_execute_request.json")
    payload["limits"]["max_concurrent_tools"] = 5
    assert CopilotExecuteRequest.model_validate(payload).limits.max_concurrent_tools == 5
    payload["limits"]["max_concurrent_tools"] = 6
    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)


def test_hard_timeout_accepts_release_maximum_and_rejects_higher_value():
    payload = _load("valid_execute_request.json")
    payload["limits"]["hard_timeout_seconds"] = 150
    assert CopilotExecuteRequest.model_validate(payload).limits.hard_timeout_seconds == 150
    payload["limits"]["hard_timeout_seconds"] = 151
    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)


def test_request_accepts_exact_go_serialized_app_context():
    payload = _load("valid_execute_request.json")
    payload["app_context"] = {
        "route": "/coin/42",
        "activeCoinId": 42,
    }

    request = CopilotExecuteRequest.model_validate(payload)

    assert request.app_context is not None
    assert request.app_context.active_coin_id == 42


def test_request_rejects_python_field_name_at_strict_json_boundary():
    payload = _load("valid_execute_request.json")
    payload["app_context"] = {
        "route": "/coin/42",
        "active_coin_id": 42,
    }

    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)


@pytest.mark.parametrize(
    "fixture",
    [
        "valid_checkpoint_frame.json",
        "valid_tool_completed_frame.json",
        "valid_completed_frame.json",
    ],
)
def test_valid_frame_fixtures_parse(fixture):
    frame = CopilotExecutionFrame.model_validate(_load(fixture))
    assert frame.run_id == "ccr_fixture"


@pytest.mark.parametrize(
    ("fixture", "model"),
    [
        ("invalid_execute_extra_field.json", CopilotExecuteRequest),
        ("invalid_frame_reasoning.json", CopilotExecutionFrame),
        ("invalid_checkpoint_duplicate_call_id.json", CopilotExecutionFrame),
    ],
)
def test_invalid_contract_fixtures_fail_closed(fixture, model):
    with pytest.raises(ValidationError):
        model.model_validate(_load(fixture))


def test_request_rejects_unknown_and_duplicate_allowed_tools():
    payload = _load("valid_execute_request.json")
    payload["allowed_tools"] = ["collection_summary", "web_search"]
    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)

    payload["allowed_tools"] = ["collection_summary", "collection_summary"]
    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)


def test_request_rejects_checkpoint_over_budget():
    payload = _load("valid_execute_request.json")
    payload["checkpoint"]["counters"]["tool_calls"] = 13
    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)


@pytest.mark.parametrize(
    ("path", "field"),
    [
        (("limits",), "max_estimated_cost_micros"),
        (("checkpoint", "counters"), "estimated_cost_micros"),
    ],
)
def test_request_rejects_removed_cost_fields(path, field):
    payload = _load("valid_execute_request.json")
    target = payload
    for part in path:
        target = target[part]
    target[field] = 1

    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)


def test_usage_keeps_reliable_token_counts_without_cost():
    frame = CopilotExecutionFrame.model_validate(_load("valid_completed_frame.json"))

    assert frame.payload.usage.input_tokens == 100
    assert frame.payload.usage.output_tokens == 30
    assert "estimated_cost_micros" not in frame.payload.usage.model_dump()


def test_frame_rejects_unknown_tool():
    payload = _load("valid_tool_completed_frame.json")
    payload["payload"]["tool_name"] = "web_search"
    with pytest.raises(ValidationError):
        CopilotExecutionFrame.model_validate(payload)


def test_specialist_checkpoint_and_frame_results_are_discriminated():
    specialist_result = _load_specialist("market_search_complete.json")
    request_payload = _load("valid_execute_request.json")
    request_payload["allowed_tools"] = [
        "search_my_collection",
        "get_coin",
        "collection_summary",
        "top_coins_by_value",
        "portfolio_review",
        "gap_analysis",
        "market_search",
        "auction_search",
        "price_trends",
        "similar_lots",
    ]
    request_payload["checkpoint"]["completed_tools"] = [
        {
            "tool_call_id": "call_market",
            "tool_name": "market_search",
            "result_digest": "a" * 64,
            "result": specialist_result,
            "original_bytes": 1024,
            "persisted_bytes": 1024,
            "truncated": False,
        }
    ]
    request_payload["checkpoint"]["counters"]["tool_calls"] = 1

    request = CopilotExecuteRequest.model_validate(request_payload)
    completed = request.checkpoint.completed_tools[0]

    assert isinstance(completed.result, SpecialistResult)
    assert completed.result.capability == "market_search"

    frame_payload = _load("valid_tool_completed_frame.json")
    frame_payload["payload"]["tool_name"] = "market_search"
    frame_payload["payload"]["result"] = specialist_result
    frame = CopilotExecutionFrame.model_validate(frame_payload)

    assert isinstance(frame.payload.result, SpecialistResult)
    assert frame.payload.result.capability == "market_search"


@pytest.mark.parametrize("container", ["checkpoint", "frame"])
def test_specialist_tool_result_capability_mismatch_fails_closed(container):
    specialist_result = _load_specialist("auction_search_complete.json")
    if container == "checkpoint":
        payload = _load("valid_execute_request.json")
        payload["allowed_tools"].append("market_search")
        payload["checkpoint"]["completed_tools"] = [
            {
                "tool_call_id": "call_market",
                "tool_name": "market_search",
                "result_digest": "a" * 64,
                "result": specialist_result,
                "original_bytes": 1024,
                "persisted_bytes": 1024,
                "truncated": False,
            }
        ]
        payload["checkpoint"]["counters"]["tool_calls"] = 1
        model = CopilotExecuteRequest
    else:
        payload = _load("valid_tool_completed_frame.json")
        payload["payload"]["tool_name"] = "market_search"
        payload["payload"]["result"] = specialist_result
        model = CopilotExecutionFrame

    with pytest.raises(ValidationError, match="capability does not match"):
        model.model_validate(payload)


@pytest.mark.parametrize("container", ["checkpoint", "frame"])
def test_specialist_result_cannot_impersonate_bounded_fallback(container):
    specialist_result = _load_specialist("auction_search_complete.json")
    specialist_result["truncated"] = True
    if container == "checkpoint":
        payload = _load("valid_execute_request.json")
        payload["allowed_tools"].append("market_search")
        payload["checkpoint"]["completed_tools"] = [
            {
                "tool_call_id": "call_market",
                "tool_name": "market_search",
                "result_digest": "a" * 64,
                "result": specialist_result,
                "original_bytes": 1024,
                "persisted_bytes": 1024,
                "truncated": True,
            }
        ]
        payload["checkpoint"]["counters"]["tool_calls"] = 1
        model = CopilotExecuteRequest
    else:
        payload = _load("valid_tool_completed_frame.json")
        payload["payload"]["tool_name"] = "market_search"
        payload["payload"]["result"] = specialist_result
        model = CopilotExecutionFrame

    with pytest.raises(ValidationError):
        model.model_validate(payload)


@pytest.mark.parametrize("container", ["checkpoint", "frame"])
def test_specialist_result_accepts_exact_feature_359_bounded_fallback(container):
    fallback = _bounded_fallback()
    if container == "checkpoint":
        payload = _load("valid_execute_request.json")
        payload["allowed_tools"].append("market_search")
        payload["checkpoint"]["completed_tools"] = [
            {
                "tool_call_id": "call_market",
                "tool_name": "market_search",
                "result_digest": fallback["digest"],
                "result": fallback,
                "original_bytes": fallback["original_bytes"],
                "persisted_bytes": 180,
                "truncated": True,
            }
        ]
        payload["checkpoint"]["counters"]["tool_calls"] = 1
        result = CopilotExecuteRequest.model_validate(payload).checkpoint.completed_tools[0].result
    else:
        payload = _load("valid_tool_completed_frame.json")
        payload["payload"]["tool_name"] = "market_search"
        payload["payload"]["result"] = fallback
        result = CopilotExecutionFrame.model_validate(payload).payload.result

    assert isinstance(result, dict)
    assert result == fallback


def test_existing_feature_359_checkpoint_and_tool_frame_results_remain_dicts():
    checkpoint = CopilotExecutionFrame.model_validate(_load("valid_checkpoint_frame.json"))
    completed_frame = CopilotExecutionFrame.model_validate(_load("valid_tool_completed_frame.json"))

    assert isinstance(checkpoint.payload.completed_tools[0].result, dict)
    assert isinstance(completed_frame.payload.result, dict)


@pytest.mark.parametrize(
    "capability",
    ["market_search", "auction_search", "price_trends", "similar_lots"],
)
def test_specialist_query_fixtures_are_strict_and_bounded(capability):
    query = SpecialistQuery.model_validate(_load_specialist(f"{capability}_input.json"))

    assert query.query == "Domitian denarius Minerva"
    assert query.limit == 5

    with pytest.raises(ValidationError):
        SpecialistQuery.model_validate({"query": "x", "limit": 5, "provider": "arbitrary"})
    with pytest.raises(ValidationError):
        SpecialistQuery.model_validate({"query": "", "limit": 5})
    with pytest.raises(ValidationError):
        SpecialistQuery.model_validate({"query": "x" * 501, "limit": 5})
    with pytest.raises(ValidationError):
        SpecialistQuery.model_validate({"query": "x", "limit": 0})
    with pytest.raises(ValidationError):
        SpecialistQuery.model_validate({"query": "x", "limit": 11})


@pytest.mark.parametrize(
    ("capability", "expected_kind"),
    [
        ("market_search", "dealer_listing"),
        ("auction_search", "auction_lot"),
        ("price_trends", "sale_observation"),
        ("similar_lots", "similar_lot"),
    ],
)
@pytest.mark.parametrize("outcome", ["complete", "partial", "no_match", "unavailable"])
def test_specialist_result_fixtures_cover_capabilities_and_outcomes(
    capability,
    expected_kind,
    outcome,
):
    result = SpecialistResult.model_validate(_load_specialist(f"{capability}_{outcome}.json"))

    assert result.capability == capability
    assert result.outcome == outcome
    assert all(item.kind == expected_kind for item in result.items)
    assert len(result.items) <= 10
    assert len(result.provider_attempts) <= 10
    assert len(result.warnings) <= 10
    assert len({item.canonical_source_id for item in result.items}) == len(result.items)
    for item in result.items:
        assert item.source_url.startswith("https://")
        assert item.provenance
        assert all(entry.source_url == item.source_url for entry in item.provenance)


def test_market_search_dealer_listing_preserves_typed_provenance():
    result = SpecialistResult.model_validate(_load_specialist("market_search_complete.json"))
    item = result.items[0]

    assert item.kind == "dealer_listing"
    assert item.dealer_name == "Classical Numismatic Group"
    assert item.listed_price == 275
    assert item.currency == "USD"
    assert {entry.field for entry in item.provenance} >= {
        "title",
        "dealer_name",
        "listed_price",
        "currency",
        "availability",
    }


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("items", 0, "title"), "x" * 301),
        (("items", 0, "description"), "x" * 501),
        (("items", 0, "provider"), "x" * 65),
        (("items", 0, "dealer_name"), "x" * 301),
        (("items", 0, "listed_price"), -1),
        (("warnings",), ["warning"] * 11),
        (("warnings",), ["x" * 501]),
        (("provider_attempts", 0, "accepted_items"), 11),
        (
            ("provider_attempts",),
            [
                {
                    "provider": f"provider_{index}",
                    "status": "success",
                    "observed_at": "2026-09-18T12:00:00Z",
                    "accepted_items": 1,
                    "warning_code": None,
                }
                for index in range(11)
            ],
        ),
    ],
)
def test_market_search_declared_result_bounds_fail_closed(path, value):
    payload = _load_specialist("market_search_complete.json")
    target = payload
    for part in path[:-1]:
        target = target[part]
    target[path[-1]] = value

    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(payload)


def test_market_search_source_url_bound_fails_closed():
    payload = _load_specialist("market_search_complete.json")
    old_url = payload["items"][0]["source_url"]
    long_url = "https://www.cngcoins.com/" + ("x" * 2024)
    payload = json.loads(json.dumps(payload).replace(old_url, long_url))

    assert len(long_url) > 2048
    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(payload)


def test_evidence_provenance_accepts_20_entries_and_rejects_21():
    payload = _load_specialist("auction_search_complete.json")
    item = payload["items"][0]
    item["description"] = "Source-backed description"
    item["current_bid"] = 200
    provenance_fields = [
        "title",
        "description",
        "auction_house",
        "sale_name",
        "lot_number",
        "sale_date",
        "estimate",
        "current_bid",
        "currency",
        "lot_status",
        "ruler",
        "denomination",
        "era",
        "material",
        "kind",
        "source_url",
        "canonical_source_id",
        "provider",
        "observed_at",
        "confidence",
        "verification_state",
    ]
    entry = item["provenance"][0]
    item["provenance"] = [{**entry, "field": field} for field in provenance_fields[:20]]

    SpecialistResult.model_validate(payload)

    item["provenance"].append({**entry, "field": provenance_fields[20]})
    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(payload)


def test_canonical_source_id_accepts_2048_characters_and_rejects_2049():
    payload = _load_specialist("market_search_complete.json")
    prefix = "https://www.cngcoins.com/"
    payload["items"][0]["canonical_source_id"] = prefix + ("x" * (2048 - len(prefix)))

    SpecialistResult.model_validate(payload)

    payload["items"][0]["canonical_source_id"] += "x"
    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(payload)


def test_price_trends_result_preserves_comparable_typed_summary():
    result = SpecialistResult.model_validate(_load_specialist("price_trends_complete.json"))

    assert result.trend is not None
    assert result.trend.state == "rising"
    assert result.trend.sample_size == 3
    assert result.trend.currency == "USD"
    assert result.trend.price_basis == "hammer"
    assert result.trend.low == 210
    assert result.trend.median == 260
    assert result.trend.high == 325
    assert set(result.trend.supporting_source_ids) == {item.canonical_source_id for item in result.items}


@pytest.mark.parametrize(
    ("field", "value"),
    [
        ("limitations", ["limitation"] * 11),
        ("limitations", ["x" * 501]),
        ("sample_size", 11),
    ],
)
def test_price_trend_declared_summary_bounds_fail_closed(field, value):
    payload = _load_specialist("price_trends_complete.json")["trend"]
    payload[field] = value

    with pytest.raises(ValidationError):
        PriceTrendSummary.model_validate(payload)


def test_price_trend_supporting_sources_accept_10_and_reject_11():
    payload = _load_specialist("price_trends_complete.json")["trend"]
    payload["sample_size"] = 10
    payload["supporting_source_ids"] = [
        f"https://www.numisbids.com/n.php?p=lot&sid=7100&lot={index}" for index in range(10)
    ]

    PriceTrendSummary.model_validate(payload)

    payload["supporting_source_ids"].append("https://www.numisbids.com/n.php?p=lot&sid=7100&lot=10")
    with pytest.raises(ValidationError):
        PriceTrendSummary.model_validate(payload)


@pytest.mark.parametrize(
    ("path", "value"),
    [
        (("items", 0, "amount"), -1),
        (("items", 0, "currency"), "US"),
        (("trend", "low"), -1),
    ],
)
def test_price_trend_sale_observation_bounds_fail_closed(path, value):
    payload = _load_specialist("price_trends_complete.json")
    target = payload
    for part in path[:-1]:
        target = target[part]
    target[path[-1]] = value

    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(payload)


@pytest.mark.parametrize(
    "fixture",
    [
        "price_trend_range_mismatch.json",
        "price_trend_median_mismatch.json",
        "price_trend_coverage_mismatch.json",
        "price_trend_mixed_currency.json",
        "price_trend_mixed_basis.json",
        "price_trend_insufficient_evidence.json",
        "price_trend_direction_mismatch.json",
    ],
)
def test_price_trend_adversarial_fixtures_reject_unsupported_derivations(fixture):
    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(_load_adversarial_specialist(fixture))


@pytest.mark.parametrize(
    ("amounts", "state"),
    [
        ([210, 260, 325], "rising"),
        ([325, 260, 210], "declining"),
        ([250, 250, 250], "stable"),
    ],
)
def test_price_trend_direction_is_derived_chronologically(amounts, state):
    payload = _load_specialist("price_trends_complete.json")
    for item, amount in zip(payload["items"], amounts, strict=True):
        item["amount"] = amount
    payload["trend"].update(
        {
            "state": state,
            "low": min(amounts),
            "median": sorted(amounts)[1],
            "high": max(amounts),
        }
    )
    payload["items"].reverse()

    SpecialistResult.model_validate(payload)


def test_price_trend_rejects_direction_contradicting_observations():
    payload = _load_adversarial_specialist("price_trend_direction_mismatch.json")

    with pytest.raises(
        ValidationError,
        match="trend direction must be derived from chronologically ordered observations",
    ):
        SpecialistResult.model_validate(payload)


def test_similar_lot_lists_accept_declared_boundaries():
    payload = _load_specialist("similar_lots_complete.json")
    item = payload["items"][0]
    item["matched_attributes"] = [f"match {index}" for index in range(20)]
    item["material_differences"] = [f"difference {index}" for index in range(20)]

    SpecialistResult.model_validate(payload)

    item["material_differences"] = []
    SpecialistResult.model_validate(payload)


@pytest.mark.parametrize(
    ("field", "value"),
    [
        ("matched_attributes", []),
        ("matched_attributes", [f"match {index}" for index in range(21)]),
        ("material_differences", [f"difference {index}" for index in range(21)]),
    ],
)
def test_similar_lot_lists_reject_values_outside_declared_bounds(field, value):
    payload = _load_specialist("similar_lots_complete.json")
    payload["items"][0][field] = value

    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(payload)


@pytest.mark.parametrize(
    "digest",
    [
        "a" * 63,
        "a" * 65,
        "A" * 64,
        "g" * 64,
    ],
)
def test_truncation_digest_rejects_non_sha256_shape(digest):
    payload = _load_specialist("market_search_complete.json")["truncation"]
    payload["digest"] = digest

    with pytest.raises(ValidationError):
        TruncationMetadata.model_validate(payload)


def test_truncation_accepts_sha256_digest_and_rejects_negative_omitted_items():
    payload = _load_specialist("market_search_complete.json")["truncation"]

    TruncationMetadata.model_validate(payload)

    payload["omitted_items"] = -1
    with pytest.raises(ValidationError):
        TruncationMetadata.model_validate(payload)


def test_oversized_result_uses_deterministic_32_kib_digest_fallback():
    source = {"items": [{"title": "x" * 40000}]}
    canonical = json.dumps(source, separators=(",", ":"), sort_keys=True).encode()

    first = bound_tool_result(source, 32768)
    second = bound_tool_result(source, 32768)

    assert first == second
    bounded, original_bytes, truncated, digest = first
    assert truncated is True
    assert original_bytes == len(canonical)
    assert digest == hashlib.sha256(canonical).hexdigest()
    assert bounded == {
        "truncated": True,
        "original_bytes": original_bytes,
        "digest": digest,
        "summary": "Tool result exceeded the persisted-result limit.",
    }
    assert len(json.dumps(bounded, separators=(",", ":"), sort_keys=True).encode()) <= 32768


@pytest.mark.parametrize(
    "capability",
    ["market_search", "auction_search", "price_trends", "similar_lots"],
)
def test_specialist_public_event_fixtures_are_additive_and_sanitized(capability):
    event = _load_specialist(f"{capability}_event.json")
    payload = event["payload"]
    specialist_result = payload["specialistResult"]

    assert event["type"] == "tool_completed"
    assert payload["toolName"] == capability
    assert specialist_result["capability"] == capability
    assert specialist_result["outcome"] == "complete"
    assert "result" not in payload
    assert "provider_attempts" not in specialist_result
    assert "query" not in specialist_result
    assert "originalBytes" in specialist_result["truncation"]
    assert "original_bytes" not in specialist_result["truncation"]
    assert all(item["sourceUrl"].startswith("https://") for item in specialist_result["items"])
    if specialist_result["trend"] is not None:
        assert "sampleSize" in specialist_result["trend"]
        assert "sample_size" not in specialist_result["trend"]


@pytest.mark.parametrize(
    "fixture",
    [
        "unknown_query_field.json",
        "invalid_enum.json",
        "missing_provenance.json",
        "unsafe_url_http.json",
        "unsafe_url_credentials.json",
        "unsafe_url_private.json",
        "duplicate_identity.json",
        "conflicting_identity.json",
        "oversized_items.json",
        "oversized_text.json",
        "prompt_injection_field.json",
        "hidden_reasoning.json",
    ],
)
def test_invalid_specialist_fixtures_fail_closed(fixture):
    payload = _load_invalid_specialist(fixture)
    model = SpecialistQuery if payload.get("fixture_type") == "query" else SpecialistResult
    payload.pop("fixture_type", None)

    with pytest.raises(ValidationError):
        model.model_validate(payload)


@pytest.mark.parametrize(
    "fixture",
    [
        "invalid_source_host.json",
        "malformed_provider_attempt.json",
        "token_shaped_content.json",
        "incomparable_trend.json",
        "truncation_corruption.json",
    ],
)
def test_adversarial_specialist_fixtures_fail_closed(fixture):
    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(_load_adversarial_specialist(fixture))


def test_prompt_injection_in_declared_title_fails_for_semantic_reason():
    payload = _load_invalid_specialist("prompt_injection_field.json")

    with pytest.raises(ValidationError, match="instruction-shaped content is forbidden in title"):
        SpecialistResult.model_validate(payload)

    payload["items"][0]["title"] = "Roman gaming token with collector instructions included"
    SpecialistResult.model_validate(payload)


@pytest.mark.parametrize(
    "fixture",
    [
        "declared_ignore_system_prompt.json",
        "declared_tool_instruction.json",
    ],
)
def test_declared_evidence_instruction_patterns_fail_closed(fixture):
    payload = _load_adversarial_specialist(fixture)

    with pytest.raises(ValidationError, match="instruction-shaped content is forbidden in title"):
        SpecialistResult.model_validate(payload)


@pytest.mark.parametrize(
    "content",
    [
        "Roman gaming token: ABCDEFGH",
        "Roman gaming token: ABCDEFGH, likely used as a tessera",
    ],
)
def test_benign_numismatic_token_content_is_accepted(content):
    SpecialistQuery.model_validate({"query": content})

    payload = _load_specialist("market_search_complete.json")
    payload["items"][0]["title"] = content
    SpecialistResult.model_validate(payload)


def test_token_shaped_declared_title_fails_for_semantic_reason():
    payload = _load_adversarial_specialist("token_shaped_content.json")

    with pytest.raises(ValidationError, match="token-shaped content is forbidden"):
        SpecialistResult.model_validate(payload)


def test_market_search_rejects_numisbids_provider_and_source_policy():
    payload = _load_adversarial_specialist("market_search_numisbids.json")

    with pytest.raises(ValidationError, match="provider is not allowed for capability"):
        SpecialistResult.model_validate(payload)


def test_completed_specialist_replay_fixture_is_explicit_and_bounded():
    payload = _load_invalid_specialist("completed_call_replay.json")
    completed_tool = payload["checkpoint"]["completed_tools"][0]

    assert completed_tool["tool_name"] == "price_trends"
    assert completed_tool["tool_call_id"] == "call_specialist_01"
    assert completed_tool["truncated"] is False
    SpecialistResult.model_validate(completed_tool["result"])
