"""Tests for the Dynamic Set Builder workflow route and team.

Covers spec 011 Phase 2 acceptance: route validation, a successful
monkeypatched workflow response, ambiguous-prompt clarification handling,
and that output is structured proposal data only (no set creation).
"""

import json

import pytest
from fastapi.testclient import TestClient

from app.main import app
from app.models.requests import SetBuilderRequest
from app.models.responses import SetBuilderResponse
from app.teams.set_builder import run_set_builder_workflow

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


def valid_payload(**overrides) -> dict:
    payload = {
        "llm": {"provider": "anthropic", "api_key": "k", "model": "m"},
        "user": {"user_id": 1},
        "prompt": "All American wheat pennies by date and mint",
    }
    payload.update(overrides)
    return payload


class _FakeResponse:
    def __init__(self, content):
        self.content = content


class _FakeModel:
    """Returns queued JSON payloads in order, one per `ainvoke` call."""

    def __init__(self, payloads: list[dict | list]):
        self._payloads = list(payloads)

    async def ainvoke(self, _messages, **_kwargs):
        payload = self._payloads.pop(0)
        if isinstance(payload, str) or (
            isinstance(payload, list)
            and payload
            and isinstance(payload[0], dict)
            and "type" in payload[0]
        ):
            return _FakeResponse(payload)
        return _FakeResponse(json.dumps(payload))


# ---------------------------------------------------------------------------
# Route-level validation and auth
# ---------------------------------------------------------------------------


def test_set_builder_requires_internal_token():
    resp = client.post("/api/set-builder/run", json=valid_payload())
    assert resp.status_code == 401


def test_set_builder_rejects_invalid_body():
    resp = client.post("/api/set-builder/run", json={}, headers=AUTH_HEADERS)
    assert resp.status_code == 422


def test_set_builder_rejects_empty_prompt():
    resp = client.post(
        "/api/set-builder/run", json=valid_payload(prompt=""), headers=AUTH_HEADERS
    )
    assert resp.status_code == 422


