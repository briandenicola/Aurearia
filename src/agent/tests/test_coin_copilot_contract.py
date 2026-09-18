"""Shared Go/Python Coin Copilot contract fixture validation."""

import json
from pathlib import Path

import pytest
from pydantic import ValidationError

from app.models.requests import CopilotExecuteRequest
from app.models.responses import CopilotExecutionFrame

FIXTURES = Path(__file__).parent / "fixtures" / "coin_copilot"


def _load(name: str) -> dict:
    return json.loads((FIXTURES / name).read_text(encoding="utf-8"))


def test_valid_execute_request_fixture_is_strict_and_current():
    request = CopilotExecuteRequest.model_validate(_load("valid_execute_request.json"))
    assert request.schema_version == 1
    assert request.limits.max_iterations == 8
    assert request.limits.max_tool_calls == 12
    assert request.limits.hard_timeout_seconds == 120
    assert request.limits.max_persisted_tool_result_bytes == 32768
    assert request.app_context is not None
    assert request.app_context.active_coin_id is None


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
