"""Shared Go/Python Coin Copilot contract fixture validation."""

import hashlib
import json
from copy import deepcopy
from pathlib import Path

import httpx
import pytest
from pydantic import ValidationError

from app.models.requests import (
    COPILOT_ALLOWED_TOOLS,
    CopilotCompletedTool,
    CopilotExecuteRequest,
    DeepAnalysisHandoffArguments,
    DeepAnalysisHandoffRequest,
    validate_deep_analysis_handoff_request_envelope,
)
from app.models.responses import (
    CopilotExecutionFrame,
    CopilotToolCompletedPayload,
    DeepAnalysisHandoffResult,
    validate_deep_analysis_handoff_persisted_result_envelope,
    validate_deep_analysis_handoff_public_event_envelope,
)
from app.teams.specialist_contracts import (
    PriceTrendSummary,
    SpecialistQuery,
    SpecialistResult,
    validate_registered_source_url,
)
from app.tools.copilot_collection_tools import (
    ARG_MODELS,
    CALLBACK_TOOLS,
    RESULT_MODELS,
    CopilotCollectionToolClient,
    CopilotToolError,
)

FIXTURES = Path(__file__).parent / "fixtures" / "coin_copilot"
SPECIALIST_FIXTURES = FIXTURES / "specialists"
INVALID_SPECIALIST_FIXTURES = FIXTURES / "specialists_invalid"
HANDOFF_FIXTURES = (
    Path(__file__).parents[3]
    / "specs"
    / "362-coin-copilot-attribution"
    / "contracts"
    / "fixtures"
)


def _load(name: str) -> dict:
    return json.loads((FIXTURES / name).read_text(encoding="utf-8"))


def _load_handoff(name: str) -> dict:
    return json.loads((HANDOFF_FIXTURES / name).read_text(encoding="utf-8"))


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


@pytest.mark.asyncio
async def test_deep_analysis_handoff_callback_uses_fixed_route_and_injected_authority():
    fixture = _load_handoff("deep-analysis-handoff-valid.json")
    observed = {}

    def handler(request: httpx.Request) -> httpx.Response:
        observed["url"] = str(request.url)
        observed["authorization"] = request.headers.get("Authorization")
        observed["body"] = json.loads(request.content)
        return httpx.Response(200, json=fixture["results"]["accepted"])

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
        tool_client = CopilotCollectionToolClient(
            tools_base_url="http://test-api:8080",
            execution_token="canonical-execution-token",
            allowed_tools=["deep_analysis_handoff"],
            max_result_bytes=32768,
            checkpoint_version=7,
            client=client,
        )
        result, _ = await tool_client.execute(
            "deep_analysis_handoff",
            "call_request_01",
            {"operation": "request", "target": {"type": "coin", "id": 42}},
        )

    assert observed["url"] == "http://test-api:8080/api/internal/copilot/tools/deep_analysis_handoff"
    assert observed["authorization"] == "Bearer canonical-execution-token"
    assert observed["body"]["tool_call_id"] == "call_request_01"
    assert observed["body"]["expected_checkpoint_version"] == 7
    assert observed["body"]["handoff_idempotency_key"] == hashlib.sha256(b"call_request_01").hexdigest()
    assert result["outcome"] == "accepted"


def test_deep_analysis_handoff_callback_registry_and_model_authority_are_closed():
    assert "deep_analysis_handoff" in CALLBACK_TOOLS
    assert ARG_MODELS["deep_analysis_handoff"] is DeepAnalysisHandoffArguments
    assert RESULT_MODELS["deep_analysis_handoff"] is DeepAnalysisHandoffResult
    for field in ("owner", "snapshot", "providers", "apply", "url"):
        with pytest.raises(ValidationError):
            DeepAnalysisHandoffArguments.model_validate(
                {"operation": "request", "target": {"type": "coin", "id": 42}, field: "forged"}
            )


