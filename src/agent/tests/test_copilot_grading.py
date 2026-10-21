"""Rubric judge shared by the Copilot eval and nightly grading of real runs."""

import json
from types import SimpleNamespace

import pytest
from fastapi.testclient import TestClient

from app.main import app
from app.teams import copilot_grading

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}

PASSING_VERDICT = {
    "answers_directly": True,
    "grounded": True,
    "accurate": True,
    "cites_sources": "yes",
    "no_offer_menu": True,
    "no_false_inability": True,
    "case_points": [],
    "explanation": "All criteria pass.",
}


def _response(verdict: dict, stop_reason: str = "end_turn"):
    return SimpleNamespace(
        stop_reason=stop_reason,
        model="claude-sonnet-5-5",
        content=[SimpleNamespace(type="text", text=json.dumps(verdict))],
        usage=SimpleNamespace(model_dump=lambda **_: {"input_tokens": 900, "output_tokens": 120}),
    )


class _FakeClient:
    def __init__(self, response):
        self.calls = []
        self.response = response
        self.beta = SimpleNamespace(messages=SimpleNamespace(create=self._create))

    async def _create(self, **kwargs):
        self.calls.append(kwargs)
        return self.response


def test_score_counts_sources_only_when_applicable():
    assert copilot_grading.score(PASSING_VERDICT) == 1.0
    no_sources = {**PASSING_VERDICT, "cites_sources": "not_applicable", "no_offer_menu": False}
    assert copilot_grading.score(no_sources) == pytest.approx(4 / 5)
    uncited = {**PASSING_VERDICT, "cites_sources": "no", "case_points": [True, False]}
    assert copilot_grading.score(uncited) == pytest.approx(6 / 8)


def test_render_conversation_marks_untrusted_transcript_and_truncated_tools():
    rendered = copilot_grading.render_conversation(
        [{"role": "user", "content": "What are my three most valuable coins?"}],
        [
            {"tool_name": "top_coins_by_value", "result": {"coins": [{"name": "Athens owl"}]}},
            {"tool_name": "market_search", "result": {"truncated": True}, "truncated": True},
        ],
        "Your Athens owl leads.",
    )

    assert rendered.startswith("<transcript>")
    assert "[user] What are my three most valuable coins?" in rendered
    assert "[tool result top_coins_by_value]" in rendered and "Athens owl" in rendered
    assert "exceeded the saved-result limit" in rendered
    assert rendered.index("[assistant] Your Athens owl leads.") > rendered.index("[tool result market_search]")


@pytest.mark.asyncio
async def test_judge_uses_structured_output_and_fallbacks():
    fake = _FakeClient(_response(PASSING_VERDICT))

    graded = await copilot_grading.judge(fake, "claude-sonnet-5-5", "system", "transcript")

    assert graded["quality"] == 1.0
    assert graded["judge_model"] == "claude-sonnet-5-5"
    call = fake.calls[0]
    assert call["output_config"]["format"]["schema"] == copilot_grading.JUDGE_SCHEMA
    assert call["fallbacks"] == "default"


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("response", "expected_points"),
    [
        (_response(PASSING_VERDICT, stop_reason="refusal"), 0),
        (_response(PASSING_VERDICT), 2),
    ],
)
async def test_judge_rejects_refusals_and_wrong_point_counts(response, expected_points):
    with pytest.raises(copilot_grading.GradingError):
        await copilot_grading.judge(_FakeClient(response), "claude-sonnet-5-5", "system", "transcript", expected_points)


def _grade_payload(**overrides):
    payload = {
        "llm": {"provider": "anthropic", "api_key": "key", "model": "claude-sonnet-5-5"},
        "messages": [{"role": "user", "content": "Is my Hadrian sestertius a common type?"}],
        "tools": [{"tool_name": "search_my_collection", "result": {"coins": []}}],
        "answer": "Yes, the Fortuna type is common.",
    }
    payload.update(overrides)
    return payload


def test_grade_endpoint_returns_the_verdict(monkeypatch):
    seen = {}

    async def fake_judge(_client, judge_model, system, transcript, expected_points=0):
        seen.update(model=judge_model, system=system, transcript=transcript, points=expected_points)
        return {
            "quality": 0.8,
            "verdict": {**PASSING_VERDICT, "no_offer_menu": False, "cites_sources": "not_applicable"},
            "judge_model": "claude-sonnet-5-5",
            "judge_usage": {"input_tokens": 100, "cache_read_input_tokens": 50, "output_tokens": 30},
        }

    monkeypatch.setattr(copilot_grading, "judge", fake_judge)

    response = client.post("/api/copilot/grade", json=_grade_payload(), headers=AUTH_HEADERS)

    assert response.status_code == 200, response.text
    body = response.json()
    assert body["quality"] == 0.8
    assert body["no_offer_menu"] is False
    assert body["cites_sources"] == "not_applicable"
    assert body["input_tokens"] == 150
    assert seen["system"] == copilot_grading.PRODUCTION_JUDGE_SYSTEM
    assert seen["points"] == 0
    assert "Hadrian sestertius" in seen["transcript"]


def test_grade_endpoint_requires_an_anthropic_judge():
    payload = _grade_payload(llm={"provider": "ollama", "model": "llama3.1", "ollama_url": "http://localhost:11434"})

    response = client.post("/api/copilot/grade", json=payload, headers=AUTH_HEADERS)

    assert response.status_code == 422


def test_grade_endpoint_maps_unusable_verdicts_to_502(monkeypatch):
    async def failing_judge(*_args, **_kwargs):
        raise copilot_grading.GradingError("judge refused")

    monkeypatch.setattr(copilot_grading, "judge", failing_judge)

    response = client.post("/api/copilot/grade", json=_grade_payload(), headers=AUTH_HEADERS)

    assert response.status_code == 502


def test_grade_endpoint_requires_the_internal_token():
    response = client.post("/api/copilot/grade", json=_grade_payload())

    assert response.status_code in (401, 403)
