"""Bounds-clamp tests (T077) — a caller must never exceed the deployment's
configured `AGENT_DEEP_*` ceilings regardless of what `request.bounds`
claims (contracts/agent-internal-contract.md §2/§6).
"""

import app.teams.deep_identification.graph as graph_module
from app.models.requests import DeepIdentifyBounds


def test_over_ceiling_request_is_clamped_down(monkeypatch):
    monkeypatch.setattr(graph_module.settings, "deep_max_providers", 4)
    monkeypatch.setattr(graph_module.settings, "deep_max_concurrency", 2)
    monkeypatch.setattr(graph_module.settings, "deep_provider_timeout", 45)
    monkeypatch.setattr(graph_module.settings, "deep_total_timeout", 280)
    monkeypatch.setattr(graph_module.settings, "deep_recursion_limit", 12)

    requested = DeepIdentifyBounds(
        max_providers=10,
        max_concurrency=10,
        provider_timeout_s=120,
        total_timeout_s=900,
        recursion_limit=50,
    )

    clamped = graph_module._clamp_bounds_to_ceilings(requested)

    assert clamped.max_providers == 4
    assert clamped.max_concurrency == 2
    assert clamped.provider_timeout_s == 45
    assert clamped.total_timeout_s == 280
    assert clamped.recursion_limit == 12


def test_under_ceiling_request_is_left_alone(monkeypatch):
    monkeypatch.setattr(graph_module.settings, "deep_max_providers", 4)
    monkeypatch.setattr(graph_module.settings, "deep_max_concurrency", 2)
    monkeypatch.setattr(graph_module.settings, "deep_provider_timeout", 45)
    monkeypatch.setattr(graph_module.settings, "deep_total_timeout", 280)
    monkeypatch.setattr(graph_module.settings, "deep_recursion_limit", 12)

    requested = DeepIdentifyBounds(
        max_providers=2,
        max_concurrency=1,
        provider_timeout_s=5,
        total_timeout_s=30,
        recursion_limit=8,
    )

    clamped = graph_module._clamp_bounds_to_ceilings(requested)

    assert clamped.max_providers == 2
    assert clamped.max_concurrency == 1
    assert clamped.provider_timeout_s == 5
    assert clamped.total_timeout_s == 30
    assert clamped.recursion_limit == 8