def test_deep_analysis_handoff_bounded_fallback_is_valid_for_live_frame_and_checkpoint():
    fallback = _bounded_fallback()

    frame_payload = CopilotToolCompletedPayload.model_validate(
        {
            "tool_call_id": "call_status_01",
            "tool_name": "deep_analysis_handoff",
            "step_id": "step_status_01",
            "status": "succeeded",
            "duration_ms": 12,
            "result_summary": "Deep Analysis status loaded.",
            "result": fallback,
        }
    )
    checkpoint_tool = CopilotCompletedTool.model_validate(
        {
            "tool_call_id": "call_status_01",
            "tool_name": "deep_analysis_handoff",
            "result": fallback,
            "truncated": True,
        }
    )

    assert frame_payload.result == fallback
    assert checkpoint_tool.result == fallback


@pytest.mark.asyncio
async def test_deep_analysis_handoff_callback_rejects_oversized_result():
    transport = httpx.MockTransport(
        lambda _request: httpx.Response(200, json={"padding": "x" * 32769})
    )
    async with httpx.AsyncClient(transport=transport) as client:
        tool_client = CopilotCollectionToolClient(
            tools_base_url="http://test-api:8080",
            execution_token="token",
            allowed_tools=["deep_analysis_handoff"],
            max_result_bytes=32768,
            checkpoint_version=1,
            client=client,
        )
        with pytest.raises(CopilotToolError):
            await tool_client.execute(
                "deep_analysis_handoff",
                "call_status_01",
                {"operation": "status", "job_id": 314},
            )


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


def test_request_accepts_bounded_collector_context_and_rejects_action_fields():
    payload = _load("valid_execute_request.json")
    payload["collector_context"] = {
        "budget_min": 100,
        "budget_max": 500,
        "currency": "USD",
        "preferred_periods": ["Roman Imperial"],
        "preferred_categories": ["Roman"],
        "excluded_categories": [],
        "preferred_dealers": ["VCoins"],
        "collecting_goals": ["Build a representative Probus mint set"],
        "captured_at": "2026-09-19T13:00:00Z",
    }

    request = CopilotExecuteRequest.model_validate(payload)

    assert request.collector_context.currency == "USD"
    assert request.collector_context.collecting_goals == [
        "Build a representative Probus mint set"
    ]

    payload["collector_context"]["owner_id"] = 7
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
            "result": specialist_result,
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
                "result": specialist_result,
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
                "result": specialist_result,
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
                "result": fallback,
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
    assert "truncated" in specialist_result["truncation"]
    assert all(item["sourceUrl"].startswith("https://") for item in specialist_result["items"])
    if specialist_result["trend"] is not None:
        assert "sampleSize" in specialist_result["trend"]
        assert "sample_size" not in specialist_result["trend"]


def test_deep_analysis_handoff_requests_are_strict_and_mutually_exclusive():
    fixture = _load_handoff("deep-analysis-handoff-valid.json")
    for operation in ("request", "status", "rerun"):
        request = DeepAnalysisHandoffRequest.model_validate(fixture["requests"][operation])
        assert request.operation == operation

    invalid = _load_handoff("deep-analysis-handoff-invalid.json")
    for case in invalid["request_cases"]:
        if case["name"] == "duplicate_call":
            continue
        with pytest.raises(ValidationError):
            DeepAnalysisHandoffRequest.model_validate(case["payload"])


def test_deep_analysis_handoff_result_requires_outcome_and_forbids_top_level_status():
    fixture = _load_handoff("deep-analysis-handoff-valid.json")
    result = DeepAnalysisHandoffResult.model_validate(fixture["results"]["status_complete"])
    assert result.outcome == "status"
    assert result.job is not None
    assert result.job.status == "completed"

    invalid = _load_handoff("deep-analysis-handoff-invalid.json")
    status_case = next(case for case in invalid["result_cases"] if case["name"] == "result_level_status_discriminant")
    with pytest.raises(ValidationError):
        DeepAnalysisHandoffResult.model_validate(status_case["payload"])


def _canonical_padding_envelope(size: int) -> bytes:
    empty = b'{"padding":""}'
    return b'{"padding":"' + (b"x" * (size - len(empty))) + b'"}'