def test_set_builder_rejects_prompt_over_length_cap():
    resp = client.post(
        "/api/set-builder/run",
        json=valid_payload(prompt="x" * 501),
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 422


def test_set_builder_rejects_unknown_fields():
    resp = client.post(
        "/api/set-builder/run",
        json=valid_payload(unexpected_field="nope"),
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 422


def test_set_builder_rejects_out_of_range_max_slots():
    resp = client.post(
        "/api/set-builder/run",
        json=valid_payload(max_slots=10_000),
        headers=AUTH_HEADERS,
    )
    assert resp.status_code == 422


# ---------------------------------------------------------------------------
# Successful monkeypatched workflow response (route level)
# ---------------------------------------------------------------------------


def test_set_builder_route_returns_structured_proposal(monkeypatch):
    async def fake_run(_request):
        return SetBuilderResponse(
            status="completed",
            proposal={
                "name": "Lincoln Wheat Cents",
                "scope_summary": "One coin per date and mint from 1909 to 1958.",
                "selected_scope": "date+mint",
                "group_by": "decade",
                "scope_options": [],
                "slots": [
                    {
                        "label": "1943 Lincoln Wheat Cent",
                        "criteria": {"year": "1943"},
                        "group": "1940s",
                        "sort_order": 0,
                        "verification_status": "verified",
                        "source_note": "Well-known date.",
                        "validation_notes": "",
                    }
                ],
                "prematch_summary": {"estimated_filled": 0, "estimated_total": 1, "notes": ""},
            },
            transcript_summary="Intent analysis: Lincoln Wheat Cents | Roster research produced 1 candidate slots.",
            turns_used=4,
        )

    monkeypatch.setattr("app.routes.run_set_builder_workflow", fake_run)

    resp = client.post("/api/set-builder/run", json=valid_payload(), headers=AUTH_HEADERS)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "completed"
    assert data["proposal"]["name"] == "Lincoln Wheat Cents"
    assert data["proposal"]["slots"][0]["verification_status"] == "verified"
    assert data["turns_used"] == 4
    # Output is data only — no set/coin identifiers of any kind are present.
    assert "id" not in data["proposal"]
    assert set(data.keys()) == {
        "status",
        "proposal",
        "clarification_question",
        "failure_reason",
        "transcript_summary",
        "turns_used",
    }


# ---------------------------------------------------------------------------
# Workflow-level tests (monkeypatch the LLM provider, not the route)
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_workflow_full_pipeline_produces_completed_proposal(monkeypatch):
    intent = {
        "subject": "Lincoln Wheat Cents",
        "is_numismatic": True,
        "clarification_needed": False,
        "clarification_question": "",
        "scope_summary": "One coin per date and mint from 1909 to 1958.",
        "selected_scope": "date+mint",
        "group_by": "decade",
        "scope_options": [{"label": "date+mint", "description": "By date and mint", "recommended": True}],
    }
    roster = [
        {
            "label": "1909-S VDB", "criteria": {"year": "1909", "mint": "S"},
            "group": "1900s", "sort_order": 0, "source_note": "Key date.",
        },
        {
            "label": "1943 Steel Cent", "criteria": {"year": "1943"},
            "group": "1940s", "sort_order": 1, "source_note": "Wartime composition.",
        },
    ]
    match = {"estimated_filled": 1, "estimated_total": 2, "notes": "One likely match found."}
    validated = [
        {**roster[0], "verification_status": "verified", "validation_notes": ""},
        {**roster[1], "verification_status": "unverified", "validation_notes": "Could not confirm variety."},
    ]

    fake_model = _FakeModel([intent, roster, match, validated])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: fake_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: fake_model)

    request = SetBuilderRequest(**valid_payload())
    response = await run_set_builder_workflow(request)

    assert response.status == "completed"
    assert response.proposal is not None
    assert response.proposal.name == "Lincoln Wheat Cents"
    assert len(response.proposal.slots) == 2
    assert response.proposal.slots[0].verification_status == "verified"
    assert response.proposal.slots[1].verification_status == "unverified"
    assert response.proposal.prematch_summary.estimated_filled == 1
    assert response.turns_used == 4
    assert "Intent analysis" in response.transcript_summary


@pytest.mark.asyncio
async def test_workflow_parses_anthropic_content_blocks(monkeypatch):
    intent = {
        "subject": "Lincoln Wheat Cents",
        "is_numismatic": True,
        "clarification_needed": False,
        "clarification_question": "",
        "scope_summary": "One coin per date and mint from 1909 to 1930.",
        "selected_scope": "date+mint",
        "group_by": "decade",
        "scope_options": [],
    }
    roster = [
        {
            "label": "1909-S VDB",
            "criteria": {"year": "1909", "mint": "S"},
            "group": "1900s",
            "sort_order": 0,
            "source_note": "Key date.",
        },
    ]
    match = {"estimated_filled": 0, "estimated_total": 1, "notes": ""}
    validated = [{**roster[0], "verification_status": "verified", "validation_notes": ""}]
    roster_block = [{"type": "text", "text": "```json\n" + json.dumps(roster) + "\n```"}]

    fake_model = _FakeModel([intent, roster_block, match, validated])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: fake_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: fake_model)

    request = SetBuilderRequest(**valid_payload())
    response = await run_set_builder_workflow(request)

    assert response.status == "completed"
    assert response.proposal is not None
    assert response.proposal.slots[0].label == "1909-S VDB"


@pytest.mark.asyncio
async def test_workflow_ambiguous_prompt_returns_clarification(monkeypatch):
    intent = {
        "subject": "cool coins",
        "is_numismatic": True,
        "clarification_needed": True,
        "clarification_question": "What era, country, or theme of coins would you like this set to cover?",
        "scope_summary": "",
        "selected_scope": "",
        "group_by": "",
        "scope_options": [],
    }
    fake_model = _FakeModel([intent])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: fake_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: fake_model)

    request = SetBuilderRequest(**valid_payload(prompt="cool coins"))
    response = await run_set_builder_workflow(request)

    assert response.status == "clarification_needed"
    assert response.proposal is None
    assert response.clarification_question
    assert response.failure_reason == ""


@pytest.mark.asyncio
async def test_workflow_non_numismatic_prompt_is_rejected(monkeypatch):
    intent = {
        "subject": "cooking recipes",
        "is_numismatic": False,
        "clarification_needed": False,
        "clarification_question": "",
        "scope_summary": "",
        "selected_scope": "",
        "group_by": "",
        "scope_options": [],
    }
    fake_model = _FakeModel([intent])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: fake_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: fake_model)

    request = SetBuilderRequest(**valid_payload(prompt="Give me a recipe for lasagna"))
    response = await run_set_builder_workflow(request)

    assert response.status == "rejected"
    assert response.proposal is None
    assert response.failure_reason != ""


