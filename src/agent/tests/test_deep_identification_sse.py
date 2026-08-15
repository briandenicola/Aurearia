"""SSE envelope tests (T076) — contracts/agent-internal-contract.md §3.

Verifies the internal streaming envelope for `run_deep_identification_stream`:
terminal-frame invariants, sanitized output (no token-shaped strings leak),
and the expiry-with-partial-evidence fallback path (T069).
"""

import asyncio
import json

import pytest

import app.teams.deep_identification.graph as graph_module
from app.models.requests import (
    DeepIdentifyBounds,
    DeepIdentifyImage,
    DeepIdentifyRequest,
    DeepProviderCatalogEntry,
)
from app.models.responses import DeepSynthesis, ProviderEvidence


class FakeModel:
    async def ainvoke(self, messages, **kwargs):
        return type("Resp", (), {"content": "A generic ancient bronze coin with worn legends."})()


def _tiny_data_uri() -> str:
    return (  # noqa: E501 — a real minimal PNG data URI cannot be shortened
        "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
    )


def _request(total_timeout_s: int = 30) -> DeepIdentifyRequest:
    from app.models.requests import LLMConfig

    return DeepIdentifyRequest(
        job_id=1,
        llm=LLMConfig(provider="anthropic", model="claude-3-5-sonnet-20241022", api_key="test-key"),
        images=[
            DeepIdentifyImage(role="obverse", data_uri=_tiny_data_uri()),
            DeepIdentifyImage(role="reverse", data_uri=_tiny_data_uri()),
        ],
        notes="",
        provider_catalog=[
            DeepProviderCatalogEntry(provider="numista", automatable=True),
            DeepProviderCatalogEntry(provider="ngc", automatable=False, link_out="https://www.ngccoin.com/verify/"),
            DeepProviderCatalogEntry(provider="ocre", automatable=False, reason="pending_license_validation"),
            DeepProviderCatalogEntry(provider="rpc", automatable=False, reason="no_public_api"),
        ],
        bounds=DeepIdentifyBounds(
            max_providers=2,
            max_concurrency=2,
            provider_timeout_s=5,
            total_timeout_s=total_timeout_s,
            recursion_limit=10,
        ),
        tools_base_url="http://test-api:8080",
        internal_token="test-token-12345",
    )


async def _collect_frames(request):
    frames = []
    async for chunk in graph_module.run_deep_identification_stream(request):
        assert chunk.startswith("data: ")
        payload = json.loads(chunk[len("data: "):].strip())
        frames.append(payload)
    return frames


@pytest.mark.asyncio
async def test_happy_path_frame_sequence_and_terminal_invariant(monkeypatch):
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def fake_numista_run(entry, tools, quick_evidence, notes):
        return ProviderEvidence(provider="numista", status="no_match", automatable=True, call_count=1)

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", fake_numista_run)

    frames = await _collect_frames(_request())

    types = [f["type"] for f in frames]
    assert types[0] == "progress"
    assert "router_selected" in types
    assert "provider_started" in types
    assert "provider_result" in types
    assert "evaluation" in types
    assert "synthesis_started" in types
    assert types[-1] == "synthesis"

    # Terminal frame invariant: exactly one terminal frame (synthesis/error), at the end.
    terminal_frames = [f for f in frames if f["type"] in ("synthesis", "error")]
    assert len(terminal_frames) == 1
    assert terminal_frames[0] is frames[-1]

    report = frames[-1]["report"]
    DeepSynthesis.model_validate(report)  # must validate against the typed contract


@pytest.mark.asyncio
async def test_no_token_shaped_strings_leak_in_any_frame(monkeypatch):
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def leaky_numista_run(entry, tools, quick_evidence, notes):
        # Simulate a buggy upstream echoing an Authorization header inside a
        # claim value — must be redacted before it ever reaches an SSE frame.
        from app.models.responses import ProviderClaim

        return ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            confidence=0.5,
            call_count=1,
            claims=[
                ProviderClaim(
                    field="issuer",
                    value="Bearer abcdefghij.abcdefghij0123.abcdefghij0123",
                    confidence=0.5,
                    citation="https://en.numista.com/catalogue/1",
                )
            ],
        )

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", leaky_numista_run)

    request = _request()
    raw_chunks = []
    async for chunk in graph_module.run_deep_identification_stream(request):
        raw_chunks.append(chunk)
    full_text = "".join(raw_chunks)

    assert "Bearer abcdefghij" not in full_text
    assert "[REDACTED_INTERNAL_TOKEN]" in full_text


@pytest.mark.asyncio
async def test_timeout_with_partial_evidence_falls_back_to_partial_synthesis(monkeypatch):
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def slow_numista_run(entry, tools, quick_evidence, notes):
        await asyncio.sleep(30)
        return ProviderEvidence(provider="numista", status="no_match", automatable=True)

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", slow_numista_run)

    # Non-automatable providers (ngc/ocre/rpc) complete instantly, giving the
    # timeout branch *some* evidence to synthesize from even though numista hangs.
    request = _request(total_timeout_s=1)
    request.bounds.provider_timeout_s = 60  # provider-level timeout must not fire first

    frames = await _collect_frames(request)

    assert frames[-1]["type"] == "synthesis"
    assert frames[-1]["report"]["partial_success"] is True


@pytest.mark.asyncio
async def test_timeout_with_zero_evidence_emits_typed_error(monkeypatch):
    class HangingModel:
        async def ainvoke(self, messages, **kwargs):
            await asyncio.sleep(30)

    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: HangingModel())

    request = _request(total_timeout_s=1)
    request.provider_catalog = [
        entry for entry in request.provider_catalog if entry.provider == "numista"
    ]
    frames = await _collect_frames(request)

    assert frames[-1]["type"] == "error"
    assert frames[-1]["error_kind"] == "timeout"