def test_deep_analysis_handoff_has_independent_request_public_and_persisted_byte_limits():
    assert validate_deep_analysis_handoff_request_envelope(_canonical_padding_envelope(65536)) is not None
    with pytest.raises(ValueError):
        validate_deep_analysis_handoff_request_envelope(_canonical_padding_envelope(65537))

    assert validate_deep_analysis_handoff_public_event_envelope(_canonical_padding_envelope(65536)) is not None
    with pytest.raises(ValueError):
        validate_deep_analysis_handoff_public_event_envelope(_canonical_padding_envelope(65537))

    assert validate_deep_analysis_handoff_persisted_result_envelope(_canonical_padding_envelope(32768)) is not None
    with pytest.raises(ValueError):
        validate_deep_analysis_handoff_persisted_result_envelope(_canonical_padding_envelope(32769))


def test_deep_analysis_handoff_truncation_discloses_digest_and_omission_counts():
    fixture = _load_handoff("deep-analysis-handoff-valid.json")
    result = DeepAnalysisHandoffResult.model_validate(fixture["results"]["truncated"])
    assert result.truncation is not None
    assert result.truncation.truncated is True
    assert result.truncation.persisted_bytes == 32768
    assert len(result.truncation.digest) == 64
    assert result.truncation.omitted_evidence == 18
    assert result.limitations

    checkpoint_tool = CopilotCompletedTool.model_validate(
        {
            "tool_call_id": "call_status_01",
            "tool_name": "deep_analysis_handoff",
            "result": fixture["results"]["truncated"],
            "truncated": True,
        }
    )
    assert isinstance(checkpoint_tool.result, DeepAnalysisHandoffResult)

    tampered = deepcopy(fixture["results"]["truncated"])
    tampered["truncation"]["digest"] = "not-a-digest"
    with pytest.raises(ValidationError):
        CopilotCompletedTool.model_validate(
            {
                "tool_call_id": "call_status_01",
                "tool_name": "deep_analysis_handoff",
                "result": tampered,
            }
        )


def test_deep_analysis_handoff_execution_token_is_forwarded_only_as_execution_context():
    payload = _load("valid_execute_request.json")
    payload["allowed_tools"] = [*payload["allowed_tools"], "deep_analysis_handoff"]
    payload["execution_token"] = "canonical-execution-token"
    request = CopilotExecuteRequest.model_validate(payload)
    assert "deep_analysis_handoff" in COPILOT_ALLOWED_TOOLS
    assert request.execution_token == "canonical-execution-token"

    handoff = _load_handoff("deep-analysis-handoff-valid.json")["requests"]["request"]
    with pytest.raises(ValidationError):
        DeepAnalysisHandoffRequest.model_validate({**handoff, "authorization": "Bearer forbidden"})


@pytest.mark.parametrize(
    "fixture",
    [
        "unknown_query_field.json",
        "invalid_enum.json",
        "unsafe_url_http.json",
        "unsafe_url_credentials.json",
        "unsafe_url_private.json",
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
        "malformed_provider_attempt.json",
        "token_shaped_content.json",
    ],
)
def test_adversarial_specialist_fixtures_fail_closed(fixture):
    with pytest.raises(ValidationError):
        SpecialistResult.model_validate(_load_adversarial_specialist(fixture))


def test_configured_specialist_source_host_fails_closed():
    payload = _load_adversarial_specialist("invalid_source_host.json")
    source_url = payload["items"][0]["source_url"]

    with pytest.raises(ValueError, match="not configured"):
        validate_registered_source_url(
            "configured_dealer_search",
            source_url,
            frozenset({"cngcoins.com"}),
        )


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


def test_completed_specialist_replay_fixture_is_explicit_and_bounded():
    payload = _load_invalid_specialist("completed_call_replay.json")
    completed_tool = payload["checkpoint"]["completed_tools"][0]

    assert completed_tool["tool_name"] == "price_trends"
    assert completed_tool["tool_call_id"] == "call_specialist_01"
    assert completed_tool["truncated"] is False
    SpecialistResult.model_validate(completed_tool["result"])
