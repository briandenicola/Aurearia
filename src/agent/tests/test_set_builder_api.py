"""Phase 2 QA for the Python set-builder workflow.

Contract anchor: specs/011-dynamic-set-builder-correction-plan.md (Phase 2).

Model-level tests exercise the `SetBuilderRequest` / `SetBuilderResponse`
schemas (`app/models/requests.py`, `app/models/responses.py`). Route-level
tests exercise `POST /api/set-builder/run` (`app/routes.py`) by monkeypatching
`app.routes.run_set_builder_workflow` — the async entrypoint defined in
`app/teams/set_builder.py` — so no test requires a live AI provider.
"""

import pytest
from fastapi.testclient import TestClient
from pydantic import ValidationError

from app.main import app
from app.models.requests import MAX_SET_BUILDER_MAX_SLOTS, MAX_SET_BUILDER_MAX_TURNS, SetBuilderRequest
from app.models.responses import SetBuilderProposal, SetBuilderResponse, SetBuilderSlot

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}

SET_BUILDER_ROUTE = "/api/set-builder/run"

_VALID_LLM = {"provider": "anthropic", "api_key": "k", "model": "claude-opus-4-8"}
_VALID_USER = {"user_id": 1}


def _proposal_payload(prompt: str = "Build a set of Julio-Claudian denarii") -> dict:
    return {
        "llm": _VALID_LLM,
        "user": _VALID_USER,
        "prompt": prompt,
    }


# ---------------------------------------------------------------------------
# Request schema validation (coverage expectation 2: invalid prompt -> 422).
# ---------------------------------------------------------------------------


def test_set_builder_request_accepts_minimal_valid_payload():
    request = SetBuilderRequest(llm=_VALID_LLM, user=_VALID_USER, prompt="Build a set of Julio-Claudian denarii")
    assert request.prompt
    assert request.max_turns >= 1
    assert request.enable_external_lookup is True


def test_set_builder_request_rejects_empty_prompt():
    with pytest.raises(ValidationError):
        SetBuilderRequest(llm=_VALID_LLM, user=_VALID_USER, prompt="")


def test_set_builder_request_rejects_missing_prompt():
    with pytest.raises(ValidationError):
        SetBuilderRequest(llm=_VALID_LLM, user=_VALID_USER)


def test_set_builder_request_rejects_oversized_prompt():
    with pytest.raises(ValidationError):
        SetBuilderRequest(llm=_VALID_LLM, user=_VALID_USER, prompt="x" * 501)


def test_set_builder_request_rejects_oversized_feedback():
    with pytest.raises(ValidationError):
        SetBuilderRequest(
            llm=_VALID_LLM,
            user=_VALID_USER,
            prompt="Build a set of Julio-Claudian denarii",
            feedback="x" * 1001,
        )


def test_set_builder_request_rejects_max_turns_out_of_bounds():
    with pytest.raises(ValidationError):
        SetBuilderRequest(
            llm=_VALID_LLM,
            user=_VALID_USER,
            prompt="Build a set of Julio-Claudian denarii",
            max_turns=MAX_SET_BUILDER_MAX_TURNS + 1,
        )


def test_set_builder_request_rejects_max_slots_out_of_bounds():
    with pytest.raises(ValidationError):
        SetBuilderRequest(
            llm=_VALID_LLM,
            user=_VALID_USER,
            prompt="Build a set of Julio-Claudian denarii",
            max_slots=MAX_SET_BUILDER_MAX_SLOTS + 1,
        )


def test_set_builder_request_rejects_unknown_fields():
    """Drift detection: Go proxy contract changes must be caught, not silently dropped."""
    with pytest.raises(ValidationError):
        SetBuilderRequest(
            llm=_VALID_LLM,
            user=_VALID_USER,
            prompt="Build a set of Julio-Claudian denarii",
            unexpected_field="nope",
        )


# ---------------------------------------------------------------------------
# Response schema shape (coverage expectation 5: slot schema includes
# criteria, group, sortOrder, verificationStatus).
# ---------------------------------------------------------------------------


def test_set_builder_slot_schema_includes_required_fields():
    slot = SetBuilderSlot(
        label="Augustus Denarius",
        criteria={"ruler": "Augustus", "denomination": "Denarius"},
        group="Julio-Claudian",
        sort_order=1,
        verification_status="verified",
    )
    dumped = slot.model_dump()
    assert dumped["criteria"] == {"ruler": "Augustus", "denomination": "Denarius"}
    assert dumped["group"] == "Julio-Claudian"
    assert dumped["sort_order"] == 1
    assert dumped["verification_status"] == "verified"


def test_set_builder_slot_defaults_to_unverified():
    slot = SetBuilderSlot(label="Unresearched Slot")
    assert slot.verification_status == "unverified"
    assert slot.criteria == {}


def test_set_builder_slot_rejects_invalid_verification_status():
    with pytest.raises(ValidationError):
        SetBuilderSlot(label="Bad Slot", verification_status="maybe")


