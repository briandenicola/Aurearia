"""T107 follow-up (Cassius): keep the Go contract-drift fixture honest.

`src/api/services/deep_identification_contract_drift_test.go` compares Go's
mirror structs against a checked-in JSON Schema fixture,
`src/api/services/testdata/deep_identify_contract_schema.json`, generated
once from ``DeepIdentifyRequest.model_json_schema()`` /
``DeepSynthesis.model_json_schema()``. That Go test is hermetic (no live
Python needed) but, by construction, can only ever compare Go against
*whatever the fixture says* -- if a Pydantic model changes and nobody
regenerates the fixture, the Go test keeps passing against a stale
snapshot. T106 (the live Go<->Python SSE round trip) is excluded from the
default CI path, so it cannot be relied on to catch that staleness on every
run.

This test closes that gap from the Python side: it recomputes the same two
schemas the fixture was generated from, right now, from the real Pydantic
models, and asserts they are byte-for-byte identical (modulo the
``_generated_by`` envelope key) to what's checked in. It needs no Go
toolchain and no live service, so it runs on every ``pytest`` invocation --
Python can no longer drift away from the fixture unnoticed.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from app.models.requests import DeepIdentifyRequest
from app.models.responses import DeepSynthesis

# Mirrors the regeneration command documented atop
# src/api/services/deep_identification_contract_drift_test.go -- keep both
# in sync if the fixture's shape/location ever changes.
REGENERATE_COMMAND = (
    "python -c \"import json; from app.models.requests import "
    "DeepIdentifyRequest; from app.models.responses import DeepSynthesis; "
    "json.dump({'_generated_by': 'src/agent (python -m "
    "app.models.requests/responses .model_json_schema()) - see "
    "src/api/services/deep_identification_contract_drift_test.go header for "
    "regeneration command', 'DeepIdentifyRequest': "
    "DeepIdentifyRequest.model_json_schema(), 'DeepSynthesis': "
    "DeepSynthesis.model_json_schema()}, "
    "open('../api/services/testdata/deep_identify_contract_schema.json', "
    "'w'), indent=2, sort_keys=True)\" (run from src/agent/)"
)


def _fixture_path() -> Path | None:
    """Resolve the checked-in fixture relative to this test file, not the
    current working directory (pytest may be invoked from src/agent/,
    the repo root, or elsewhere). Returns None if the expected repo layout
    (src/agent/tests/ next to src/api/services/testdata/) isn't found, so
    callers can skip rather than produce a false red in an unexpected
    packaging/checkout context.
    """
    # this file: src/agent/tests/test_contract_schema_fixture_is_current.py
    # -> parents[0]=tests, [1]=agent, [2]=src, [3]=repo root
    repo_root = Path(__file__).resolve().parents[3]
    candidate = repo_root / "src" / "api" / "services" / "testdata" / "deep_identify_contract_schema.json"
    return candidate if candidate.is_file() else None


def test_contract_schema_fixture_is_current() -> None:
    """Recomputes DeepIdentifyRequest/DeepSynthesis's own JSON Schema right
    now and asserts it matches the checked-in Go-side fixture exactly, so
    the fixture can never silently go stale behind a Pydantic model change.
    """
    fixture_path = _fixture_path()
    if fixture_path is None:
        pytest.skip(
            "could not resolve src/api/services/testdata/"
            "deep_identify_contract_schema.json relative to this test file "
            "(unexpected repo layout/packaged checkout) - skipping rather "
            "than risk a false red"
        )

    on_disk = json.loads(fixture_path.read_text(encoding="utf-8"))

    # Recompute the same two schemas the fixture is generated from, via the
    # exact same call the regeneration command uses.
    recomputed = {
        "DeepIdentifyRequest": DeepIdentifyRequest.model_json_schema(),
        "DeepSynthesis": DeepSynthesis.model_json_schema(),
    }

    # Compare only the two model schemas; the "_generated_by" envelope key
    # is provenance metadata, not part of the contract, so it's excluded
    # from both sides before comparing (avoids a spurious failure if that
    # string's wording ever changes independent of the schemas themselves).
    on_disk_models = {k: v for k, v in on_disk.items() if k != "_generated_by"}

    # json.dump(..., sort_keys=True) at generation time already normalizes
    # key order; re-serializing both sides through the same sort_keys=True
    # round trip makes the comparison apples-to-apples regardless of dict
    # insertion order on either side.
    on_disk_normalized = json.dumps(on_disk_models, sort_keys=True)
    recomputed_normalized = json.dumps(recomputed, sort_keys=True)

    assert on_disk_normalized == recomputed_normalized, (
        "src/api/services/testdata/deep_identify_contract_schema.json is "
        "stale - DeepIdentifyRequest/DeepSynthesis's Pydantic schema no "
        "longer matches the checked-in fixture the Go contract-drift test "
        "(deep_identification_contract_drift_test.go) compares against. "
        f"Regenerate it with:\n{REGENERATE_COMMAND}"
    )