@pytest.mark.asyncio
async def test_workflow_turn_limit_reached_before_roster(monkeypatch):
    intent = {
        "subject": "All Roman Imperial coins",
        "is_numismatic": True,
        "clarification_needed": False,
        "clarification_question": "",
        "scope_summary": "Every Roman Imperial coin type ever minted.",
        "selected_scope": "all",
        "group_by": "none",
        "scope_options": [],
    }
    fake_model = _FakeModel([intent])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: fake_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: fake_model)

    request = SetBuilderRequest(**valid_payload(prompt="All Roman Imperial coins", max_turns=1))
    response = await run_set_builder_workflow(request)

    assert response.status == "limit_reached"
    assert response.proposal is None
    assert response.turns_used == 1


@pytest.mark.asyncio
async def test_workflow_caps_slots_at_max_slots(monkeypatch):
    intent = {
        "subject": "US State Quarters",
        "is_numismatic": True,
        "clarification_needed": False,
        "clarification_question": "",
        "scope_summary": "One quarter per state.",
        "selected_scope": "by-state",
        "group_by": "state",
        "scope_options": [],
    }
    roster = [
        {"label": f"State {i}", "criteria": {"state": str(i)}, "group": "", "sort_order": i, "source_note": ""}
        for i in range(5)
    ]
    match = {"estimated_filled": 0, "estimated_total": 5, "notes": ""}
    validated = [{**item, "verification_status": "verified", "validation_notes": ""} for item in roster]

    fake_model = _FakeModel([intent, roster, match, validated])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: fake_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: fake_model)

    request = SetBuilderRequest(**valid_payload(prompt="US State Quarters", max_slots=2))
    response = await run_set_builder_workflow(request)

    assert response.status == "completed"
    assert len(response.proposal.slots) == 2
    assert "truncated" in response.proposal.prematch_summary.notes.lower()


@pytest.mark.asyncio
async def test_workflow_roster_retry_recovers_after_invalid_research_output(monkeypatch):
    intent = {
        "subject": "Byzantine Emperors 476-565",
        "is_numismatic": True,
        "clarification_needed": False,
        "clarification_question": "",
        "scope_summary": "Coins from Byzantine emperors between 476 and 565 AD.",
        "selected_scope": "ruler+date-range",
        "group_by": "ruler",
        "scope_options": [],
    }
    roster = [
        {
            "label": "Anastasius I",
            "criteria": {"ruler": "Anastasius I", "era": "medieval"},
            "group": "Byzantine Emperors",
            "sort_order": 0,
            "source_note": "Common attribution target.",
        }
    ]
    match = {"estimated_filled": 0, "estimated_total": 1, "notes": ""}
    validated = [{**roster[0], "verification_status": "verified", "validation_notes": ""}]

    plain_model = _FakeModel([intent, roster, match, validated])
    research_model = _FakeModel(["this is not valid json"])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: plain_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: research_model)

    request = SetBuilderRequest(**valid_payload(prompt="Create a set of all Byzantine Emperors from 476AD to 565AD"))
    response = await run_set_builder_workflow(request)

    assert response.status == "completed"
    assert response.proposal is not None
    assert len(response.proposal.slots) == 1
    assert response.proposal.slots[0].label == "Anastasius I"


@pytest.mark.asyncio
async def test_workflow_roster_retry_returns_clarification_when_both_attempts_fail(monkeypatch):
    intent = {
        "subject": "English Kings 1800-2026",
        "is_numismatic": True,
        "clarification_needed": False,
        "clarification_question": "",
        "scope_summary": "Coins from English/British kings between 1800 and 2026.",
        "selected_scope": "ruler+date-range",
        "group_by": "ruler",
        "scope_options": [],
    }

    plain_model = _FakeModel([intent, "still not valid json"])
    research_model = _FakeModel(["not valid json either"])
    monkeypatch.setattr("app.teams.set_builder.get_chat_model", lambda _cfg: plain_model)
    monkeypatch.setattr("app.teams.set_builder.get_search_model", lambda _cfg: research_model)

    request = SetBuilderRequest(**valid_payload(prompt="Create a set of English Kings from 1800 to 2026"))
    response = await run_set_builder_workflow(request)

    assert response.status == "clarification_needed"
    assert response.proposal is None
    assert "narrow" in response.clarification_question.lower()