def test_set_builder_response_completed_status_carries_proposal():
    response = SetBuilderResponse(
        status="completed",
        proposal=SetBuilderProposal(
            name="Julio-Claudian Denarii",
            slots=[
                SetBuilderSlot(
                    label="Augustus Denarius",
                    criteria={"ruler": "Augustus"},
                    group="Julio-Claudian",
                    sort_order=1,
                    verification_status="unverified",
                )
            ],
        ),
    )
    assert response.status == "completed"
    assert response.proposal is not None
    assert response.proposal.slots[0].verification_status == "unverified"


@pytest.mark.parametrize("status", ["clarification_needed", "failed", "limit_reached", "rejected"])
def test_set_builder_response_non_completed_statuses_omit_proposal(status):
    """Coverage expectation 4: ambiguous/unbounded prompts must not fabricate slots."""
    response = SetBuilderResponse(status=status, clarification_question="Which era?", failure_reason="too broad")
    assert response.proposal is None
    assert response.status == status


def test_set_builder_response_rejects_unknown_fields():
    with pytest.raises(ValidationError):
        SetBuilderResponse(status="completed", unexpected_field="nope")


# ---------------------------------------------------------------------------
# Route-level tests for POST /api/set-builder/run.
# Coverage expectations 1, 3, 4, 6.
# ---------------------------------------------------------------------------


def test_set_builder_route_requires_internal_token():
    resp = client.post(SET_BUILDER_ROUTE, json=_proposal_payload())
    assert resp.status_code == 401


def test_set_builder_route_rejects_invalid_body():
    resp = client.post(SET_BUILDER_ROUTE, json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_set_builder_route_rejects_empty_prompt():
    resp = client.post(SET_BUILDER_ROUTE, json=_proposal_payload(prompt=""), headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_set_builder_route_returns_structured_proposal(monkeypatch):
    """Coverage expectation 3: monkeypatched workflow, no live AI provider call."""
    import app.routes as routes

    captured = {}

    async def fake_run_set_builder_workflow(request):
        captured["request"] = request
        return SetBuilderResponse(
            status="completed",
            proposal=SetBuilderProposal(
                name="Julio-Claudian Denarii",
                slots=[
                    SetBuilderSlot(
                        label="Augustus Denarius",
                        criteria={"ruler": "Augustus"},
                        group="Julio-Claudian",
                        sort_order=1,
                        verification_status="unverified",
                    )
                ],
            ),
            transcript_summary="intent -> roster -> match -> validate",
            turns_used=3,
        )

    monkeypatch.setattr(routes, "run_set_builder_workflow", fake_run_set_builder_workflow)

    resp = client.post(SET_BUILDER_ROUTE, json=_proposal_payload(), headers=AUTH_HEADERS)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "completed"
    slot = data["proposal"]["slots"][0]
    assert slot["criteria"] == {"ruler": "Augustus"}
    assert slot["group"] == "Julio-Claudian"
    assert slot["sort_order"] == 1
    assert slot["verification_status"] == "unverified"
    assert captured["request"].prompt == "Build a set of Julio-Claudian denarii"


def test_set_builder_route_ambiguous_prompt_returns_clarification(monkeypatch):
    """Coverage expectation 4: ambiguous/unbounded prompt returns clarification, not a fabricated roster."""
    import app.routes as routes

    async def fake_run_set_builder_workflow(_request):
        return SetBuilderResponse(
            status="clarification_needed",
            clarification_question="Do you mean Roman Republic or Roman Imperial denarii?",
            turns_used=1,
        )

    monkeypatch.setattr(routes, "run_set_builder_workflow", fake_run_set_builder_workflow)

    resp = client.post(SET_BUILDER_ROUTE, json=_proposal_payload(prompt="Build me a coin set"), headers=AUTH_HEADERS)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "clarification_needed"
    assert data.get("proposal") is None
    assert data["clarification_question"]


def test_set_builder_route_unbounded_prompt_returns_limit_reached_not_fabricated(monkeypatch):
    """Unbounded prompts that exhaust the turn/token budget must fail structured, never fabricate slots."""
    import app.routes as routes

    async def fake_run_set_builder_workflow(_request):
        return SetBuilderResponse(
            status="limit_reached",
            failure_reason="Roster research exceeded max turns without a bounded scope.",
            turns_used=8,
        )

    monkeypatch.setattr(routes, "run_set_builder_workflow", fake_run_set_builder_workflow)

    resp = client.post(
        SET_BUILDER_ROUTE,
        json=_proposal_payload(prompt="Build a set of every coin ever minted"),
        headers=AUTH_HEADERS,
    )

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "limit_reached"
    assert data.get("proposal") is None


def test_set_builder_route_workflow_failure_returns_structured_failure(monkeypatch):
    """Coverage expectation 6: even on unexpected workflow exceptions, no live AI call is required
    and the route must not raise — it should surface a structured `failed` status."""
    import app.routes as routes

    async def fake_run_set_builder_workflow(_request):
        return SetBuilderResponse(
            status="failed",
            failure_reason="The set-builder workflow could not complete. Please try again.",
            turns_used=0,
        )

    monkeypatch.setattr(routes, "run_set_builder_workflow", fake_run_set_builder_workflow)

    resp = client.post(SET_BUILDER_ROUTE, json=_proposal_payload(), headers=AUTH_HEADERS)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "failed"
    assert data.get("proposal") is None
    assert data["failure_reason"]
